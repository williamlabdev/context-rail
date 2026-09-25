// Reader-language wording for RuleAdvisor's fixed sentences, keyed by the
// code the API sends (internal/change/advisor.go). English is the recorded
// text itself, so only other locales need entries. Anything without a code —
// a model's option, a human's words — is shown exactly as recorded.
import type { Locale } from "./index";

interface OptionWording { title: string; summary: string; cost_drivers: string[]; risks: string[] }

const optionsZh: Record<string, OptionWording> = {
  minimal_reversible_slice: {
    title: "最小、可退回的一步",
    summary: "只在允許的檔案路徑內實作已核准的範圍，不新增基礎設施、資料儲存或權限（IAM）變更；先在目標環境驗證，再考慮加其他東西。",
    cost_drivers: ["現有服務的運算資源", "建置與測試時間"],
    risks: ["單靠這一步可能無法滿足完整的業務需求", "若這一步不保存狀態，資料會在重啟時重置"],
  },
  defer: {
    title: "暫緩／維持現行流程",
    summary: "現在不改系統；維持人工流程，等未知問題釐清後再評估。",
    cost_drivers: ["持續的人工作業（未量測）"],
    risks: ["等待對業務的影響尚未量化"],
  },
  managed_storage_with_signed_urls: {
    title: "託管物件儲存＋短效簽名網址",
    summary: "檔案存在私有 bucket，服務先授權、再發出短效簽名網址；要先知道保存期限、區域與存取模式，才能估算成本。",
    cost_drivers: ["保存期間的儲存量", "下載流量（egress）", "中繼資料讀寫"],
    risks: ["網址可能被重複使用，撤銷能力有限", "IAM 與 bucket 政策必須在 agent 的範圍之外建立"],
  },
  expanded_scope_new_infrastructure: {
    title: "擴大範圍並新增基礎設施",
    summary: "用新服務或新的資料儲存解決更大的需求；核准前需要另一份架構決策、成本證據與 IAM 計畫。",
    cost_drivers: ["新的託管服務", "維運與值班負擔"],
    risks: ["範圍蔓延", "規模估算前成本未知", "要更久才拿得到第一份證據"],
  },
};

const notesZh: Record<string, (value: string) => string> = {
  constraint_not_declared: (value) => `未宣告 ${value}，成本狀態未知`,
  project_context_not_current: (value) => `專案情境為 ${value}，部分架構事實可能已過時`,
  repository_unverified: () => "儲存庫網址是測試用的佔位值；實際的遠端與分支保護尚未驗證",
  no_out_of_scope_list: () => "沒有明確的「不包含」清單；agent 的邊界只靠允許路徑（allowed_paths）與禁止動作（forbidden_actions）",
};

export function optionWording<T extends { title: string; summary: string; cost_drivers?: string[]; risks?: string[] }>(
  option: T, code: string | undefined, locale: Locale,
): T {
  const wording = code && locale === "zh-TW" ? optionsZh[code] : undefined;
  if (!wording) return option;
  const translated: T = { ...option, title: wording.title, summary: wording.summary };
  // Lists are replaced only when they still match the known wording's shape.
  if (option.cost_drivers && option.cost_drivers.length === wording.cost_drivers.length) translated.cost_drivers = wording.cost_drivers;
  if (option.risks && option.risks.length === wording.risks.length) translated.risks = wording.risks;
  return translated;
}

export function noteWording(note: { text: string; code?: string; value?: string }, locale: Locale): string {
  const render = note.code && locale === "zh-TW" ? notesZh[note.code] : undefined;
  return render ? render(note.value ?? "") : note.text;
}
