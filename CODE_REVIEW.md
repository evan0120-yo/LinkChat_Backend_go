# LinkChat CODE REVIEW

## 範圍

- 本文件只描述 LinkChat\Backend\Go\LinkChat 目前實際落地的 Go backend。
- 來源優先順序是 current code -> runtime wiring -> 既有文件。
- LinkChat\Backend\Java\LinkChat 不在這份 CODE_REVIEW 範圍內。

## BLOCK 1: AI 對產品的想像

從現在這份 code 看，LinkChat 還不是一個已經長成的 AI 軍師產品，比較像是一個拿來打底的後端骨架。

它現在真的有的，是帳號、JWT、聯絡人搜尋、好友關係與一些驗證測試路由。它看起來像是在為後續更大的 AI 分析產品暖機，但 AI 本體目前還沒接上。README、PLAN、ARCHITECTURE 講的 traits、copilot integration、InternalAICopliot 邊界，還停在規劃層，不在 runtime。

我的推測是，這個專案現在比較像個小規模、開發者自用的 backend playground。啟動時會直接清空 emulator 資料再重灌 seed，這種做法很明顯不是給正式環境用，而是為了快速重跑 auth/link 流程。

它不是什麼：

- 不是 production-ready 服務
- 不是已接好 AI 分析入口的產品
- 不是完整社交平台
- 不是有完整權限模型與測試覆蓋的成熟後端

## BLOCK 2: 讀者模式

### 1. 這個系統現在實際是什麼

它現在實際上是一個只落地了 auth 與 link 的 Gin + Firestore emulator backend。

你可以把它理解成：先把「人怎麼進來、彼此怎麼建立關係」這兩件事做起來，後面再看要不要把 AI 功能接上去。整體的執行起點非常直接，沒有太多基礎設施抽象。

```text
啟動 main
│
├─ 連 Firestore emulator
├─ 把 users / link_users / links 清空
├─ 組 Link module
├─ 組 Auth module
├─ 註冊 /citrus 路由
├─ 跑 auth seed
├─ 跑 link seed
└─ listen :8082
```

> 注意: 每次啟動都會清空三個集合，所以這份 code 的預設心智模型是「開發環境重跑流程」，不是「保存既有資料」。

> 注意: 文件裡提到的 AI pipeline、traits/profile、copilot integration，現在都沒有對應模組與 API。

### 2. 帳號與登入

這塊做的事情很單純：註冊、登入、刪除帳號，以及兩個測試用的受保護 API。

註冊時，系統不只是建一個 users 文件，還會順手幫 link 模組建一份 link_users projection，讓之後搜尋人與建立好友關係可以直接用。這個同步不是靠 MQ 或 event，而是直接包在同一個 Firestore transaction 裡做掉。

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

登入流程也很直接：查 email、驗密碼、簽 token。這裡沒有額外的帳號狀態檢查，也沒有 refresh token、session、blacklist 之類的配套。

刪除帳號不是 hard delete。現在的做法是把 users.is_active 與 link_users.is_active 都改成 false，等於告訴系統「這個人不要再當活人用」，但舊的 links 關係資料會留下來。

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

> 注意: login 沒有檢查 is_active，所以被刪成 inactive 的帳號，現在只要密碼對，還是能登入拿 token。

> 注意: /citrus/auth/delete 的權限錯誤最後會被 handler 回成 500，不是比較合理的 403。

### 3. 搜尋與好友關係

link 模組現在最像一個「簡化版關係系統」。它不是聊天，也不是推薦，而是提供搜尋對象、送出申請、接受/拒絕/取消/解除這些基礎動作。

搜尋是 prefix search，只搜 active 的 link_users，最多回 20 筆。這份資料不是直接查 auth users，而是查 auth 同步過來的 projection。

好友申請的核心邏輯是：不能加自己、不能加不存在或 inactive 的人、兩人之間不能已經有任何 link 文件。通過之後才會寫一筆 pending link。

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

接受、拒絕、取消、解除則是把操作權限與狀態限制切得很明確：

