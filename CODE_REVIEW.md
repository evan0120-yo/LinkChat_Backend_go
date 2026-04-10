# LinkChat CODE REVIEW

## 範圍

- 本文件只描述 LinkChat\Backend\Go\LinkChat 目前實際落地的 Go backend。
- 來源優先順序是 current code -> runtime wiring -> 既有文件。
- 這份文件是 current implementation walkthrough，不是 BDD / SDD / PLAN。
- LinkChat\Backend\Java\LinkChat 不在這份 CODE_REVIEW 範圍內。

## BLOCK 1: AI 對產品的想像

從現在這份 code 看，LinkChat 已經不是只有 auth 與好友關係的底座了。它現在更像一個「先把人物背景資料整理好、之後再接 AI」的 Go backend。

目前已經落地的有三塊：

- 帳號與 JWT
- 搜尋與好友關係
- 人物背景資料

第三塊的人物背景資料，就是這次新長出來的 profile 模組。它讓使用者可以對某個已連結的人保存補充三句、tag 選擇、查詢 tag catalog，並把這些資料整理成之後 AI 分析會需要的 context。

如果用一句話總結現在的產品形狀，大概會是這樣：

```text
現在是 auth + link + profile 的 Go backend
│
├─ 已落地
│  ├─ 帳號 / JWT
│  ├─ 搜尋與好友關係
│  └─ 人物背景資料（notes / tags / catalog / context）
│
└─ 未落地
   ├─ copilot integration
   ├─ InternalAICopliot client
   └─ AI 分析入口
```

它現在不是什麼：

- 不是 production-ready 服務
- 不是已經接好 AI 分析入口的產品
- 不是完整社交平台
- 不是擁有完整自動化測試覆蓋的成熟後端

profile 落地後，這個專案的定位比以前清楚很多。以前它比較像只有「人怎麼進來、彼此怎麼連上」的 backend playground；現在它已經開始保存 AI 會需要的人物背景資料，只是還沒把 LinkChat -> InternalAICopliot 這條線真的接上。

## BLOCK 2: 讀者模式

### 1. 這個系統現在實際是什麼

它現在實際上是一個 Gin + Firestore emulator backend，核心模組是 auth、link、profile 三塊。

你可以把它理解成：先把「登入」、「聯絡人關係」、「人物背景資料」都固定下來，等這些資料邊界穩定後，再往 AI 分析入口延伸。整體 runtime 仍然很偏開發驗證用途，不是正式環境。

```text
啟動 main
│
├─ 連 Firestore emulator
├─ 清 users / link_users / links
├─ 清 subject_profiles / profile_tag_groups / profile_tags
├─ 組 Link module
├─ 組 Profile module
├─ 組 Auth module
├─ 註冊 /citrus 路由
├─ 跑 auth seed
├─ 跑 link seed
├─ 跑 profile seed
└─ listen :8082
```

> 注意: 每次啟動都會清空 6 個集合，所以預設心智模型仍然是「開發時重跑流程」，不是保留既有資料。

> 注意: 系統現在已經不只 auth/link，profile 也在 runtime 裡了；但 AI 呼叫、copilot integration、InternalAICopliot client 仍未接上。

### 2. 帳號與登入

這塊做的事情和之前差不多，仍然是註冊、登入、刪除帳號，以及幾條測試路由。

註冊時，系統不只會建一份 users 文件，還會順手幫 link 模組建立 link_users projection。這代表 auth 仍然是帳號真實來源，而 link_users 只是給搜尋與關係資料用的 projection。

```text
註冊
│
├─ 先看 email 有沒有重複
├─ bcrypt hash 密碼
├─ 產 uuid v7
└─ transaction
   ├─ 建 users
   └─ 建 link_users
```

登入流程也維持很單純：查 email、驗密碼、簽 token。沒有 refresh token、沒有 session、沒有 blacklist。

刪除帳號仍然不是 hard delete，而是把 users.is_active 與 link_users.is_active 改成 false。links 不會被清，所以舊關係資料仍會留在系統裡。

```text
刪除帳號
│
├─ middleware 先解 token
├─ usecase 判斷是不是自己，或是不是 admin
├─ 找 target user
└─ transaction
   ├─ users.is_active = false
   └─ link_users.is_active = false
```

