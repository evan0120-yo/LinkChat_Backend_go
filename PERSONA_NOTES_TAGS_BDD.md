# LinkChat Persona Notes And Tags BDD

## 範圍

- 本文件描述即將開發的 LinkChat 人物補充三句與 tag 功能。
- 這份 BDD 是 items 1、2 的目標驗收文件。
- 本次在 LinkChat Go backend 已落地的範圍是 LinkChat 端保存、查詢、catalog 與 access rule。
- InternalAICopliot 相關 renderer 與最終 profile consult 對接，仍屬跨 repo 後續驗收。
- 收費、解鎖、訂閱與權限切分不在本文件範圍內。

## Feature: 人物補充三句

### Scenario: 使用者可以為同一個對象保存最多三條短句

Given 使用者已選定一個分析對象

And 該對象允許編輯人物補充資料

When 使用者提交 0 到 3 條短句

Then 系統會保存這些短句作為該對象的人物補充資料

And 每條短句都會先做 trim

And 空白字串不會被保存

And 保存後的順序必須與使用者輸入順序一致

### Scenario: 超過三條短句時請求失敗

Given 使用者已選定一個分析對象

When 使用者提交 4 條以上非空短句

Then 系統應拒絕保存

And response 應明確指出超過三條上限

And 不得發生部分寫入

### Scenario: 單條短句過長時請求失敗

Given 系統規定單條短句上限為 60 字

When 使用者提交任一條超過上限的短句

Then 系統應拒絕保存

And response 應指出是哪一條短句超長

And 不得發生部分寫入

### Scenario: 系統不自動合併或改寫短句內容

Given 使用者提交合法的三條短句

When 系統完成保存

Then 系統只應做 trim 與空值移除

And 不應自動摘要、重寫、翻譯或合併短句

## Feature: 人物 tag

### Scenario: 使用者可以為對象選擇分組 tag

Given LinkChat 已配置可用的 tag group 與 tag definition

And tag 至少包含 `groupKey`、`tagKey`、`label`

When 使用者為對象選擇 tag

Then 系統應保存 canonical key 組合

And 保存資料應以 `groupKey + tagKey` 為主

And `label` 只作為 LinkChat UI 顯示用途

### Scenario: single-select 群組一次只能選一個 tag

Given 某個 tag group 的 selection mode 是 `single`

When 使用者在同一群組提交 2 個以上 tag

Then 系統應拒絕保存

And response 應指出該群組只允許單選

### Scenario: multi-select 群組可以同時選多個 tag

Given 某個 tag group 的 selection mode 是 `multi`

When 使用者在同一群組提交多個合法 tag

Then 系統應允許保存

And 後續分析請求應保留全部選擇

### Scenario: 未知或 inactive 的 tag 不可保存

Given 使用者提交的 tag 不存在於目前可用 catalog

Or 該 tag 已被標成 inactive

When 系統驗證 tag 請求

Then 系統應拒絕保存

And 不得發生部分寫入

### Scenario: 重複 tag 只保存一份

Given 使用者重複提交相同 `groupKey + tagKey`

When 系統完成正規化

Then 最終保存結果中同一個 tag 只能存在一次

## Feature: LinkChat Profile API

### Scenario: 只有 active linked user 可以作為可編輯對象

Given 使用者已登入 LinkChat

When 使用者嘗試保存或查詢某個對象的人物資料

Then 該對象必須是目前 active 的 linked user

And owner 不可把自己當成 subject

### Scenario: LinkChat 提供獨立的 notes 與 tags 保存入口

Given 使用者已登入 LinkChat

When 使用者保存人物補充三句

Then LinkChat 應提供 `PUT /citrus/profiles/notes`

And request body 應帶 `subjectId` 與 `lines`

When 使用者保存人物 tag

Then LinkChat 應提供 `PUT /citrus/profiles/tags`

And request body 應帶 `subjectId` 與 `selected`

### Scenario: LinkChat 可以查詢人物背景資料與 tag catalog

Given 使用者已登入 LinkChat

When 使用者查詢某個對象的人物背景資料

