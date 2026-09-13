# ContextRail UI 的 Figma 匯入說明

版本 1.0｜2026-09-12

## 可以匯入嗎？

可以。`output/figma/context-rail-project-crud.svg` 是 ContextRail Project CRUD 的不依賴外部資源 SVG 向量檔，直接將檔案拖曳到 Figma 畫布即可匯入。

檔案包含三個靜態畫面：

1. Project Registry：Project 的建立、查詢、進入 Workspace、編輯與封存入口。
2. Project Workspace：Project 內的 Ticket、Environment Topology、Decision Gate 與 SSOT。
3. Project Settings：Project identity、policy 版本、環境 target 與 archive 語義。

互動 mockup 的進入路徑是：先點左側「專案管理」，再在 active Project 列按「進入 Workspace」。視覺稿也把 Project 名稱做成快捷入口；封存的 Project 不提供新 Workspace，只能「查看歷史」。

## 匯入後會保留什麼？

- 文字仍可在 Figma 中編輯。
- 矩形、線條、顏色與狀態標籤會以向量物件匯入。
- `frame-project-registry`、`frame-project-workspace` 與 `frame-project-settings` 是三個主要畫面群組，可分開整理。

SVG 匯入不會自動轉換成完整的 Figma component、Auto Layout、Prototype interaction 或 Design Tokens。這份檔案的用途是先固定資訊架構、畫面層級與操作語義；進入正式產品設計時，再在 Figma 內將按鈕、狀態、表格列、環境節點與 Project card 抽成元件。

## 設計邊界

UI 只展示 Project governance，不延伸成 Jira、完整多租戶管理、Billing 或 IaC 編輯器。`Delete` 的介面文案固定使用「封存並保留證據」，以避免使用者誤以為歷史 DecisionRecord、Evidence Bundle 或 Release Receipt 會被刪除。