> 注意: 註冊時 role 一律寫成 user，所以 seed 裡雖然有一個叫 System Admin 的帳號，實際上也只是 user。

> 注意: login 現在沒有檢查 users.is_active，所以被停用的帳號只要密碼對，還是能登入拿 token。

> 注意: /citrus/auth/delete 的權限錯誤最後仍會被 handler 回成 500，不是比較合理的 403。

### 3. 搜尋與好友關係

link 模組現在仍是簡化版關係系統。它負責搜尋人、送出申請、接受、拒絕、取消、解除這些基礎動作。

搜尋是 prefix search，只搜 active 的 link_users，最多回 20 筆。好友申請的核心邏輯也沒變：不能加自己、不能加不存在或 inactive 的人、兩人之間不能已經有任何 link 文件。通過之後才會寫一筆 pending link。

```text
ApplyLink
│
├─ requester / target 不可空
├─ requester != target
├─ transaction
│  ├─ 先檢查兩人之間有沒有既有 link
│  ├─ 再檢查 target 是否存在且 active
│  └─ 建一筆 pending
└─ 回傳新 link
```

接受、拒絕、取消、解除的規則也一樣切得很硬：

```text
pending
├─ target accept  -> active
├─ target reject  -> rejected
└─ requester cancel -> hard delete

active
└─ 任一 participant remove -> hard delete
```

這次對 link 模組最重要的新變化，不是多了新關係 API，而是多了一個給 profile 模組用的查詢入口。profile 不會直接去碰 link repository 或 link service，它是透過 link usecase/query 驗證「這個 subject 是不是目前 active 的 linked user」。

用產品語言講，意思就是：

- 人物背景資料不是隨便對任何人都能寫
- 你只能對目前真的已連結、而且還活著的對象寫這些資料

> 注意: 只要兩人之間已有任意 link 文件，包含 rejected，後續都不能再 Apply。

> 注意: blocked 狀態雖然存在於 model，但目前沒有任何 API 會把 link 寫成 blocked。

### 4. 人物補充三句與 tag

這是這次實作新增的主體。profile 模組現在已經能保存「某個 owner 對某個 linked subject 的人物背景資料」。

對外有兩條寫入 API：

- PUT /citrus/profiles/notes
- PUT /citrus/profiles/tags

這兩條 API 的 request 都會帶 subjectId，但邏輯邊界不同：notes 專門處理補充三句，tags 專門處理 tag 選擇。兩邊分開保存，但最後都會 merge 回同一份 subject_profiles 文件。

#### 補充三句

notes 的邏輯很明確：

- body 帶 subjectId、lines
- 每條會先 trim
- trim 後空字串直接移除
- 最多只能留 3 條非空短句
- 每條上限 60 字
- 保留原本順序
- 不會自動改寫、摘要、翻譯或合併

```text
保存 notes
│
├─ 驗證 owner 與 subject 是否可編輯
├─ trim / 移除空字串
├─ 檢查最多 3 條
├─ 檢查每條最多 60 字
├─ 讀既有 subject profile
├─ 保留既有 tags
└─ upsert 或 delete
```

#### tag

tags 的邏輯也很結構化：

- body 帶 subjectId、selected
- tag 走 canonical groupKey/tagKey
- 會做 trim + lowercase normalization
- 不存在或 inactive 的 group/tag 會直接失敗
- single group 只能選一個
- multi group 可以多個
- 重複 tag 只會留一份

```text
保存 tags
│
├─ 驗證 owner 與 subject 是否可編輯
├─ 讀 active tag catalog
├─ 檢查 group / tag 是否存在
├─ 檢查 single / multi 規則
├─ dedupe 重複 tag
├─ 讀既有 subject profile
├─ 保留既有 notes
└─ upsert 或 delete
```

#### merge 與 delete 行為

這兩條 API 有一個很重要的資料行為：它們雖然分開保存，但不是各存一份資料。

- 保存 notes 時，既有 tags 要保留
- 保存 tags 時，既有 notes 要保留
- 如果 notes + tags 最後都空，系統會直接刪掉 subject_profiles 文件

#### access rule

這塊的限制非常明確：