Then LinkChat 應提供 `GET /citrus/profiles/context?subjectId=...`

And 回傳資料應包含 `subjectId`、`noteLines`、`selectedTags`

When 使用者查詢可用 tag catalog

Then LinkChat 應提供 `GET /citrus/profiles/tag-catalog`

And 回傳資料應以 group 為單位列出可用 tags

## Feature: LinkChat 組裝分析請求

以下 scenario 屬於跨 repo 對接驗收，不在本次 LinkChat Go backend code 驗收範圍。

### Scenario: 發問時把人物資料與本次問題分開傳送

Given 某個對象已經有星座資料

And 該對象也有補充三句與 tag

And 使用者輸入一段本次真正想問的問題

When 使用者送出分析請求

Then LinkChat 應將本次問題放進 `text`

And LinkChat 不得把補充三句或 tag label 直接串進 `text`

And LinkChat 應將人物補充與 tag 以 `subjectProfile.analysisPayloads` 傳送

### Scenario: notes 與 tags 應作為獨立 analysis payload 傳送

Given 使用者送出分析請求

When LinkChat 組裝 `subjectProfile`

Then payload 應至少允許以下 analysis type：

And `astrology`

And `persona_notes`

And `persona_tags`

And 每個 analysis payload 都只攜帶自己的 canonical 資料

### Scenario: 沒有資料的 payload 不應送空殼

Given 某個對象沒有補充三句

And 某個對象沒有 tag

When LinkChat 組裝分析請求

Then LinkChat 應省略 `persona_notes` 或 `persona_tags` payload

And 不應送出空陣列占位用 payload

## Feature: InternalAICopliot 處理 persona_notes

以下 scenario 屬於 InternalAICopliot repo 驗收，不在本次 LinkChat Go backend code 驗收範圍。

### Scenario: persona_notes 以 subject profile context 進入 prompt

Given InternalAICopliot 收到 `analysisType=persona_notes`

And payload 內容為已正規化的短句陣列

When builder 組裝 LinkChat profile prompt

Then 這些短句應進入 subject profile block

And 順序必須與 LinkChat 傳入順序一致

And 不得被併入 `[RAW_USER_TEXT]`

### Scenario: persona_notes 不應被當成使用者本次指令

Given 某條補充短句本身帶有命令語氣

When InternalAICopliot 組裝 prompt

Then 該內容仍應被當成人物背景資料

And 不應改變 `text` 的原始使用者輸入語意

## Feature: InternalAICopliot 處理 persona_tags

### Scenario: persona_tags 以 canonical key 做內部映射

Given InternalAICopliot 收到 `analysisType=persona_tags`

And payload 只包含 `groupKey` 與 `tagKey`

When builder 進入 LinkChat strategy

Then InternalAICopliot 應以 canonical key 對應內部 prompt fragment

And 不依賴 LinkChat 的顯示 label 做語意判斷

### Scenario: tag 可以影響回覆風格

Given 對象帶有 `role=student`

When AI 產生回覆

Then 回覆應更傾向教學式、步驟式、較容易被理解的說法

And 最終回覆不得直接暴露 internal fragment key 或 prompt 名稱

### Scenario: 未映射的 tag 應在進 AI 前失敗

Given LinkChat 傳來的某個 tag key 在 InternalAICopliot 沒有對應 fragment

When builder 進行 tag render

Then 系統應在 AI 呼叫前直接回錯

And 錯誤應明確表示 tag mapping 缺失

## Feature: 最終人物分析請求契約

### Scenario: 完整 payload 同時帶有星座、補充三句與 tag

Given 某個對象同時具備三類資料

When LinkChat 呼叫 InternalAICopliot 的 profile consult

Then request 應符合以下形狀：

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

And `text` 應只代表這次真正的問題

And `subjectProfile` 應只代表被分析對象的背景資料

## Out Of Scope

- 收費、解鎖、訂閱 gating
- 自動從長文抽 tag
- 自由輸入自訂 tag label
- 讓 InternalAICopliot 保存 LinkChat 的人物原始資料
