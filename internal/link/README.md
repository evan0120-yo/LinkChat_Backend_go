# Link 模組

## 模組目的

`link` 是 LinkChat 的關係與聯絡人資料基礎模組。

目前它主要負責：

- searchable user projection
- 關係資料
- 好友申請流程
- 關係查詢與列表

這個模組原本是社交產品核心的一部分，但在新的 AI 導向規劃下，它仍然保有很高的再利用價值。

## 目前已承擔的功能

- 搜尋使用者
- 建立好友申請
- 接受、拒絕、取消、解除關係
- 查詢好友列表
- 維護 `link_users` 與 `links` 資料模型

## 模組價值

在新的規劃下，`link` 可視為：

- 聯絡人資料來源
- 關係資料來源
- AI 分析前的結構化社交 context 基底

未來即使不先做完整社交平台，這個模組仍可作為：

- `source` 模組的底層來源之一
- contact profile 與 relationship metadata 的資料基礎

## 主要責任

- 聯絡人 / 關係資料維護
- 關係狀態更新
- 搜尋與列表查詢
- user projection 維護

## 不負責的事情

- 不負責 AI context 組裝
- 不負責理論知識檢索
- 不直接與 AI provider 溝通
- 不應直接承擔 provider-specific prompt 格式責任

## 與其他模組的關係

- 目前與 `auth` 有同步關係
- 未來 `source` 很可能會重用 `link` 的 query 能力
- `composer` 不應直接依賴 `link` 的內部細節，而應透過 `source` 取得統一格式資料

## 後續演化方向

### 第一階段

- 保持目前可作為 contact/relationship data source 的角色
- 不急著重寫成全新 contacts 模組

### 第二階段

- 視需要將 AI 使用的結構化資料逐步抽到 `source`
- 讓 `link` 回歸專注在 relationship domain

### 第三階段

- 若未來產品重新回到上架版本，可再強化完整社交互動流程