- owner 不能把自己當成 subject
- subjectId 必須是 active linked user

也就是說，profile 模組保存的不是任意人物資料，而是「目前這個使用者已連結對象的人物背景資料」。

> 注意: 這裡的 tag label 只存在 catalog 與 response，不會被寫進 subject_profiles 當成主要資料。

> 注意: 這些資料目前只是本地保存與查詢，還沒有被真正送進 InternalAICopliot 的 AI consult flow。

### 5. 人物背景資料與 tag catalog 查詢

除了寫入之外，profile 模組現在也能把資料查回來。

對外有兩條查詢 API：

- GET /citrus/profiles/context?subjectId=...
- GET /citrus/profiles/tag-catalog

#### context 查詢

這條 API 會先做和寫入一樣的 access rule 檢查，也就是 subject 必須是 active linked user。通過後，系統會用 ownerID__subjectID 去找 subject_profiles。

如果找不到文件，不會當成錯誤，而是回一個空資料結構。這代表前端可以直接拿來畫初始狀態，不需要把「還沒建立人物資料」當 exception flow。

```text
查 profile context
│
├─ 驗證 owner 與 subject 是否可讀
├─ 用 ownerID__subjectID 查 subject_profiles
└─ 找不到就回空結構
```

#### tag catalog 查詢

這條 API 回的是目前 active 的 group 與 active 的 tags，而且已經按 group 組好了。它的角色比較像 UI catalog，不是 prompt source。

```text
查 tag catalog
│
├─ 讀 active groups
├─ 讀 active tags
├─ 依 group 組裝
└─ 回傳給前端選擇
```

前端應把這份資料理解成：

- 這是「現在可以選哪些 tag」的 UI 資料
- 不是 Internal prompt fragment
- 不是 AI 理論 mapping

> 注意: /context 沒文件時是 200 + 空資料，不是 404。

> 注意: /tag-catalog 只回 active groups / active tags，已 inactive 的資料不會出現在前端選單。

### 6. 測試資料與成熟度

這個專案現在的驗證方式仍然偏手動，但已經不再是「完全沒有 test file」的狀態。

目前它有三種驗證手段：

- auth seeder
- link seeder
- profile seeder

外加：

- /citrus/test/ping
- /citrus/test/profile
- /citrus/test/system
- profile 模組的 unit tests

auth seeder 仍然透過正式的 Register usecase 建帳號；link seeder 會幫 Normal User 與 EvanHe 建一筆 pending 關係；profile seeder 則會植入 persona tag catalog。

```text
AuthSeeder
├─ 建 System Admin
├─ 建 Normal User
└─ 建 EvanHe

LinkSeeder
├─ 用名字找 Normal User
├─ 用名字找 EvanHe
└─ ApplyLink: Normal User -> EvanHe

ProfileSeeder
├─ 植入 profile_tag_groups
└─ 植入 profile_tags
```

自動化測試目前只補在 profile 模組，覆蓋的是 deterministic rule 與 command usecase 的核心合併行為，不是整個系統都已有完整回歸保護。

> 注意: 專案大部分模組仍然偏手動驗證；目前有單元測試的主要是 profile validator 與 profile command usecase。

> 注意: 整個專案的預設心智模型仍然是 emulator-first 開發，而不是正式環境 rollout。

## BLOCK 3: 技術補充

### 1. 這個系統現在實際是什麼

關鍵檔案

- cmd/api/main.go (line 39)
- internal/link/provider.go (line 13)
- internal/profile/provider.go (line 17)
- internal/auth/provider.go (line 17)

啟動與 wiring：

