# LinkChat 開發文件

## 文件目的

本文件定義 LinkChat 專案的開發規範、架構原則、模組責任、程式設計準則與未來擴充方向。

這份文件的用途不是取代：

- [PLAN.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/PLAN.md)
- [ARCHITECTURE.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/ARCHITECTURE.md)

而是補上「實際開發時要遵守什麼規則」。

本文件是三份主文件中最偏向實作規範的文件。

若出現重複描述時，原則如下：

- `PLAN.md` 以規劃與階段目標為主
- `ARCHITECTURE.md` 以模組切分與資料流為主
- `DEVELOPMENT.md` 以開發規範與實作邊界為主

若三者對開發規則有重疊，應優先以本文件為準。

## 專案定位

LinkChat 目前不是以完整公開社交產品為第一優先，而是先轉型成一個個人使用的 AI 輔助分析 backend。

這個專案現階段的重點是：

- 建立可演化的 AI backend 架構
- 累積 LLM integration 經驗
- 累積 retrieval / RAG 經驗
- 支援未來多 provider 擴充
- 保留未來回到正式產品方向的空間

## 技術基底

目前專案主要技術如下：

- 語言：Go
- Web framework：Gin
- 資料庫：Firestore
- 驗證：JWT
- 部署方向：Cloud Run

Go 保留的理由：

- 啟動速度快
- 記憶體負擔較小
- 適合 Cloud Run 的低成本、scale-to-zero 模式
- 適合 request-driven backend

因此開發時應盡量遵守：

- 啟動輕量
- 避免不必要的全域重物件初始化
- 模組設計以 request-driven 為主
- 避免為了抽象而引入過多框架式負擔

## 專案分層原則

目前專案主要延續既有的模組化與分層風格：

- `provider`
- `handler`
- `usecase`
- `service`
- `repository`
- `middleware`
- `validator`
- `seeder`
- `model`
- `object`

## CQRS 與跨模組溝通規則

本專案目前採用明確的 CQRS 思維，但只套用在：

- `usecase`
- `service`

`repository` 不再額外切成 command repository 與 query repository，repository 維持以資料存取為主的單一層。

### CQRS 套用位置

```text
                      CQRS 套用範圍圖
┌─────────────────────────────────────────────────┐
│                                                 │
│  ✔ 套用 CQRS                                    │
│  ┌───────────────────────────────────────────┐  │
│  │  usecase/                                 │  │
│  │    ├── command                             │  │
│  │    └── query                              │  │
│  │                                           │  │
│  │  service/                                 │  │
│  │    ├── command                             │  │
│  │    └── query                              │  │
│  └───────────────────────────────────────────┘  │
│                                                 │
│  ✘ 不套用 CQRS                                   │
│  ┌───────────────────────────────────────────┐  │
│  │  repository/                              │  │
│  │    └── 維持單一層，不拆成 command / query   │  │
│  │       只專注 persistence 與 query 存取     │  │
│  └───────────────────────────────────────────┘  │
│                                                 │
└─────────────────────────────────────────────────┘
```

### 模組間溝通原則

目前模組間先不使用 MQ、event bus 或 async broker 進行切分。

也就是說，現階段跨模組溝通以應用層直接呼叫為主，不先引入：

- message queue
- event streaming
- broker-based integration

原因：

- 目前專案仍在快速演化與驗證階段
- 當前重點是 LinkChat 與 `InternalAICopliot` 的分工以及模組責任清楚
- 過早引入 MQ 會增加複雜度與維護成本

### Command 溝通規則

```text
跨模組 Command 呼叫鏈

  ✔ 允許                                  ✘ 禁止
  ──────────────────────                  ──────────────────────

  Module A                Module B        Module A                Module B
  ┌──────────┐           ┌──────────┐    ┌──────────┐           ┌──────────┐
  │ usecase/ │           │ usecase/ │    │ service/ │     ✘     │ service/ │
  │ command  │ ───────→  │ command  │    │ command  │ ──╳───→   │ command  │
  └──────────┘           └──────────┘    └──────────┘           └──────────┘

                                          Module A                Module B
                                          ┌──────────┐           ┌──────────┐
                                          │ usecase/ │     ✘     │repository│
                                          │ command  │ ──╳───→   │          │
                                          └──────────┘           └──────────┘
```

