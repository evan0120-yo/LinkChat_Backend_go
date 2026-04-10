# Profile 模組

## 模組目的

`profile` 是 LinkChat 在目前 `auth + link` 基底上新增的人物背景資料模組。

第一版主要負責：

- 保存人物補充三句
- 保存人物 tag 選擇
- 提供 tag catalog 查詢
- 提供組裝分析前需要的 subject profile context

## 目前責任

- 驗證人物補充三句規則
- 驗證 tag catalog 與 single / multi selection rule
- 以 `subject_profiles` 保存 owner 對 linked subject 的人物背景資料
- 以 `profile_tag_groups`、`profile_tags` 保存可用 tag catalog

## 不負責的事情

- 不負責 InternalAICopliot 呼叫
- 不負責 prompt fragment 與理論 mapping
- 不把 notes 或 tags 混進原始 user text

## 與現有模組關係

- 透過 `link usecase/query` 驗證 subject 是否為 active linked user
- 不直接依賴 `link` 的 repository 或 service
- 後續 `copilot integration` 應透過 `profile usecase/query` 取人物背景資料
