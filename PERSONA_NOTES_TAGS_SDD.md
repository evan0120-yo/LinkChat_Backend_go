# LinkChat Persona Notes And Tags SDD

## 文件目的

本文件描述 LinkChat 人物補充三句與 tag 功能設計，以及目前已在 LinkChat Go backend 落地的第一版實作。

這份文件對應：

- LinkChat/預期想法.md 中的第 1 項「三條可傳入文字」
- LinkChat/預期想法.md 中的第 2 項「tag」

目前狀態切分如下：

- LinkChat Go backend 已落地：
  - notes 保存
  - tags 保存
  - profile context 查詢
  - tag catalog 查詢
  - active linked subject access rule
- 尚未在本 repo 落地：
  - LinkChat -> InternalAICopliot profile consult integration
  - InternalAICopliot 的 `persona_notes` / `persona_tags` renderer

## 產品定位

LinkChat 這條線的目標不是做交友配對，也不是讓 AI 直接替使用者聊天。

這次設計要強化的是：

- 讓使用者更完整描述某個對象
- 讓 AI 在分析時不只看星座
- 讓 tag 能逐步演化成互動風格與建議策略的控制項

因此這次新增的兩個功能，本質上都是「人物背景資料」：

- 補充三句
- tag

而不是：

- 本次使用者問題
- Internal 內部 prompt
- Internal 內部理論資料

## 核心設計決策

```text
使用者在 LinkChat 編輯人物資料
│
├─ 補充三句
│  └─ 屬於人物背景
│
├─ tag
│  └─ 屬於人物背景
│
└─ 本次發問文字
   └─ 屬於這一次真的想問 AI 的問題

送往 InternalAICopliot 時
│
├─ text
│  └─ 只放本次問題
│
└─ subjectProfile.analysisPayloads
   ├─ astrology
   ├─ persona_notes
   └─ persona_tags

責任分工
│
├─ LinkChat
│  ├─ 保存人物補充三句
│  ├─ 保存 tag 選擇
│  ├─ 保存 tag 的 UI label / group
│  └─ 組裝對外 request
│
└─ InternalAICopliot
   ├─ 將 notes render 成 profile context
   ├─ 將 tag key 映射成 prompt fragment
   ├─ 組裝最終 prompt
   └─ 呼叫 AI
```

## 為什麼不能把 notes 與 tag 直接塞進 text

這是本次最重要的拒絕方案。

若把補充三句與 tag 直接串進 `text`，會產生以下問題：

- `text` 在 Internal 內目前被視為原始使用者輸入，而不是人物背景資料
- `text` 會先進 `[RAW_USER_TEXT]` 安全檢查區塊
- 補充三句與 tag 會和本次問題混在一起，語意邊界變差
- 後續若要加上 tag skill、版本化或更細的 renderer，幾乎都要重拆
- 使用者若在補充三句中寫出命令語氣，會增加 prompt 判讀混亂

因此本次明確決策：

- `text` 只放本次發問
- 補充三句走 `analysisType=persona_notes`
- tag 走 `analysisType=persona_tags`

## 範圍

### In Scope

- 對象補充三句資料模型
- tag group / tag definition / subject tag selection
- LinkChat -> InternalAICopliot profile consult 契約
- Internal 端 notes/tag renderer 設計
- Prompt 邊界與錯誤策略

### Out Of Scope

- 收費、解鎖與訂閱
- 自動從長文抽 tag
- 使用者自建任意 tag
- 長篇人物自由描述
- LinkChat 本地實作完整 AI pipeline

## 邊界分工

### 應放在 LinkChat 的內容

- 被分析對象的補充三句原文
- 被分析對象的 tag 選擇結果
- tag 的顯示 label
- tag group 的顯示名稱與選取規則
- 這次要送哪幾個 analysis payload

### 應放在 InternalAICopliot 的內容

- `persona_notes` 如何排進 prompt
- `persona_tags` 對應的 prompt fragment
- tag 對回答風格的理論解釋
- source / rag / fragment 的實際組裝方式
- 最終 AI 呼叫

### 不應出現在 InternalAICopliot 的內容

- LinkChat UI label 作為主要判斷依據
- 被分析對象的原始 tag catalog 管理
- 把 LinkChat 的補充三句改存成 Internal 自己的本地人物資料

## 資料模型

### 1. Subject Persona Context

建議新增邏輯模型：

```json
{
  "ownerId": "user-001",
  "subjectId": "subject-001",
  "noteLines": [
    "慢熟，剛開始不太主動聊天",
    "壓力大時會先自己消化",
    "如果先給步驟，會比較願意配合"
  ],
  "selectedTags": [
    {
      "groupKey": "role",
      "tagKey": "student"
    },
    {
      "groupKey": "communication_style",
      "tagKey": "slow_warmup"
    }
  ],
  "updatedAt": "2026-03-31T15:00:00Z"
}
```