範例：`auth` 建立 user 後同步 `link` 時，應由 `auth usecase` 呼叫 `link usecase`，不應由 `auth service` 直接呼叫 `link service`。

### Query 溝通規則

```text
跨模組 Query 呼叫鏈

  ✔ 允許                                  ✘ 不建議
  ──────────────────────                  ──────────────────────

  Module A                Module B        Module A                Module B
  ┌──────────┐           ┌──────────┐    ┌──────────┐           ┌──────────┐
  │ usecase/ │           │ usecase/ │    │  任意層   │     ✘     │ service/ │
  │  query   │ ───────→  │  query   │    │          │ ──╳───→   │  query   │
  └──────────┘           └──────────┘    └──────────┘           └──────────┘
```

原因：

- query 也可能碰到資料分級與欄位暴露風險
- 讓 `usecase` 成為對外查詢契約，邊界較清楚
- 可避免 internal structured data 被下游模組誤當成前端輸出

範例：`copilot integration` 若要取得 `traits / profile` 的分析用資料，應呼叫 `traits / profile usecase/query`，不應直接呼叫 `traits / profile service/query`。

### Command 與 Query 的實作原則

#### Command

- 負責改變狀態
- 負責交易邊界
- 跨模組時只能走 `usecase -> usecase`

#### Query

- 負責讀取與整合資料
- 跨模組時也應走 `usecase -> usecase`
- `service/query` 主要保留在 module 內部使用

### 這條規則的目的

這樣切的目的是：

- 保持 command 邊界清楚
- 保持 query 邊界也清楚
- 避免狀態修改繞過應用層
- 避免敏感欄位或 internal data 被錯誤外露
- 降低跨模組 side effect 失控
- 讓 module 對外契約集中在 usecase

### 各層意義

#### `provider`

負責組裝模組依賴。

用途：

- 建立 repository
- 建立 service
- 建立 usecase
- 建立 handler
- 對外暴露模組入口

原則：

- 不寫業務邏輯
- 不寫複雜流程控制
- 只做 DI 與 wiring

#### `handler`

負責 HTTP request / response 轉換。

用途：

- 接 request
- 做 binding
- 呼叫 usecase
- 回傳 HTTP response

原則：

- 不寫業務規則
- 不直接查資料庫
- 不直接組 AI context

#### `usecase`

負責應用流程控制。

用途：

- 串接多個 service / query / module
- 執行交易邊界
- 決定業務流程順序

原則：

- 可以協調多個模組
- 可以呼叫多個 service
- command 類跨模組溝通只能透過 `usecase`
- 不直接承擔 persistence 細節

#### `service`

負責單一領域行為或邏輯。

用途：

- domain action
- helper logic
- 驗證邏輯
- query facade

原則：

- 聚焦單一責任
- 不處理 HTTP 層
- 不混入過多流程 orchestration
- 不應作為跨模組 command 的直接入口

#### `repository`

負責資料存取。

用途：

- 封裝 Firestore 或未來其他 storage 的 CRUD / query

原則：

- 不寫業務判斷
- 不做 AI 組裝
- 不處理 request validation
- 不做 CQRS 切分

#### `middleware`

負責 request 進入 handler 前的通用攔截處理。

用途：

- 驗證 token
- 寫入 request context
- 執行基礎授權前置處理

原則：

- 不承擔主要業務邏輯
- 保持輕量與可重用
- 不直接負責複雜資料組裝
- 可作為跨模組重用的例外層，但只限 request/admission 類用途

#### `validator`

負責聚焦式驗證邏輯。

用途：

- 唯一性檢查
- 前置條件檢查
- 單點業務規則驗證

原則：

- 可依賴 query service
- 不直接改變系統狀態
- 驗證責任集中，避免散落在多層
- 預設只作為 module 內部驗證層，不作為跨模組公開入口

