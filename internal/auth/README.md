# Auth 模組

## 模組目的

`auth` 是 LinkChat 的帳號與身份驗證模組。

它負責管理使用者註冊、登入、JWT 驗證、角色資訊，以及和其他模組之間與身份相關的同步流程。

在目前專案中，這是基礎模組之一，負責提供整個系統的身份基底。

## 目前已承擔的功能

- 使用者註冊
- 使用者登入
- JWT 產生與驗證
- middleware 驗證登入狀態
- 部分角色權限控制
- 與 `link` 模組同步基本 user projection

## 模組價值

即使專案短期內轉向個人使用的 AI 分析工具，`auth` 仍然有價值，因為：

- request admission 仍可能依賴登入狀態
- 後續若恢復多使用者模式可繼續沿用
- `gatekeeper` 可直接重用這裡的身份驗證能力

## 主要責任

- 使用者帳號建立與登入
- token 發行與驗證
- 使用者基本身份查詢
- 權限資訊提供
- 帳號狀態同步

## 不負責的事情

- 不負責 AI context 組裝
- 不負責知識檢索
- 不負責 AI provider 呼叫
- 不應承擔過多與分析邏輯相關的責任

## 與其他模組的關係

- `gatekeeper` 會重用 `auth` 的驗證能力
- `link` 目前依賴 `auth` 的 user lifecycle
- 未來 `composer` 或 `source` 若需要 user identity，也應透過明確 query 能力取得，而不是直接耦合內部細節

## 後續調整方向

### 第一階段

- 保持現有登入與驗證流程可用
- 避免為了 AI 功能重寫整個 auth

### 第二階段

- 將身份驗證能力更清楚地對接 `gatekeeper`
- 視需要補 config 化與 secret 管理

### 第三階段

- 若重新回到多使用者正式產品模式，再補更完整的 permission model 與 user lifecycle 規則
