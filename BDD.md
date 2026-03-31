# LinkChat BDD

## 範圍

- 本文件只描述 LinkChat\Backend\Go\LinkChat 目前已落地的 Go backend 行為。
- 行為來源以目前程式碼為準，不以 PLAN.md、ARCHITECTURE.md 中尚未落地的 AI 規劃為準。
- LinkChat\Backend\Java\LinkChat 不在本文件範圍內。

## Feature: 開發模式啟動

### Scenario: 啟動 API 會清空 emulator 資料並重新植入 seed

Given 開發者直接啟動 cmd/api/main.go

When 服務初始化 Firestore emulator 與 Gin router

Then 系統會先清空 users、link_users、links 三個集合

And 之後才註冊 /citrus 路由

And 啟動前會自動執行 auth seeder 與 link seeder

And API 監聽在 :8082

And 這是一個開發模式啟動流程，不是 production-safe 啟動方式

## Feature: 註冊帳號

### Scenario: 合法註冊會同時建立 auth user 與 link user

Given users 集合中還沒有相同 email

And request body 具有合法的 email、password、display_name

When client 呼叫 POST /citrus/auth/register

Then response status 會是 201

And 系統會在同一個 Firestore transaction 內建立 users 文件與 link_users 文件

And users.role 一律寫成 user

And users.is_active 與 link_users.is_active 都會是 true

### Scenario: email 重複時註冊失敗

Given users 集合中已經存在相同 email

When client 呼叫 POST /citrus/auth/register

Then response status 會是 400

And error message 會包含 email already exists

### Scenario: HTTP 請求格式不合法時直接被 handler 擋下

Given request body 缺少必要欄位

Or password 長度小於 6

When client 呼叫 POST /citrus/auth/register

Then response status 會是 400

And usecase 不會被執行

## Feature: 登入

### Scenario: 帳密正確時取得 Bearer token

Given 使用者存在於 users 集合

And 密碼比對成功

When client 呼叫 POST /citrus/auth/login

Then response status 會是 200

And response body 會包含 accessToken、tokenType、expiresIn

And token 使用 HS256 簽發

And token 會攜帶 sub、name、role、exp、iss claims

### Scenario: email 不存在或密碼錯誤時回傳相同失敗訊息

Given email 不存在

Or 密碼錯誤

When client 呼叫 POST /citrus/auth/login

Then response status 會是 401

And error message 會是 invalid credentials 或其包裝訊息

### Scenario: 已停用帳號目前仍可登入

Given users.is_active 已被改成 false

And 帳號 email 與密碼仍然正確

When client 呼叫 POST /citrus/auth/login

Then response status 仍然會是 200

And 系統仍然會發 token

And 這是目前程式碼的真實行為，不是文件推測

## Feature: 刪除帳號

### Scenario: 使用者可以刪除自己

Given 使用者已登入

And request body 的 userId 等於 token 內的 sub

When client 呼叫 POST /citrus/auth/delete

Then response status 會是 200

And users.is_active 會被改成 false

And link_users.is_active 也會被改成 false

And 這兩個更新會在同一個 transaction 內完成

### Scenario: 非管理員刪除別人時會被 usecase 拒絕

Given 使用者已登入

And token role 不是 admin

And request body 的 userId 不是自己的 sub

When client 呼叫 POST /citrus/auth/delete

Then usecase 會回傳 permission denied

And handler 目前會回 500，而不是 403

### Scenario: 刪除帳號不會清除既有好友關係文件

Given 目標帳號在 links 集合中仍有關係資料

When 帳號刪除成功

Then links 集合中的既有 link 文件不會被刪除

And 只有 users 與 link_users 會被標成 inactive

## Feature: 搜尋聯絡人

### Scenario: 登入後可以用 display name 前綴搜尋活躍使用者

Given 使用者已登入

And request body 提供 displayName

When client 呼叫 POST /citrus/links/search

Then response status 會是 200

And 系統會以 display_name prefix search 查詢 link_users

And 最多回傳 20 筆

And 只回傳 is_active = true 的 link_users

## Feature: 送出好友申請

### Scenario: 合法申請會建立 pending 關係

Given 使用者已登入

And targetId 對應到存在且 active 的 link user

And 兩人之間目前沒有任何 link 文件

When client 呼叫 POST /citrus/links/apply

Then response status 會是 200

And 系統會建立一筆 status = pending 的 link 文件

And participants 會保存 requesterID 與 targetID

### Scenario: 不能加自己

Given request body 的 targetId 等於 token 內的 sub

