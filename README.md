# LinkChat

## 專案簡介

LinkChat 是一個以 Go 撰寫的 backend 專案。

這個專案最初以社交互動產品為方向，現階段則暫時轉為個人使用的 AI 輔助分析 backend，用來累積：

- 結構化資料建模
- LinkChat 與 InternalAICopliot 的整合經驗
- AI 分析入口與資料契約設計
- 後端模組分層與協作經驗

目前專案仍在演化中，並非完整上架版產品。

## 目前技術

- Go
- Gin
- Firestore
- Firebase Admin SDK
- JWT
- bcrypt

## 專案目前重點

目前專案有兩條並行重點：

1. 維持既有基礎模組可運作
   - `auth`
   - `link`

2. 完成 LinkChat 作為 AI 分析入口與資料來源系統
   - 保存結構化 traits / codes
   - 整理對 `InternalAICopliot` 的固定 request contract
   - 呼叫 `InternalAICopliot` 並回傳分析結果

## 主要文件

- [PLAN.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/PLAN.md)
  - 專案規劃、方向調整原因、階段性目標
- [ARCHITECTURE.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/ARCHITECTURE.md)
  - 架構說明、模組責任、資料流
- [DEVELOPMENT.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/DEVELOPMENT.md)
  - 開發規範、分層原則、CQRS、跨模組規則

## 文件權威來源

為避免多份文件重複維護而產生不一致，建議以下列方式理解：

- `PLAN.md`
  - 回答「為什麼要這樣做」與「現在做到哪」
- `ARCHITECTURE.md`
  - 回答「系統怎麼切、資料流怎麼走」
- `DEVELOPMENT.md`
  - 回答「實際寫程式時要遵守什麼規則」

若三者有重疊，涉及實作規範時應以 `DEVELOPMENT.md` 為主。

## 架構圖

根目錄下的 [架構圖.drawio](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/%E6%9E%B6%E6%A7%8B%E5%9C%96.drawio) 是目前架構草圖來源之一。

文字版說明與最新整理請以 [ARCHITECTURE.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/ARCHITECTURE.md) 為主；若 drawio 與文件內容不一致，應優先更新文件，再同步修正 drawio。

## 目錄說明

- `cmd/`
  - 啟動入口
- `internal/`
  - 主要業務模組
- `config/`
  - 預留給未來設定檔、環境設定與部署設定使用
- `pkg/`
  - 預留給未來可跨模組重用、且不屬於單一業務模組的共用元件

目前 `config/` 與 `pkg/` 仍為保留目錄。

## 執行狀態說明

目前 [cmd/api/main.go](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/cmd/api/main.go) 偏向開發用啟動方式，包含：

- 連接 Firestore emulator
- 清空部分資料集合
- 執行 seed

因此目前不應直接視為正式環境啟動流程。

## 開發原則摘要

- 保持 module boundary 清楚
- CQRS 只套用在 `usecase` 與 `service`
- 跨模組 `command` 只能 `usecase -> usecase`
- 跨模組 `query` 也應 `usecase -> usecase`
- `middleware` 可跨模組重用，但僅限 request/admission 類用途
- `validator` 預設只作為 module 內部驗證層
- `LinkChat` 只保存結構化資料與跨專案傳輸所需欄位
- 私有理論 mapping / prompt 組裝 / AI provider 呼叫由 `InternalAICopliot` 負責
- `LinkChat` 自己不應再維護本地 AI pipeline 的 `gatekeeper/composer/source/memory/aiclient` 分層

詳細規範請看 [DEVELOPMENT.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/DEVELOPMENT.md)。