#### `seeder`

負責開發期或初始化資料植入。

用途：

- 建立開發資料
- 建立示範資料
- 驗證主要流程是否能正常運作

原則：

- 優先重用既有 usecase
- 不應繞過主要業務邏輯隨意寫資料，除非有明確理由
- 不視為正式 runtime 核心流程

## 核心模組定義

目前專案核心模組為：

- `auth`
- `link`
- `traits / profile`
- `copilot integration`

### `auth`

身份基礎模組，詳細角色請參考 [internal/auth/README.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/internal/auth/README.md)。

### `link`

關係與聯絡人基底模組，詳細角色請參考 [internal/link/README.md](/d:/WorkSpace/LinkChat/Backend/Go/LinkChat/internal/link/README.md)。

### `traits / profile`

保存分析所需的結構化資料。

負責：

- `theoryVersion`
- `slot -> code`
- 分析目標對象的 traits
- 未來其他固定 schema 的 profile system

規則：

- `LinkChat` 只保存結構化事實資料
- `LinkChat` 不保存私有理論 prompt
- `LinkChat` 不在本地解釋 `slot + code` 的理論含義

### `copilot integration`

這是 LinkChat 與 `InternalAICopliot` 的整合模組。

負責：

- 接收分析 request
- 驗證身份與必要欄位
- 載入 `auth` / `link` / `traits` 所需資料
- 整理固定 payload
- 呼叫 `InternalAICopliot`
- 回傳結果與處理 integration error mapping

規則：

- `copilot integration` 不持有私有理論 mapping
- `copilot integration` 不組最終 prompt
- `copilot integration` 不實作 RAG / retrieval
- `copilot integration` 的重點是資料整理與跨專案協調

## 目前推薦的資料流

在目前的設計下，建議流程如下：

```text
user request
    │
    ▼
LinkChat handler / middleware
    ├─ 驗證身份
    ├─ 驗證基本請求格式
    └─ 決定分析入口
    │
    ▼
copilot integration usecase
    ├─ 查 auth / link / traits
    ├─ 整理 theoryVersion + slot/code payload
    └─ 呼叫 InternalAICopliot
    │
    ▼
InternalAICopliot
    ├─ gatekeeper
    ├─ builder
    ├─ rag
    └─ output
    │
    ▼
LinkChat 收結果 -> 回前端
```

## 領域資料與 AI 資料的切法

### 應放在 LinkChat 的內容

- 已知 profile 欄位
- 關係狀態
- `theoryVersion`
- `slot -> code`
- 已確定的明確欄位資料

### 應放在 `InternalAICopliot` 的內容

- 私有理論 mapping
- prompt fragment
- 中性解釋文字
- RAG / retrieval
- AI provider 呼叫細節
- 理論補充文件

### 應放在 code 的內容

- request 驗證規則
- 權限規則
- deterministic mapping
- 不可模糊的商業規則

## 關於 AI 專案的特別開發準則

### 1. 不要讓 LinkChat 承擔私有理論語意

錯誤做法：

- 在 LinkChat 內決定 `sun + a1` 代表什麼
- 在 LinkChat 內把 `slot + code` 直接展開成 prompt 片段

正確做法：

- LinkChat 只整理固定資料契約
- `InternalAICopliot` 才做理論解釋與 prompt 組裝

### 2. 不要把跨專案契約和前端 response 混在一起

錯誤做法：

- 直接把 LinkChat 的內部 model 當成 `InternalAICopliot` request
- 直接把 `InternalAICopliot` response 原樣暴露而不做 mapping

正確做法：

- 在整合模組中維護明確的 request / response DTO
- 對跨專案契約做顯式 mapping

### 3. deterministic rule 優先寫在 code

若規則是明確、可驗證、不可模糊的，就不要只交給模型。

例如：

- 權限檢查
- `theoryVersion` 是否必填
- `slot` / `code` 是否存在
- 哪些欄位允許送往 `InternalAICopliot`