```text
cmd/api/main.go
│
├─ clearCollection("users")              line 60
├─ clearCollection("link_users")         line 61
├─ clearCollection("links")              line 62
├─ clearCollection("subject_profiles")   line 63
├─ clearCollection("profile_tag_groups") line 64
├─ clearCollection("profile_tags")       line 65
│
├─ link.NewLinkModule(client)                                     line 73
├─ profile.NewProfileModule(client, linkModule.LinkQueryUseCase)  line 74
├─ auth.NewAuthModule(client, linkModule.LinkUserCommandUseCase)  line 78
│
├─ authHandler.RegisterRoutes(rootGroup, authMiddleware)          line 93
├─ testHandler.RegisterRoutes(rootGroup, authMiddleware)          line 94
├─ linkModule.Handler.RegisterRoutes(rootGroup, authMiddleware)   line 98
├─ profileModule.Handler.RegisterRoutes(rootGroup, authMiddleware) line 99
│
├─ authSeeder.Seed(ctx)                                           line 104
├─ linkModule.Seeder.Seed(ctx)                                    line 109
├─ profileModule.Seeder.Seed(ctx)                                 line 113
└─ r.Run(":8082")                                                 line 118
```

實際對外路由總表：

| Method | Path | 來源 |
| --- | --- | --- |
| GET | /citrus/health | cmd/api/main.go |
| POST | /citrus/auth/register | internal/auth/handler/auth_handler.go |
| POST | /citrus/auth/login | internal/auth/handler/auth_handler.go |
| POST | /citrus/auth/delete | internal/auth/handler/auth_handler.go |
| POST | /citrus/test/ping | internal/auth/handler/test_handler.go |
| POST | /citrus/test/profile | internal/auth/handler/test_handler.go |
| POST | /citrus/test/system | internal/auth/handler/test_handler.go |
| POST | /citrus/links/search | internal/link/handler/link_handler.go |
| POST | /citrus/links/apply | internal/link/handler/link_handler.go |
| POST | /citrus/links/accept | internal/link/handler/link_handler.go |
| POST | /citrus/links/reject | internal/link/handler/link_handler.go |
| POST | /citrus/links/remove | internal/link/handler/link_handler.go |
| POST | /citrus/links/cancel | internal/link/handler/link_handler.go |
| GET | /citrus/links/list | internal/link/handler/link_handler.go |
| PUT | /citrus/profiles/notes | internal/profile/handler/profile_handler.go |
| PUT | /citrus/profiles/tags | internal/profile/handler/profile_handler.go |
| GET | /citrus/profiles/context | internal/profile/handler/profile_handler.go |
| GET | /citrus/profiles/tag-catalog | internal/profile/handler/profile_handler.go |

目前已接上的模組：

- auth
- link
- profile

目前未接上的模組：

- copilot integration
- InternalAICopliot client
- AI request contract

### 2. 帳號與登入

關鍵檔案

- internal/auth/handler/auth_handler.go (line 61)
- internal/auth/usecase/command/auth_usecase.go (line 52)
- internal/auth/usecase/query/auth_usecase.go (line 31)
- internal/auth/service/query/auth_service.go (line 53)
- internal/auth/service/validator/auth_validator.go (line 27)
- internal/auth/repository/user_repository.go (line 77)
- internal/link/usecase/command/link_user_usecase.go (line 35)
- internal/auth/middleware/auth_middleware.go (line 36)
- internal/auth/handler/test_handler.go (line 21)

註冊 call chain：

```text
POST /citrus/auth/register
-> AuthHandler.Register
-> AuthCommandUseCase.Register
   -> AuthValidator.ValidateEmailUnique
      -> AuthQueryService.FindByEmail
         -> UserRepository.FindByEmail
   -> AuthCommandService.HashPassword
   -> Firestore RunTransaction
      -> AuthCommandService.WithTx(...).CreateUser
         -> UserRepository.CreateUser
      -> LinkUserCommandUseCase.SyncUser
         -> LinkUserCommandService.WithTx(...).CreateLinkUser
            -> LinkUserRepository.CreateLinkUser
```

登入 call chain：

```text
POST /citrus/auth/login
-> AuthHandler.Login
-> AuthQueryUseCase.Login
   -> AuthQueryService.FindByEmail
      -> UserRepository.FindByEmail
   -> AuthQueryService.VerifyPassword
   -> AuthQueryService.GenerateToken
```

刪除帳號 call chain：

```text
POST /citrus/auth/delete
-> VerifyToken middleware
-> AuthHandler.DeleteUser
-> AuthCommandUseCase.DeleteUser
   -> AuthQueryService.FindUserByID
      -> UserRepository.FindByID
   -> Firestore RunTransaction
      -> AuthCommandService.WithTx(...).UpdateUser
         -> UserRepository.UpdateUser
      -> LinkUserCommandUseCase.DeleteLinkUser
         -> LinkUserQueryService.GetLinkUserByID
            -> LinkUserRepository.FindLinkUserByID
         -> LinkUserCommandService.WithTx(...).UpdateLinkUser
            -> LinkUserRepository.UpdateLinkUser
```

