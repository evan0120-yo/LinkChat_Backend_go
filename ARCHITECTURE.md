# LinkChat 架構文件

## 概述

LinkChat 正在從一個關係互動導向的 backend prototype，演化成一個個人使用的 AI 輔助分析 backend。

更新後的架構，核心目標不是在 LinkChat 內重建一套完整 AI pipeline，而是把以下責任分清楚：

- 使用者與聯絡人資料維護
- 結構化 traits / codes 保存
- 分析請求入口與身份驗證
- 跨專案資料契約整理
- 對 `InternalAICopliot` 的整合呼叫

## 最新架構圖

目前預計的架構如下：

```text
user
  |
  v
LinkChat API
  |
  +--> auth（身份 / JWT / 角色）
  |
  +--> link（聯絡人 / 關係資料）
  |
  +--> traits / profile（slot -> code / theoryVersion）
  |
  +--> copilot integration（整理 payload，呼叫 InternalAICopliot）
                                  |
                                  v
                        InternalAICopliot
                          - gatekeeper
                          - builder
                          - rag
                          - output
                          - aiclient
```

這張圖對應目前討論後的最新版設計方向。

補充說明：

- `auth` 是目前系統的身份基礎模組
- `link` 是目前系統的關係與聯絡人基礎模組
- `InternalAICopliot` 才是私有理論解釋、prompt 組裝、RAG 與 AI provider 溝通的中心
- 根目錄的 [架構圖.drawio](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/%E6%9E%B6%E6%A7%8B%E5%9C%96.drawio) 是架構草圖檔案；若與本文件不一致，應以本文件文字內容為主，再同步更新 drawio

## 基礎模組位置

目前已存在的重要基礎模組為：

- `auth`
- `link`

它們在架構中的角色如下：

- `auth`
  - 提供身份驗證、JWT、角色與登入基礎
- `link`
  - 提供關係資料、聯絡人資料與可搜尋 user projection

這兩個模組目前仍然是重要底層依賴來源。
未來 AI 分析能力若要成立，也會優先重用它們提供的資料，而不是在 LinkChat 再複製一套本地 AI pipeline。

## 模組責任

本文件只保留架構層級所需的責任摘要。

若需要更細的開發規範、分層限制、CQRS 與跨模組呼叫規則，請以 [DEVELOPMENT.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/DEVELOPMENT.md) 為主。

### `traits / profile`

這是 LinkChat 內用來保存分析所需結構化資料的位置。

主要負責：

- `theoryVersion`
- `slot -> code`
- 分析目標對象的結構化 traits
- 未來其他固定 schema 的 profile system

它代表的是 LinkChat 擁有的事實資料，不是 AI 解釋結果。

### `copilot integration`

這是 LinkChat 與 `InternalAICopliot` 的整合邊界。

主要負責：

- 驗證與分析入口相關的 request
- 整理固定 payload
- 呼叫 `InternalAICopliot`
- 將結果回傳前端
- 處理 timeout / error mapping / integration logging

它不負責：

- 私有理論 mapping
- prompt 組裝
- RAG / retrieval
- AI provider-specific transport 細節

## 主要請求流程

```text
  User                    LinkChat                              InternalAICopliot
   │                        │                                        │
   │  ① 發出 request        │                                        │
   │ ─────────────────────→ │                                        │
   │                        │                                        │
   │                  ② 驗證身份與                                    │
   │                    基本請求格式                                   │
   │                        │                                        │
   │                  ③ 從 auth / link / traits                      │
   │                    取得結構化資料                                  │
   │                        │                                        │
   │                  ④ 組成固定的                                     │
   │                    跨專案 payload                                 │
   │                        │                                        │
   │                        │  ⑤ 呼叫 InternalAICopliot               │
   │                        │ ─────────────────────────────────────→ │
   │                        │                                        │
   │                        │                              ⑥ gatekeeper → builder
   │                        │                                → rag → output
   │                        │                                        │
   │                        │  ⑦ 回傳結果                             │
   │                        │ ←───────────────────────────────────── │
   │                        │                                        │
   │  回傳前端               │                                        │
   │ ←───────────────────── │                                        │
```

關於這條流程中各層的具體開發規則，例如：

- 哪一層可以跨模組呼叫
- CQRS 套用在哪些層
- command 與 query 的界線

請參考 [DEVELOPMENT.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/DEVELOPMENT.md)。

## 為什麼這樣切

```text
問題                                    架構決策                              好處
─────────────────────────────────       ─────────────────────────────────     ─────────────────────────────

兩邊各自維護重疊的                ──→   AI pipeline 只放在                ──→  避免重複維護與版本不同步
AI pipeline                             InternalAICopliot

私有理論 prompt 汙染              ──→   LinkChat 不持有                   ──→  產品模組不受理論細節污染
LinkChat 產品模組                        私有 prompt / mapping

structured fact、理論解釋         ──→   事實資料 vs 理論解釋              ──→  各專案可獨立演化
與 AI transport 混在一起                 vs AI transport 分屬不同專案

調整理論需同時改兩邊              ──→   理論修改集中在                    ──→  改理論只需動一個地方
                                        InternalAICopliot
```

這個切法也支援漸進式演化：

```text
先完成 LinkChat 資料端與整合端
       │
       ▼
穩定 LinkChat → InternalAICopliot 契約
       │
       ▼
擴充更多 traits system
       │
       ▼
擴充更多分析任務入口
```

## 未來方向

這套架構預計支援：

- `InternalAICopliot` 理論持續演化而不污染 LinkChat
- 未來增加 MBTI 等額外結構化系統
- 未來增加更多分析任務與輸入形式

## 文件邊界

- 本文件回答的是「系統怎麼切、資料流怎麼走」
- [PLAN.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/PLAN.md) 回答的是「為什麼現在這樣規劃、目前在哪個階段」
- [DEVELOPMENT.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/DEVELOPMENT.md) 回答的是「實際開發時要遵守什麼規則」