### 3-1. 敏感資料模組的對外查詢必須收斂

像 `traits / profile` 這種可能同時持有：

- 給人看的 display data
- 給 `InternalAICopliot` 的 internal codes

的模組，跨模組查詢時應只透過 `usecase/query` 對外提供查詢契約。

不要依賴平行模組直接呼叫其 `service/query` 來取得 raw data。

這樣可以降低：

- internal code 被誤傳到前端
- 下游模組直接把 traits/profile model 當 response model 使用
- query 邊界逐漸失控

### 4. LinkChat 與 `InternalAICopliot` 的責任不能重疊

- LinkChat 負責資料與入口
- `InternalAICopliot` 負責理論、prompt、RAG、AI provider

若兩邊都維護同一套 AI pipeline，之後文件、實作與理論更新一定會互相衝突。

### 5. 先求可跑，再求完美抽象

這個專案目前處在逐步演化階段。

因此：

- 第一版可以先做薄版 module
- 但命名與責任邊界要先對
- 不要一開始就在 LinkChat 再造一套完整 AI pipeline

## 檔案與命名規範

### 命名方向

- 模組名稱使用短且明確的英文
- 對外概念名稱應優先和業務責任對齊
- 避免用太技術導向、未來會限制擴充的名稱

目前已定名：

- `auth`
- `link`
- `traits` 或 `profile`
- `copilot` 或 `consult`

### README 規範

每個主要模組目錄底下應有 `README.md`，說明：

- 模組目的
- 主要責任
- 不負責的事情
- 演化方向

### 物件放置規範

- request / response DTO 放 `object/req`、`object/resp`
- 模組內部共用 DTO 可放 `object/dto`
- domain model 放 `model`

## 新功能開發流程建議

新增一個新功能時，建議依照以下順序思考：

1. 這個功能屬於哪個模組
2. 這個功能是 LinkChat 結構化資料，還是 `InternalAICopliot` 理論能力
3. 這個邏輯是 deterministic rule 還是 AI interpretation
4. 這個流程應該落在 handler、usecase、service、repository 哪一層
5. 這個功能是否會讓 LinkChat 承擔本不該承擔的 AI 語意責任

如果發現：

- 一段邏輯同時碰 request 驗證、資料查詢、知識檢索、AI 呼叫

就代表切層可能有問題，需要重新整理責任。

## 未來擴充規範

### 多 trait system

未來可能增加：

- zodiac
- MBTI
- 其他自訂 traits

原則：

- LinkChat 保存結構化欄位與版本
- `InternalAICopliot` 持有理論解釋與 prompt fragment

### 多分析任務

未來可能增加：

- 關係分析
- 團隊融入
- 合作判斷

原則：

- LinkChat 決定入口與必要資料
- `InternalAICopliot` 決定分析語意與輸出組裝

## 現階段禁止事項

現階段請避免：

- 在 LinkChat 內保存完整私有 prompt
- 在 LinkChat 內實作私有理論 mapping
- 把 `slot + code` 直接展開成給模型的最終語句
- 把 `InternalAICopliot` 的責任重新複製回 LinkChat
- 把 `validator` 當成跨模組查詢入口
- 直接把平行模組的 `service/query` 當公開 API 使用
- 因為單一 use case 就把模組命名綁死
- 為了想像中的未來過度設計

## 現階段鼓勵事項

現階段應優先：

- 先打通最小 LinkChat -> `InternalAICopliot` request flow
- 保持 module boundary 清楚
- 讓文件與程式碼一起演進
- 新模組先做薄版 skeleton
- 保住固定的跨專案資料契約

## 結語

LinkChat 現階段的核心價值，不是自己成為 AI 編排中心，而是成為穩定的資料來源與分析入口，並和 `InternalAICopliot` 清楚分工。

因此開發時最重要的不是一次把所有能力做完，而是：

- 每一層責任清楚
- 每個模組邊界穩定
- 每次新增功能都能放在合理的位置

這份文件未來應和專案一起更新，而不是成為一次性文件。