JWT 技術細節：

| 項目 | 目前實作 |
| --- | --- |
| 演算法 | HS256 |
| secret | YOUR_SUPER_SECRET_KEY |
| exp | 24 小時 |
| claims | sub、name、role、exp、iss |
| middleware context key | userID、userRole |

Auth / Test API 錯誤映射：

| API | 條件 | HTTP status |
| --- | --- | --- |
| POST /auth/register | JSON binding 失敗 | 400 |
| POST /auth/register | email 重複或 transaction 失敗 | 400 |
| POST /auth/login | JSON binding 失敗 | 400 |
| POST /auth/login | 查無帳號或密碼錯誤 | 401 |
| POST /auth/delete | JSON binding 失敗 | 400 |
| POST /auth/delete | token 無效或缺失 | 401 |
| POST /auth/delete | usecase 任意錯誤，包含 permission denied | 500 |
| POST /test/ping | 永遠成功 | 200 |
| POST /test/profile | token 無效或缺失 | 401 |
| POST /test/system | 非 admin | 403 |

目前實作限制：

- RegisterReq 的 HTTP password 最小長度是 6，但 seeder 直接呼叫 usecase，不會套用這層驗證。
- Register 固定把 role 設成 user，因此 seed 內的 Role 欄位沒有實際效果。
- Login 沒檢查 users.is_active。

### 3. 搜尋與好友關係

關鍵檔案

- internal/link/handler/link_handler.go (line 29)
- internal/link/usecase/command/link_usecase.go (line 27)
- internal/link/usecase/query/link_usecase.go (line 14)
- internal/link/service/command/link_service.go (line 18)
- internal/link/service/validator/link_validator.go (line 9)
- internal/link/repository/link_repository.go (line 11)
- internal/link/repository/link_user_repository.go (line 11)

SearchUsers call chain：

```text
POST /citrus/links/search
-> VerifyToken middleware
-> LinkHandler.SearchUsers
-> LinkQueryUseCase.SearchUsers
-> LinkUserQueryService.SearchUsers
-> LinkUserRepository.SearchByDisplayName
```

Apply / Accept / Reject / Cancel / Remove call chain：

```text
POST /citrus/links/apply
-> LinkHandler.ApplyLink
-> LinkCommandUseCase.ApplyLink
   -> LinkValidator.ValidateCreateLink
   -> Firestore RunTransaction
      -> LinkQueryService.GetLinkByParticipants
         -> LinkRepository.FindLinkByParticipants
      -> LinkUserQueryService.GetLinkUserByID
         -> LinkUserRepository.FindLinkUserByID
      -> LinkCommandService.WithTx(...).CreateLink
         -> LinkRepository.CreateLink

POST /citrus/links/accept
-> LinkHandler.AcceptLink
-> LinkCommandUseCase.AcceptLink
   -> LinkQueryService.GetLinkByID
      -> LinkRepository.FindLinkByID
   -> txCmd.UpdateLink

POST /citrus/links/reject
-> LinkHandler.RejectLink
-> LinkCommandUseCase.RejectLink
   -> LinkQueryService.GetLinkByID
   -> LinkCommandService.RejectLink
      -> LinkRepository.UpdateLink

POST /citrus/links/cancel
-> LinkHandler.CancelLink
-> LinkCommandUseCase.CancelLink
   -> LinkQueryService.GetLinkByID
   -> LinkCommandService.CancelLink
      -> LinkRepository.DeleteLink

POST /citrus/links/remove
-> LinkHandler.RemoveLink
-> LinkCommandUseCase.RemoveLink
   -> LinkQueryService.GetLinkByID
   -> LinkCommandService.RemoveLink
      -> LinkRepository.DeleteLink
```

關係操作規則表：

