# LinkChat SDD

## 範圍與來源

- 本文件描述 LinkChat\Backend\Go\LinkChat 目前已落地的 Go backend 設計。
- 來源以目前程式碼為準，特別是 cmd/api/main.go、internal/auth、internal/link、internal/profile。
- 本文件不把 PLAN.md 中的 copilot integration、InternalAICopliot 呼叫、AI pipeline 當成已實作能力。
- LinkChat\Backend\Java\LinkChat 不在本文件範圍內。

## 1. 系統摘要

LinkChat 目前實際上是一個偏開發驗證用途的 Go backend。

已落地的核心有三個業務模組：

- auth
- link
- profile

目前對外可用能力是：

- 註冊
- 登入
- JWT 驗證
- 刪除帳號
- 搜尋使用者
- 好友申請、接受、拒絕、取消、解除
- 好友列表查詢
- 人物補充三句保存
- 人物 tag 保存
- 人物背景資料查詢
- tag catalog 查詢
- 驗證測試路由

目前沒有落地的能力：

- copilot integration
- InternalAICopliot 呼叫
- AI 分析入口

## 2. Runtime 結構

關鍵檔案

- cmd/api/main.go (line 38)
- internal/link/provider.go (line 25)
- internal/auth/provider.go (line 23)

啟動流程如下：

```text
main
│
├─ 連 Firestore emulator
│  FIRESTORE_EMULATOR_HOST=localhost:8090
│
├─ 清空集合
│  users
│  link_users
│  links
│  subject_profiles
│  profile_tag_groups
│  profile_tags
│
├─ 建立 Link module
│  ├─ repository
│  ├─ service
│  ├─ usecase
│  ├─ seeder
│  └─ handler
│
├─ 建立 Profile module
│  ├─ repository
│  ├─ service
│  ├─ usecase
│  ├─ seeder
│  └─ handler
│
├─ 建立 Auth module
│  └─ 注入 LinkUserCommandUseCase
│
├─ 註冊 /citrus 路由
│
├─ 執行 AuthSeeder
│
├─ 執行 LinkSeeder
│
├─ 執行 ProfileSeeder
│
└─ Gin listen :8082
```

這個 runtime 有兩個很明顯的設計特徵：

- 啟動即清資料，所以整體偏開發驗證用途
- auth 直接依賴 link 的 LinkUserCommandUseCase，負責在註冊與刪除時同步 projection

## 3. 模組與分層設計

### 3.1 auth 模組

關鍵檔案

- internal/auth/provider.go (line 23)
- internal/auth/handler/auth_handler.go (line 33)
- internal/auth/usecase/command/auth_usecase.go (line 52)
- internal/auth/usecase/query/auth_usecase.go (line 31)

auth 模組負責：