When client 呼叫 POST /citrus/links/apply

Then response status 會是 400

And error message 會包含 cannot link with yourself

### Scenario: 目標不存在或 inactive 時申請失敗

Given targetId 對應不到 link user

Or link user 的 is_active = false

When client 呼叫 POST /citrus/links/apply

Then response status 會是 400

And error message 會包含 target user not found or inactive

### Scenario: 只要兩人之間已有任何 link 文件，就不能再次申請

Given 兩人之間已存在一筆 links 文件

And 該文件可能是 pending、active 或 rejected

When client 呼叫 POST /citrus/links/apply

Then response status 會是 400

And error message 會包含 link already exists or pending

## Feature: 接受、拒絕、取消、解除關係

### Scenario: 只有 target 能接受 pending 申請

Given 一筆 link.status = pending 的好友申請存在

And operator 是該 link 的 target

When client 呼叫 POST /citrus/links/accept

Then response status 會是 200

And link.status 會變成 active

### Scenario: 只有 target 能拒絕 pending 申請

Given 一筆 link.status = pending 的好友申請存在

And operator 是該 link 的 target

When client 呼叫 POST /citrus/links/reject

Then response status 會是 200

And link.status 會變成 rejected

### Scenario: 只有 requester 能取消 pending 申請

Given 一筆 link.status = pending 的好友申請存在

And operator 是該 link 的 requester

When client 呼叫 POST /citrus/links/cancel

Then response status 會是 200

And 該 link 文件會被 hard delete

### Scenario: 只有 active 關係才能被解除

Given 一筆 link.status = active 的關係存在

And operator 是 link 其中一個 participant

When client 呼叫 POST /citrus/links/remove

Then response status 會是 200

And 該 link 文件會被 hard delete

### Scenario: 操作者不符或狀態不符時操作失敗

Given operator 不是允許的角色

Or link 狀態不是該 API 預期的狀態

When client 呼叫 accept、reject、cancel 或 remove

Then response status 會是 400

And error message 由 usecase 或 service 直接回傳

## Feature: 好友列表

### Scenario: 預設列表會隱藏 blocked，但會顯示 rejected

Given 使用者已登入

When client 呼叫 GET /citrus/links/list

Then response status 會是 200

And 預設 filter 為 all

And blocked 狀態不會出現在預設列表中

And active、pending_received、pending_sent、rejected 可能出現在列表裡

### Scenario: 系統會把原始狀態轉成前端可讀狀態

Given 某筆 link.status = pending

When requester 查列表

Then 該筆資料會顯示為 pending_sent

And direction = outgoing

When target 查列表

Then 該筆資料會顯示為 pending_received

And direction = incoming

### Scenario: 列表支援 active、received、sent 篩選

Given 使用者已登入

When client 呼叫 GET /citrus/links/list?filter=active

Then 只會保留 active

When client 呼叫 GET /citrus/links/list?filter=received

Then 只會保留 pending_received

When client 呼叫 GET /citrus/links/list?filter=sent

Then 只會保留 pending_sent

### Scenario: 列表排序規則固定

Given 列表中同時存在 active、pending_received、pending_sent、rejected

When 系統完成組裝

Then 排序優先權會是 active -> pending_received -> pending_sent -> rejected

And 若狀態相同，DisplayName 開頭為 ASCII 的資料排前面

And 若仍同類型，最後以字串字典序比較 DisplayName

### Scenario: 已停用使用者仍可能因既有 link 文件出現在列表

Given 某個對象在 link_users 已被標成 inactive

And 兩人之間仍保留既有 links 文件

When client 呼叫 GET /citrus/links/list

Then 該對象仍可能出現在列表中

And 原因是批次查詢 GetLinkUsersByIDs 不會過濾 is_active

## Feature: 測試路由

### Scenario: /citrus/test/ping 是公開路由

Given client 沒有帶 token

When client 呼叫 POST /citrus/test/ping

Then response status 會是 200

### Scenario: /citrus/test/profile 需要有效 token

Given client 帶著合法 Bearer token

When client 呼叫 POST /citrus/test/profile

Then response status 會是 200

And response 會回傳 middleware 解析出的 user_id 與 role

### Scenario: /citrus/test/system 需要 admin role

Given client 帶著 role = admin 的 token

When client 呼叫 POST /citrus/test/system

Then response status 會是 200

And response message 會是測試用的假刪除成功訊息

And 目前預設 seed 不會真的建立 admin，因此這個情境預設不會自然成立