| API | 允許操作者 | 允許狀態 | 成功結果 |
| --- | --- | --- | --- |
| apply | requester | 無既有 link、target active | 建 pending |
| accept | target | pending | 改 active |
| reject | target | pending | 改 rejected |
| cancel | requester | pending | hard delete |
| remove | 任一 participant | active | hard delete |

Link query 補充：

```text
LinkQueryUseCase.GetLinkedSubject
│
├─ owner / subject 不可空，且不可相同
├─ 用 GetLinkByParticipants 檢查兩人是否有 active link
├─ 用 GetLinkUserByID 檢查 subject 是否 active
└─ 成功才回 subject
```

這條 query 是 profile 模組 access rule 的入口，不是直接讓 profile 去碰 link repository。

關係 API 錯誤映射：

| API | 條件 | HTTP status |
| --- | --- | --- |
| POST /links/search | JSON binding 失敗 | 400 |
| POST /links/search | query service error | 500 |
| POST /links/apply | token 無效或缺失 | 401 |
| POST /links/apply | target 不存在、重複申請、自加自己 | 400 |
| POST /links/accept | token 無效或缺失 | 401 |
| POST /links/accept | 非 target、狀態錯誤、link 不存在 | 400 |
| POST /links/reject | token 無效或缺失 | 401 |
| POST /links/reject | 非 target、狀態錯誤、link 不存在 | 400 |
| POST /links/cancel | token 無效或缺失 | 401 |
| POST /links/cancel | 非 requester、狀態錯誤、link 不存在 | 400 |
| POST /links/remove | token 無效或缺失 | 401 |
| POST /links/remove | 非 participant、狀態錯誤、link 不存在 | 400 |
| GET /links/list | token 無效或缺失 | 401 |
| GET /links/list | query usecase error | 500 |

### 4. 人物補充三句與 tag

關鍵檔案

- internal/profile/handler/profile_handler.go (line 29)
- internal/profile/usecase/common.go (line 13)
- internal/profile/usecase/command/profile_usecase.go (line 31)
- internal/profile/service/validator/profile_validator.go (line 17)
- internal/profile/repository/subject_profile_repository.go (line 13)
- internal/profile/repository/tag_catalog_repository.go (line 13)

Profile module wiring：

```text
internal/profile/provider.go
│
├─ NewSubjectProfileRepository(client)       line 26
├─ NewTagCatalogRepository(client)           line 27
├─ NewSubjectProfileCommandService(...)      line 29
├─ NewSubjectProfileQueryService(...)        line 30
├─ NewTagCatalogQueryService(...)            line 31
├─ NewProfileValidator()                     line 32
├─ NewProfileCommandUseCase(...)             line 34
└─ NewProfileQueryUseCase(...)               line 41
```

保存 notes call chain：

```text
PUT /citrus/profiles/notes
-> VerifyToken middleware
-> ProfileHandler.SaveSubjectNotes
-> ProfileCommandUseCase.SaveSubjectNotes
   -> EnsureAccessibleSubject
      -> LinkQueryUseCase.GetLinkedSubject
   -> ProfileValidator.NormalizeNoteLines
   -> Firestore RunTransaction
      -> SubjectProfileQueryService.WithTx(...).GetSubjectProfileByID
      -> SubjectProfileCommandService.WithTx(...).SaveSubjectProfile / DeleteSubjectProfile
```

保存 tags call chain：

```text
PUT /citrus/profiles/tags
-> VerifyToken middleware
-> ProfileHandler.SaveSubjectTags
-> ProfileCommandUseCase.SaveSubjectTags
   -> EnsureAccessibleSubject
      -> LinkQueryUseCase.GetLinkedSubject
   -> TagCatalogQueryService.GetActiveTagCatalog
   -> ProfileValidator.ValidateAndNormalizeSelectedTags
   -> Firestore RunTransaction
      -> SubjectProfileQueryService.WithTx(...).GetSubjectProfileByID
      -> SubjectProfileCommandService.WithTx(...).SaveSubjectProfile / DeleteSubjectProfile
```

access rule：

```text
EnsureAccessibleSubject
│
├─ ownerID 不可空                line 19
├─ subjectId 不可空              line 22
├─ ownerID != subjectId          line 25
├─ GetLinkedSubject(...)         line 29
└─ subject == nil -> 回錯        line 33
```