欄位說明：

- `ownerId`
  表示是哪個 LinkChat 使用者維護這份人物資料。

- `subjectId`
  表示被分析對象的識別值。第一版可直接對應現有 LinkChat 關係中的 target。

- `noteLines`
  最多三條，保留順序，不自動改寫。

- `selectedTags`
  保存 canonical key，不保存 prompt 語意。

### 2. Tag Group Definition

建議新增邏輯模型：

```json
{
  "groupKey": "role",
  "label": "角色",
  "selectionMode": "single",
  "active": true,
  "orderNo": 10
}
```

欄位說明：

- `groupKey`
  Internal 與 LinkChat 共用的穩定群組鍵。

- `label`
  只用於 LinkChat UI。

- `selectionMode`
  `single` 或 `multi`。

### 3. Tag Definition

建議新增邏輯模型：

```json
{
  "groupKey": "role",
  "tagKey": "student",
  "label": "學生",
  "active": true,
  "orderNo": 20
}
```

欄位說明：

- `groupKey + tagKey`
  構成 canonical identity。

- `label`
  只作為 LinkChat 顯示文字，不是 Internal 的主語意來源。

## 驗證規則

### 補充三句

- 最多三條非空短句
- 每條先 trim
- trim 後空字串直接移除
- 保留輸入順序
- 不自動去重
- 單條上限 60 字

### tag

- `groupKey` 與 `tagKey` 都必填
- `groupKey + tagKey` 必須存在於 active catalog
- 同一組合不可重複保存
- `single` 群組最多一個 tag
- `multi` 群組可多選

## LinkChat 對外契約

### Profile Consult Request

LinkChat 發分析請求時，應使用現有的 structured profile consult 模型，不額外把 notes/tag 串成字串。

完整範例：

```json
{
  "appId": "linkchat",
  "builderId": 101,
  "text": "我這次該怎麼跟他談作業？",
  "subjectProfile": {
    "subjectId": "subject-001",
    "analysisPayloads": [
      {
        "analysisType": "astrology",
        "theoryVersion": "astro_v1",
        "payload": {
          "sun_sign": ["capricorn"],
          "moon_sign": ["pisces"],
          "rising_sign": ["gemini"]
        }
      },
      {
        "analysisType": "persona_notes",
        "payload": {
          "lines": [
            "慢熟，剛開始不太主動聊天",
            "壓力大時會先自己消化",
            "如果先給步驟，會比較願意配合"
          ]
        }
      },
      {
        "analysisType": "persona_tags",
        "payload": {
          "selected": [
            {
              "groupKey": "role",
              "tagKey": "student"
            },
            {
              "groupKey": "communication_style",
              "tagKey": "slow_warmup"
            }
          ]
        }
      }
    ]
  }
}
```

契約規則：

- `text`
  只放本次發問。

- `persona_notes`
  只放已正規化的短句陣列。

- `persona_tags`
  只放 canonical `groupKey/tagKey`。

- 沒有資料的 analysis payload 直接省略，不送空殼。

## LinkChat 本地 API

目前已落地的能力切分如下：

- 補充三句有獨立保存入口
- tag 有獨立保存入口
- profile context 有獨立查詢入口
- tag catalog 有獨立查詢入口

第一版 access rule 固定為：

- `subjectId` 必須對應目前 active 的 linked user
- owner 不可把自己當成 subject

### 保存補充三句

```text
PUT /citrus/profiles/notes
```

request:

```json
{
  "subjectId": "subject-001",
  "lines": [
    "慢熟，剛開始不太主動聊天",
    "壓力大時會先自己消化",
    "如果先給步驟，會比較願意配合"
  ]
}
```

### 保存 tag

```text
PUT /citrus/profiles/tags
```

request:

```json
{
  "subjectId": "subject-001",
  "selected": [
    {
      "groupKey": "role",
      "tagKey": "student"
    },
    {
      "groupKey": "communication_style",
      "tagKey": "slow_warmup"
    }
  ]
}
```

### 取得可用 tag catalog

```text
GET /citrus/profiles/tag-catalog
```

### 取得人物背景資料

```text
GET /citrus/profiles/context?subjectId=subject-001
```

response:

```json
{
  "subjectId": "subject-001",
  "noteLines": [
    "慢熟，剛開始不太主動聊天",
    "壓力大時會先自己消化"
  ],
  "selectedTags": [
    {
      "groupKey": "role",
      "tagKey": "student"
    }
  ],
  "updatedAt": "2026-03-31T15:00:00Z"
}
```

## 本 repo 已落地的資料流程