```text
pending
├─ target accept  -> active
├─ target reject  -> rejected
└─ requester cancel -> hard delete

active
└─ 任一 participant remove -> hard delete
```

> 注意: 這裡的「不能重複申請」是很硬的。只要兩人之間已有任何 link 文件，包含 rejected，後續都不能再 Apply。

> 注意: blocked 狀態雖然存在於 model，但目前沒有任何 API 會把 link 寫成 blocked。

### 4. 好友列表與狀態流轉

好友列表不是直接把 links 原封不動吐出去，而是會做一次「把資料變成人比較能看的列表項目」的組裝。

流程是先把與我有關的 links 都撈出來，再找每一筆關係中的「對方」是誰，批次去查 link_users，把名字補上，最後再把 pending 轉成 pending_sent 或 pending_received，並加上 direction。

```text
GetLinkList
│
├─ 先撈我的所有 links
├─ 收集每筆的 otherID
├─ 批次查 link_users
├─ 轉換狀態
│  ├─ pending + 我是 requester -> pending_sent
│  ├─ pending + 我是 target    -> pending_received
│  ├─ active                  -> active
│  └─ rejected                -> rejected
├─ 套 filter
└─ 排序
   ├─ active
   ├─ pending_received
   ├─ pending_sent
   └─ rejected
```

這裡有兩個產品層面很值得注意的點。第一，預設 all 列表會隱藏 blocked，但 rejected 會留著。第二，批次查 link_users 的那條路沒有再過濾 is_active，所以如果某個人被刪成 inactive，但舊 link 還在，他還是可能出現在列表裡。

狀態流轉可以整理成下面這樣：

```text
pending
├─ accept by target
│  └─ active
├─ reject by target
│  └─ rejected
└─ cancel by requester
   └─ deleted

active
└─ remove by participant
   └─ deleted

blocked
└─ 目前沒有落地寫入路徑
```

> 注意: DeleteUser 不會清 links，所以「inactive user 還留在好友列表裡」不是理論上的可能，而是現在資料模型自然會發生的結果。

### 5. 測試資料與成熟度

這個專案現在的驗證方式很偏手動。它有 seeder，也有 /citrus/test/ping、/citrus/test/profile、/citrus/test/system 這些測試路由，但沒有真正的 test file。

auth seeder 會透過正式的 Register usecase 建三個帳號；link seeder 再去找 Normal User 與 EvanHe，替他們建一筆 pending 關係。這種 seed 方式的好處是能順手驗證用例流程，壞處是 seed 會一起繼承目前實作的限制。

```text
AuthSeeder
├─ 建 System Admin
├─ 建 Normal User
└─ 建 EvanHe

LinkSeeder
├─ 用名字找 Normal User
├─ 用名字找 EvanHe
└─ ApplyLink: Normal User -> EvanHe
```

> 注意: 因為 Register 一律把 role 寫成 user，預設 seed 並不會真的產生 admin，所以 /citrus/test/system 這條 admin-only 路由，預設狀態下其實打不通。

> 注意: 目前沒有 *_test.go。這個專案更像「能跑起來、能手動驗流程」，而不是「已有自動化回歸保護」。

## BLOCK 3: 技術補充

### 1. 這個系統現在實際是什麼

關鍵檔案

- cmd/api/main.go (line 38)
- internal/link/provider.go (line 25)
- internal/auth/provider.go (line 23)

啟動與 wiring：