notes 驗證規則：

```text
NormalizeNoteLines
│
├─ trim 每條                     line 32
├─ 空字串直接移除                line 33
├─ 單條 > 60 字 -> 回錯          line 37
└─ 非空短句 > 3 條 -> 回錯       line 44
```

tags 驗證規則：

```text
ValidateAndNormalizeSelectedTags
│
├─ normalize groupKey/tagKey     line 71
├─ groupKey / tagKey 不可空      line 74
├─ group 必須存在                line 78
├─ tag 必須存在且 active         line 83
├─ dedupe 相同 group__tag        line 88
└─ single group 只能選一個       line 92
```

資料 merge / delete 規則：

```text
SaveSubjectNotes
│
├─ RunTransaction                line 127
├─ transaction 內讀既有 profile  line 58
├─ 保留既有 selectedTags         line 63
├─ notes + tags 都空 -> delete   line 68
└─ 否則 build + save             line 75

SaveSubjectTags
│
├─ RunTransaction                line 127
├─ transaction 內讀既有 profile  line 98
├─ 保留既有 noteLines            line 103
├─ notes + tags 都空 -> delete   line 108
└─ 否則 build + save             line 115
```

Profile API 錯誤映射：

| API | 條件 | HTTP status |
| --- | --- | --- |
| PUT /profiles/notes | JSON binding 失敗 | 400 |
| PUT /profiles/notes | owner 缺失 | 401 |
| PUT /profiles/notes | subject 非 active linked user | 403 |
| PUT /profiles/notes | 超過 3 條或單條超長 | 400 |
| PUT /profiles/notes | Firestore / service unexpected error | 500 |
| PUT /profiles/tags | JSON binding 失敗 | 400 |
| PUT /profiles/tags | owner 缺失 | 401 |
| PUT /profiles/tags | subject 非 active linked user | 403 |
| PUT /profiles/tags | group / tag 不存在或 inactive | 400 |
| PUT /profiles/tags | single-select 衝突 | 400 |
| PUT /profiles/tags | Firestore / service unexpected error | 500 |

### 5. 人物背景資料與 tag catalog 查詢

關鍵檔案

- internal/profile/handler/profile_handler.go (line 84)
- internal/profile/usecase/query/profile_usecase.go (line 23)
- internal/profile/usecase/common.go (line 65)
- internal/profile/repository/tag_catalog_repository.go (line 44)

查 profile context call chain：

```text
GET /citrus/profiles/context
-> VerifyToken middleware
-> ProfileHandler.GetSubjectProfileContext
-> ProfileQueryUseCase.GetSubjectProfileContext
   -> EnsureAccessibleSubject
      -> LinkQueryUseCase.GetLinkedSubject
   -> SubjectProfileQueryService.GetSubjectProfileByID
   -> ToSubjectProfileResponse
```

查 tag catalog call chain：

```text
GET /citrus/profiles/tag-catalog
-> VerifyToken middleware
-> ProfileHandler.GetTagCatalog
-> ProfileQueryUseCase.GetTagCatalog
   -> TagCatalogQueryService.GetActiveTagCatalog
      -> TagCatalogRepository.ListActiveGroups
      -> TagCatalogRepository.ListActiveTags
```

response shaping 補充：

```text
ToSubjectProfileResponse
│
├─ profile == nil -> 回空 noteLines / selectedTags   line 72
├─ 有 notes 就照原順序 append                       line 76
├─ 有 tags 就轉成 response item                     line 80
└─ updatedAt 只有 profile 存在時才會帶             line 90
```

catalog repository 補充：

```text
ListActiveGroups
├─ 只查 active=true                                  line 45
└─ 依 orderNo / groupKey 排序                        line 65

ListActiveTags
├─ 只查 active=true                                  line 76
└─ 依 groupKey / orderNo / tagKey 排序              line 96
```

Profile query API 錯誤映射：

| API | 條件 | HTTP status |
| --- | --- | --- |
| GET /profiles/context | owner 缺失 | 401 |
| GET /profiles/context | subjectId 空白 | 400 |
| GET /profiles/context | subject 非 active linked user | 403 |
| GET /profiles/context | 沒有 profile 文件 | 200 |
| GET /profiles/context | query / repository unexpected error | 500 |
| GET /profiles/tag-catalog | repository / query service error | 500 |