```text
PUT /citrus/profiles/notes or /tags
│
├─ middleware 驗證 JWT
├─ handler 取出 ownerID 與 request body
├─ profile usecase 檢查 subject 是否為 active linked user
├─ validator 做 notes / tags normalization
├─ 讀取現有 subject_profiles
├─ merge 另一半既有資料
├─ 若 notes + tags 都空
│  └─ 刪除 subject_profiles 文件
└─ 否則 upsert subject_profiles
```

```text
GET /citrus/profiles/context
│
├─ middleware 驗證 JWT
├─ profile query usecase 檢查 subject 是否為 active linked user
└─ 回傳該 owner + subject 的 notes / selectedTags
```

```text
GET /citrus/profiles/tag-catalog
│
└─ 回傳目前 active 的 group 與 tags
```

## 跨 repo 後續對接

LinkChat server 端後續仍需要新增：

- 載入 subject 現有 astrology / noteLines / selectedTags
- 組裝 `subjectProfile`
- 呼叫 InternalAICopliot 的 profile consult

## InternalAICopliot 設計

### 1. 新增 analysis type

LinkChat strategy 第二層 factory 應新增：

- `persona_notes`
- `persona_tags`

第一層 routing 不變，仍由：

- `appId=linkchat`

進入 LinkChat strategy。

### 2. persona_notes renderer

`persona_notes` renderer 的責任：

- 讀取 `payload.lines`
- 驗證為 1 到 3 條合法短句
- 保留原始順序
- render 成 deterministic subject profile block

建議 render 結果：

```text
### [analysis:persona_notes]
note_1: 慢熟，剛開始不太主動聊天
note_2: 壓力大時會先自己消化
note_3: 如果先給步驟，會比較願意配合
```

這樣做的原因：

- deterministic
- 易測試
- 不混入 `RAW_USER_TEXT`
- 不要求 Internal 持有使用者筆記的長期存儲

### 3. persona_tags renderer

`persona_tags` renderer 的責任：

- 讀取 `payload.selected`
- 驗證每個 item 具備 `groupKey` 與 `tagKey`
- 依 canonical key 對應 internal fragment
- render 成 deterministic subject profile block

建議 payload item：

```json
{
  "groupKey": "role",
  "tagKey": "student"
}
```

建議 fragment lookup key：

```text
{groupKey}__{tagKey}
```

例如：

- `role__student`
- `communication_style__slow_warmup`

這個設計的理由：

- 避免不同 group 下出現相同 `tagKey` 時撞 key
- 可以直接沿用 Internal 現有 fragment source lookup 的思路
- 不需要把 LinkChat 的 UI label 送進 Internal

建議 source graph 形狀：

```text
moduleKey=persona_tags
│
├─ primary source
│  matchKey=role
│  prompts=角色提示
│
├─ fragment source
│  matchKey=role__student
│  prompts=這個對象更適合用教學式、步驟式、降低壓力的方式互動
│
├─ primary source
│  matchKey=communication_style
│  prompts=互動風格提示
│
└─ fragment source
   matchKey=communication_style__slow_warmup
   prompts=先暖身、先降低防備，再進主題
```

建議 render 結果：

```text
### [analysis:persona_tags]
role: 這個對象更適合用教學式、步驟式、降低壓力的方式互動
communication_style: 先暖身、先降低防備，再進主題
```

### 4. 錯誤策略

若 Internal 收到 `persona_tags` 但找不到 fragment mapping，應直接失敗，不應默默忽略。

理由：

- 忽略會讓 LinkChat 以為 tag 已生效
- 失敗更容易發現兩邊資料契約不同步
- 對第一版開發期更安全

建議錯誤語意：

- `PERSONA_TAG_MAPPING_NOT_FOUND`

## Prompt 邊界

最終 prompt 應維持以下分工：

```text
[RAW_USER_TEXT]
└─ 只放本次問題

[SUBJECT_PROFILE]
├─ astrology
├─ persona_notes
└─ persona_tags
```

這樣的好處：

- 使用者本次需求與人物背景分開
- 補充三句與 tag 不會被當成 raw user instruction
- 未來若要單獨調整 notes/tag renderer，不必動 `text` 語意

## 版本演進建議

### 第一版

- 補充三句固定最多三條
- tag 由後台預先配置 catalog
- LinkChat 組 payload
- Internal 新增兩個 renderer

### 第二版可延伸

- tag group 更細的權重或優先級
- tag 對 builder source 選擇的影響
- tag 專屬 theoryVersion
- 收費與解鎖 gating

## 與現有專案邊界的一致性

本設計維持既有方向：

- LinkChat 保存人物背景與入口資料
- InternalAICopliot 負責理論、prompt fragment、AI provider

因此本設計刻意避免：

- 在 LinkChat 保存 Internal 私有 prompt
- 在 Internal 保存 LinkChat 的人物原始資料庫
- 把 notes/tag 直接混成 `text`