```text
cmd/api/main.go
│
├─ clearCollection("users")      line 59
├─ clearCollection("link_users") line 60
├─ clearCollection("links")      line 61
│
├─ link.NewLinkModule(client)                                  line 69
├─ auth.NewAuthModule(client, linkModule.LinkUserCommandUseCase) line 73
│
├─ authHandler.RegisterRoutes(rootGroup, authMiddleware)       line 88
├─ testHandler.RegisterRoutes(rootGroup, authMiddleware)       line 89
├─ linkModule.Handler.RegisterRoutes(rootGroup, authMiddleware) line 93
│
├─ authSeeder.Seed(ctx)                                        line 98
├─ linkModule.Seeder.Seed(ctx)                                 line 103
└─ r.Run(":8082")                                              line 108
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

目前未接上的模組：

- traits / profile
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

- internal/link/handler/link_handler.go (line 70)
- internal/link/usecase/command/link_usecase.go (line 51)
- internal/link/service/command/link_service.go (line 43)
- internal/link/service/validator/link_validator.go (line 17)
- internal/link/repository/link_repository.go (line 89)
- internal/link/repository/link_user_repository.go (line 141)

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

關係 API 錯誤映射：

| API | 條件 | HTTP status |
| --- | --- | --- |
| /links/search | token 無效 | 401 |
| /links/search | JSON binding 失敗 | 400 |
| /links/search | repository / query 錯誤 | 500 |
| /links/apply | token 無效 | 401 |
| /links/apply | 自己加自己、target 不存在、已有 link | 400 |
| /links/accept | 非 target 或非 pending | 400 |
| /links/reject | 非 target 或非 pending | 400 |
| /links/cancel | 非 requester 或非 pending | 400 |
| /links/remove | 非 participant 或非 active | 400 |

目前實作限制：

- FindLinkByParticipants 只要找到任何一筆該兩人參與的 link 就算重複，不區分 rejected / active / pending。
- blocked 只存在常數，沒有對應 usecase。

### 4. 好友列表與狀態流轉

關鍵檔案

- internal/link/handler/link_handler.go (line 241)
- internal/link/usecase/query/link_usecase.go (line 39)
- internal/link/repository/link_repository.go (line 113)
- internal/link/repository/link_user_repository.go (line 86)
- internal/link/object/resp/link_item_resp.go (line 3)

列表組裝細節：

```text
GET /citrus/links/list
-> VerifyToken middleware
-> LinkHandler.GetLinkList
-> LinkQueryUseCase.GetLinkList
   -> LinkQueryService.GetLinksByUserID
      -> LinkRepository.FindLinksByUserID
   -> LinkUserQueryService.GetLinkUsersByIDs
      -> LinkUserRepository.FindByIDs
   -> usecase 內做：
      1. otherID 推導
      2. userMap 組裝
      3. status 轉換
      4. filter 套用
      5. memory sort
```

狀態轉換表：

| 原始 link.status | 視角 | 輸出 status | direction |
| --- | --- | --- | --- |
| pending | requester | pending_sent | outgoing |
| pending | target | pending_received | incoming |
| active | 任意 | active | none |
| rejected | 任意 | rejected | none |
| blocked | 任意 | blocked | none |

filter 規則：

| filter | 保留項目 |
| --- | --- |
| active | active |
| received | pending_received |
| sent | pending_sent |
| all 或空字串 | 除 blocked 外都保留 |

排序規則：

| 順序 | 規則 |
| --- | --- |
| 1 | 狀態權重：active -> pending_received -> pending_sent -> rejected |
| 2 | 同權重時，DisplayName 開頭 ASCII 優先 |
| 3 | 再同類型時，直接比較 DisplayName 字串 |

目前實作限制：

- FindByIDs 會批次查 link_users，但不檢查 is_active。
- DeleteUser 只把 link_user 標 inactive，不會清理 links。
- 兩者相加後，inactive 使用者仍可能出現在列表。

### 5. 測試資料與成熟度

關鍵檔案

- internal/auth/seeder/auth_seeder.go (line 29)
- internal/link/seeder/link_seeder.go (line 35)
- go.mod (line 1)

Seeder 流程：

```text
AuthSeeder.Seed
│
├─ Register(admin@linkchat.com, admin123, System Admin)
├─ Register(user@linkchat.com, user123, Normal User)
└─ Register(evan01203394@gmail.com, evan, EvanHe)

LinkSeeder.Seed
│
├─ SearchUsers("Normal User")
├─ SearchUsers("EvanHe")
└─ ApplyLink(Normal User -> EvanHe)
```

成熟度判斷：

- 有 compile-level 可用的程式結構，但沒有自動化測試檔。
- 主要驗證方式是 seed 與手動打 API。
- 秘密值與啟動模式仍是硬編碼。
- 文件中的 AI 方向尚未體現在目前目錄的程式碼。