- 註冊 users 文件
- 登入驗證
- JWT 簽發
- JWT middleware
- 刪除帳號時同步標記 inactive
- 測試路由 /citrus/test/*

auth 分層責任：

```text
handler
└─ 做 HTTP binding 與 status code mapping

usecase
├─ Register: 驗證、hash、產 ID、開 transaction、同步 link user
├─ Login: 查 user、驗密碼、簽 token
└─ DeleteUser: 權限判斷、查 user、開 transaction、同步 link user inactive

service
├─ command: hash password、寫入 user
└─ query: 查 user、驗密碼、簽 token

repository
└─ Firestore users collection 存取
```

### 3.2 link 模組

關鍵檔案

- internal/link/provider.go (line 25)
- internal/link/handler/link_handler.go (line 29)
- internal/link/usecase/command/link_usecase.go (line 51)
- internal/link/usecase/query/link_usecase.go (line 39)

link 模組負責：

- 維護 link_users projection
- 搜尋聯絡人
- 維護 links 關係資料
- 產生好友列表輸出模型

link 分層責任：

```text
handler
└─ 取 token 內 userID，轉成 request DTO，回傳 HTTP response

usecase/command
├─ ApplyLink: 驗證、自查重複、檢查 target 是否存在、交易寫入 pending
├─ AcceptLink: target 接受 pending
├─ RejectLink: target 拒絕 pending
├─ CancelLink: requester 收回 pending
└─ RemoveLink: participant 移除 active

usecase/query
├─ SearchUsers: prefix search
└─ GetLinkList: 撈 link、批次撈人、轉換狀態、過濾、排序

service
├─ command: Reject/Remove/Cancel 的 domain rule
└─ query: 讀取 link 與 link_user

repository
├─ link_users collection
└─ links collection
```

### 3.3 profile 模組

關鍵檔案

- internal/profile/provider.go (line 17)
- internal/profile/handler/profile_handler.go (line 14)
- internal/profile/usecase/command/profile_usecase.go (line 17)
- internal/profile/usecase/query/profile_usecase.go (line 12)

profile 模組負責：

- 保存 owner 對 linked subject 的人物補充三句
- 保存 owner 對 linked subject 的 tag 選擇
- 提供 tag catalog 查詢
- 提供 subject profile context 查詢

profile 分層責任：

```text
handler
└─ 取 token 內 userID，做 request binding 與 HTTP response mapping

usecase/command
├─ 檢查 subject 是否為 active linked user
├─ notes / tags normalization
├─ 讀取既有 subject profile
├─ merge 另一半既有資料
└─ 決定 upsert 或 delete

usecase/query
├─ 讀取 subject profile context
└─ 讀取 active tag catalog

service
├─ command: 寫入 / 刪除 subject profile
├─ query: 查 subject profile 與 tag catalog
└─ validator: notes / tags deterministic rule

repository
├─ subject_profiles collection
├─ profile_tag_groups collection
└─ profile_tags collection
```

這個模組有一個很明確的第一版限制：

- `subjectId` 必須對應目前 active 的 linked user
- profile 模組透過 `link usecase/query` 驗證這個 access rule

### 3.4 模組依賴方向

```text
main
├─ link module
├─ profile module
│  └─ 依賴 link.LinkQueryUseCase
└─ auth module
   └─ 依賴 link.LinkUserCommandUseCase

auth.register
└─ transaction 內同步建立 link_user

auth.delete
└─ transaction 內同步把 link_user 標成 inactive
```

這個依賴方向代表：

- auth 是帳號真實來源
- link_users 是給 link 模組查詢與搜尋用的 projection
- 目前沒有 event bus，也沒有 async sync

## 4. HTTP 介面設計

### 4.1 路由總表

| Method | Path | Auth | 說明 |
| --- | --- | --- | --- |
| GET | /citrus/health | 否 | 健康檢查 |
| POST | /citrus/auth/register | 否 | 註冊 |
| POST | /citrus/auth/login | 否 | 登入 |
| POST | /citrus/auth/delete | 是 | 刪除帳號 |
| POST | /citrus/test/ping | 否 | 公開測試 |
| POST | /citrus/test/profile | 是 | 驗證 middleware |
| POST | /citrus/test/system | 是，且 admin | 權限測試 |
| POST | /citrus/links/search | 是 | 搜尋使用者 |
| POST | /citrus/links/apply | 是 | 送出申請 |
| POST | /citrus/links/accept | 是 | 接受申請 |
| POST | /citrus/links/reject | 是 | 拒絕申請 |
| POST | /citrus/links/remove | 是 | 解除好友 |
| POST | /citrus/links/cancel | 是 | 取消申請 |
| GET | /citrus/links/list | 是 | 好友列表 |
| PUT | /citrus/profiles/notes | 是 | 保存人物補充三句 |
| PUT | /citrus/profiles/tags | 是 | 保存人物 tag |
| GET | /citrus/profiles/context | 是 | 查詢人物背景資料 |
| GET | /citrus/profiles/tag-catalog | 是 | 查詢可用 tag catalog |

### 4.2 Request admission

關鍵檔案

- internal/auth/middleware/auth_middleware.go (line 36)
- internal/auth/handler/test_handler.go (line 21)
- internal/link/handler/link_handler.go (line 30)

設計規則：

- Bearer token 驗證由 VerifyToken 處理
- VerifyToken 解析後把 userID 與 role 塞進 Gin context
- RequireRole 目前只用在 /citrus/test/system
- /citrus/auth/delete 沒有掛 RequireRole，而是交給 usecase 自行判斷 self-or-admin

## 5. 資料模型與持久化

關鍵檔案

- internal/auth/model/user.go (line 5)
- internal/link/model/link_user.go (line 5)
- internal/link/model/link.go (line 13)
- internal/profile/model/subject_profile.go (line 5)
- internal/profile/model/tag_catalog.go (line 9)
- internal/auth/repository/user_repository.go (line 45)
- internal/link/repository/link_user_repository.go (line 40)
- internal/link/repository/link_repository.go (line 44)
- internal/profile/repository/subject_profile_repository.go (line 13)
- internal/profile/repository/tag_catalog_repository.go (line 13)

### 5.1 users

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | 主鍵 |
| email | string | 登入帳號 |
| password | string | bcrypt hash |
| display_name | string | 顯示名稱 |
| role | string | user / vip / admin |
| created_at | time | 建立時間 |
| is_active | bool | 軟刪除狀態 |

設計說明：

- users 是帳號真實來源
- 刪除帳號不是 hard delete，而是把 is_active 改成 false

### 5.2 link_users

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | 對應 auth user id |
| display_name | string | 搜尋與列表顯示 |
| updated_at | time | 最近更新時間 |
| is_active | bool | 是否可被搜尋與申請 |

設計說明：

- link_users 是 projection，不是主帳號來源
- 搜尋只看 active link_users
- 刪除帳號時只會把 projection 標記 inactive，不會清掉 links

### 5.3 links

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | 主鍵 |
| requester_id | string | 發起申請者 |
| target_id | string | 被申請者 |
| participants | []string | 用於 array-contains 查詢 |
| status | string | pending / active / rejected / blocked |
| created_at | time | 建立時間 |
| updated_at | time | 更新時間 |

設計說明：

- participants 是主要查詢索引策略
- 關係刪除使用 hard delete
- rejected 狀態會留在資料庫
- blocked 常數存在，但目前沒有任何 API 寫入 blocked

### 5.4 subject_profiles

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | `ownerID__subjectID` 組成的主鍵 |
| owner_id | string | 維護這份人物資料的 LinkChat 使用者 |
| subject_id | string | 被維護的人物對象 |
| note_lines | []string | 最多三條補充短句 |
| selected_tags | []object | canonical `groupKey/tagKey` 組合 |
| updated_at | time | 最近更新時間 |

設計說明：

- 第一版 subject profile 只允許綁定 active linked user
- notes 與 tags 共用同一份文件，避免跨 collection merge
- 當 note_lines 與 selected_tags 都為空時，會直接刪除文件

### 5.5 profile_tag_groups

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| group_key | string | 穩定 group key |
| label | string | LinkChat UI 顯示名稱 |
| selection_mode | string | `single` 或 `multi` |
| active | bool | 是否可被選用 |
| order_no | int | UI 排序 |

設計說明：

- group 定義目前由 seeder 植入
- selection_mode 是 validator 的單選 / 多選規則來源

### 5.6 profile_tags

| 欄位 | 型別 | 用途 |
| --- | --- | --- |
| id | string | `groupKey__tagKey` 主鍵 |
| group_key | string | 所屬 tag group |
| tag_key | string | 穩定 tag key |
| label | string | LinkChat UI 顯示名稱 |
| active | bool | 是否可被選用 |
| order_no | int | UI 排序 |

設計說明：

- 保存時只寫 canonical `groupKey/tagKey` 到 subject profile
- label 只保留在 catalog，不進 subject profile 持久化

## 6. 核心流程設計

### 6.1 註冊流程

關鍵檔案

- internal/auth/handler/auth_handler.go (line 61)
- internal/auth/usecase/command/auth_usecase.go (line 52)
- internal/auth/service/validator/auth_validator.go (line 27)
- internal/link/usecase/command/link_user_usecase.go (line 35)

```text
POST /citrus/auth/register
│
├─ handler 做 JSON binding
│
├─ usecase 驗證 email 唯一
│
├─ command service 做 bcrypt hash
│
├─ usecase 產 uuid v7
│
└─ transaction
   ├─ 建 users 文件
   └─ 建 link_users projection
```

目前固定行為：

- role 一律寫成 user
- HTTP 層會檢查 password 長度至少 6
- seeder 直接呼叫 usecase，因此不受 HTTP binding 限制

### 6.2 登入流程

關鍵檔案

- internal/auth/handler/auth_handler.go (line 77)
- internal/auth/usecase/query/auth_usecase.go (line 31)
- internal/auth/service/query/auth_service.go (line 53)

```text
POST /citrus/auth/login
│
├─ 用 email 查 users
├─ bcrypt 驗密碼
└─ 產 JWT token
```

目前固定行為：

- token 過期時間 24 小時
- secret 為 hardcoded 字串 YOUR_SUPER_SECRET_KEY
- login 不檢查 is_active

### 6.3 刪除帳號流程

關鍵檔案

- internal/auth/handler/auth_handler.go (line 94)
- internal/auth/usecase/command/auth_usecase.go (line 111)
- internal/link/usecase/command/link_user_usecase.go (line 51)

```text
POST /citrus/auth/delete
│
├─ middleware 解析 token
├─ usecase 檢查 self-or-admin
├─ 查 target user 是否存在
└─ transaction
   ├─ users.is_active = false
   └─ link_users.is_active = false
```

目前固定行為：

- 不會刪 links 集合中的既有關係
- 權限錯誤最後被 handler 映射成 500

### 6.4 申請與關係操作流程

關鍵檔案

- internal/link/handler/link_handler.go (line 90)
- internal/link/usecase/command/link_usecase.go (line 51)
- internal/link/service/command/link_service.go (line 43)
- internal/link/service/validator/link_validator.go (line 17)

```text
ApplyLink
│
├─ 驗證 requester/target 不可空、不可相同
├─ transaction
│  ├─ 查兩人是否已存在任何 link 文件
│  ├─ 查 target link_user 是否存在且 active
│  └─ 建一筆 pending link
│
AcceptLink
├─ 只能 target 執行
└─ pending -> active
│
RejectLink
├─ 只能 target 執行
└─ pending -> rejected
│
CancelLink
├─ 只能 requester 執行
└─ pending -> hard delete
│
RemoveLink
├─ 任何 participant 可執行
└─ active -> hard delete
```

目前固定行為：

- 只要已有任意 link 文件，就不能重新 Apply
- rejected 會保留，因而阻止重新申請
- blocked 沒有寫入入口

### 6.5 列表查詢流程

關鍵檔案

- internal/link/handler/link_handler.go (line 241)
- internal/link/usecase/query/link_usecase.go (line 39)
- internal/link/repository/link_repository.go (line 113)
- internal/link/repository/link_user_repository.go (line 86)

```text
GET /citrus/links/list
│
├─ 撈出與我有關的所有 links
├─ 收集對方 user ID
├─ 分批查 link_users
├─ 組成 LinkItemResp
│  ├─ pending -> pending_sent / pending_received
│  ├─ active -> active
│  ├─ rejected -> rejected
│  └─ blocked -> blocked
├─ 依 filter 過濾
└─ 依狀態權重與名字排序
```

目前固定行為：

- 預設列表隱藏 blocked
- 批次查詢 FindByIDs 不過濾 is_active
- 因此 inactive 使用者若仍有 links，還是可能進列表

### 6.6 人物補充三句與 tag 保存流程

關鍵檔案

- internal/profile/handler/profile_handler.go (line 29)
- internal/profile/usecase/command/profile_usecase.go (line 31)
- internal/profile/service/validator/profile_validator.go (line 17)

```text
PUT /citrus/profiles/notes or /citrus/profiles/tags
│
├─ middleware 驗證 token
├─ handler 取 ownerID 與 request body
├─ usecase 透過 link query usecase 檢查 subject 是否為 active linked user
├─ validator
│  ├─ notes: trim、移除空字串、限制最多三條、每條最多 60 字
│  └─ tags: canonicalize key、檢查 active catalog、處理 dedupe、檢查 single/multi
├─ 讀取現有 subject_profiles
├─ merge 另一半既有資料
├─ notes + tags 若都空
│  └─ delete subject_profiles
└─ 否則 upsert subject_profiles
```

目前固定行為：

- `subjectId` 不可等於 owner 自己
- `subjectId` 若不是 active linked user，保存會直接失敗
- notes 與 tags 分開保存，但共用同一份 `subject_profiles` 文件

### 6.7 人物背景與 tag catalog 查詢流程

關鍵檔案

- internal/profile/handler/profile_handler.go (line 84)
- internal/profile/usecase/query/profile_usecase.go (line 23)
- internal/profile/repository/tag_catalog_repository.go (line 44)

```text
GET /citrus/profiles/context
│
├─ middleware 驗證 token
├─ usecase 檢查 subject 是否為 active linked user
└─ 讀取 owner__subject 對應的 subject_profiles

GET /citrus/profiles/tag-catalog
│
└─ 讀取 active 的 profile_tag_groups / profile_tags，按 group 組裝回傳
```

目前固定行為：

- 沒有 subject profile 文件時，`/context` 回空資料結構，不視為錯誤
- `/tag-catalog` 只回 active group 與 active tags

## 7. 安全設計

### 7.1 JWT

關鍵檔案

- internal/auth/service/query/auth_service.go (line 53)
- internal/auth/provider.go (line 66)
- internal/auth/middleware/auth_middleware.go (line 36)

設計重點：

- JWT 簽署演算法是 HS256
- secret 目前硬編碼在 auth query service 與 auth middleware 初始化流程
- middleware 會把 sub 映射到 userID，把 role 映射到 userRole

### 7.2 權限模型

關鍵檔案

- internal/auth/model/role.go (line 6)
- internal/auth/handler/test_handler.go (line 42)
- internal/auth/usecase/command/auth_usecase.go (line 117)

目前 role 常數有：

- user
- vip
- admin

但目前實作狀態是：

- 註冊一律產生 user
- 預設 seed 也不會真的產生 admin
- RequireRole 只出現在 /citrus/test/system

## 8. 狀態模型

### 8.1 帳號狀態

```text
active user
└─ DeleteUser
   └─ inactive user
```

```text
active link_user
└─ DeleteUser
   └─ inactive link_user
```

### 8.2 關係狀態

```text
pending
├─ AcceptLink by target
│  └─ active
├─ RejectLink by target
│  └─ rejected
└─ CancelLink by requester
   └─ deleted

active
└─ RemoveLink by participant
   └─ deleted

blocked
└─ 目前沒有任何寫入流程
```

## 9. 已知限制與未實作項

- 啟動流程會清空資料，不適合正式環境。
- README / PLAN 提到的 copilot integration、InternalAICopliot 呼叫都還沒進 code。
- login 不檢查 is_active，停用帳號仍可登入。
- 刪除帳號不會清 links，只會把 users 與 link_users 標 inactive。
- 非管理員刪除別人時，錯誤最後回 500，不是 403。
- rejected 關係會阻止重新申請。
- blocked 狀態存在於模型，但沒有 API 寫入它。
- 預設 seed 的 System Admin 仍會被註冊成 user。
- 目前 `profile` 模組只有 validator / command usecase 的 unit test，其他流程仍主要靠手動驗證。