### 6. 資料模型與持久化

關鍵檔案

- internal/auth/model/user.go (line 5)
- internal/link/model/link_user.go (line 5)
- internal/link/model/link.go (line 13)
- internal/profile/model/subject_profile.go (line 5)
- internal/profile/model/tag_catalog.go (line 3)

users：

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | 主鍵 |
| email | string | 登入帳號 |
| password | string | bcrypt hash |
| display_name | string | 顯示名稱 |
| role | string | user / vip / admin |
| created_at | time | 建立時間 |
| is_active | bool | 軟刪除狀態 |

link_users：

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | 對應 auth user id |
| display_name | string | 搜尋與列表顯示 |
| updated_at | time | 最近更新時間 |
| is_active | bool | 是否可被搜尋與申請 |

links：

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | 主鍵 |
| requester_id | string | 發起申請者 |
| target_id | string | 被申請者 |
| participants | []string | 用於 array-contains 查詢 |
| status | string | pending / active / rejected / blocked |
| created_at | time | 建立時間 |
| updated_at | time | 更新時間 |

subject_profiles：

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | ownerID__subjectID 主鍵 |
| owner_id | string | 維護這份資料的使用者 |
| subject_id | string | 被維護的對象 |
| note_lines | []string | 最多三條補充短句 |
| selected_tags | []object | canonical groupKey/tagKey |
| updated_at | time | 最近更新時間 |

profile_tag_groups：

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| group_key | string | 穩定 group key |
| label | string | LinkChat UI 顯示名稱 |
| selection_mode | string | single / multi |
| active | bool | 是否可被選用 |
| order_no | int | UI 排序 |

profile_tags：

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | groupKey__tagKey 主鍵 |
| group_key | string | 所屬 group |
| tag_key | string | 穩定 tag key |
| label | string | LinkChat UI 顯示名稱 |
| active | bool | 是否可被選用 |
| order_no | int | UI 排序 |

目前持久化設計補充：

- subject_profiles 用單文件保存 notes + tags，避免跨 collection merge。
- subject profile 的 label 不會和 tags 一起存，主要持久化的是 canonical key。
- tag catalog 目前由 seeder 植入，不是從 UI 動態管理。

### 7. Seeder 與測試成熟度

關鍵檔案

- internal/auth/seeder/auth_seeder.go (line 16)
- internal/link/seeder/link_seeder.go (line 16)
- internal/profile/seeder/profile_seeder.go (line 11)
- internal/profile/service/validator/profile_validator_test.go (line 10)
- internal/profile/usecase/command/profile_usecase_test.go (line 17)

ProfileSeeder 植入內容：

```text
profile_tag_groups
├─ role
├─ communication_style
└─ support_need

profile_tags
├─ role__student
├─ role__coworker
├─ role__family
├─ communication_style__slow_warmup
├─ communication_style__direct
├─ communication_style__step_by_step
├─ support_need__reassurance
└─ support_need__space_first
```

目前已存在的測試檔：

- internal/profile/service/validator/profile_validator_test.go
- internal/profile/usecase/command/profile_usecase_test.go

目前測到的場景：

- note normalization
- note limit rejection
- tag normalization
- tag dedupe
- single-select conflict
- SaveSubjectNotes 會保留既有 tags
- SaveSubjectTags 在空資料時會刪 profile
- inaccessible subject 會被拒絕

目前成熟度結論：

- auth / link 多數仍以手動驗證與 seeder 為主
- profile 已開始有 deterministic rule 的 unit test
- handler、repository、整體 integration flow 仍缺自動化測試

## 結語

這份 LinkChat Go backend 現在最準確的理解，不再是「只有 auth/link 的底座」，而是「已把人物背景資料也收進來，但還沒把 AI 入口接上」。

也就是說：

- 資料入口已經比以前完整
- 邊界比以前清楚
- 但真正的 LinkChat -> InternalAICopliot flow 仍然還不在 runtime

如果之後要再往 AI 功能推進，最合理的下一步不是回頭重做 profile，而是補上 copilot integration 與跨 repo request contract。
