// Workspace locale (UI-16 … UI-19).
//
// Static, user-visible copy goes through t(); the English string is the key
// and zh-TW is looked up in the dictionary. Data values (Project names,
// repository paths, ids, commits, digests, evidence refs, user input) are
// never passed through t(), so switching the locale cannot translate or
// pollute them. Machine status codes get a human label per locale through
// statusLabel(); the code itself stays visible next to the label so the
// technical detail can always be read back (UI-18).
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { zhTW } from "./zh-TW";

export type Locale = "en" | "zh-TW";
export const locales: Locale[] = ["en", "zh-TW"];
export const localeStorageKey = "context-rail.locale";

const dictionaries: Record<Locale, Record<string, string>> = { en: {}, "zh-TW": zhTW.messages };
const statusLabels: Record<Locale, Record<string, string>> = { en: statusLabelsEN(), "zh-TW": zhTW.statuses };

interface LocaleValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (message: string, values?: Record<string, string | number>) => string;
  statusLabel: (code: string) => string | null;
}

const LocaleContext = createContext<LocaleValue>({ locale: "en", setLocale: () => undefined, t: (message, values) => interpolate(message, values), statusLabel: (code) => statusLabels.en[code] ?? null });

function readStoredLocale(): Locale {
  try {
    const stored = globalThis.localStorage?.getItem(localeStorageKey);
    if (stored === "zh-TW" || stored === "en") return stored;
  } catch {
    // storage unavailable (private mode, tests): fall back to English
  }
  return "en";
}

function interpolate(message: string, values?: Record<string, string | number>): string {
  if (!values) return message;
  return message.replace(/\{(\w+)\}/g, (match, key: string) => (key in values ? String(values[key]) : match));
}

/** Translate a static message; falls back to the English key when zh-TW has no entry. */
export function translate(locale: Locale, message: string, values?: Record<string, string | number>): string {
  const translated = dictionaries[locale][message] ?? message;
  return interpolate(translated, values);
}

export function LocaleProvider({ children, initial }: { children: ReactNode; initial?: Locale }) {
  const [locale, setLocaleState] = useState<Locale>(initial ?? readStoredLocale());
  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
    try {
      globalThis.localStorage?.setItem(localeStorageKey, next);
    } catch {
      // storage unavailable: the choice lives for this page only
    }
  }, []);
  useEffect(() => {
    if (typeof document !== "undefined") document.documentElement.lang = locale;
  }, [locale]);
  const value = useMemo<LocaleValue>(() => ({
    locale,
    setLocale,
    t: (message, values) => translate(locale, message, values),
    statusLabel: (code) => statusLabels[locale][code] ?? null,
  }), [locale, setLocale]);
  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>;
}

export function useLocale(): LocaleValue {
  return useContext(LocaleContext);
}

/** Human labels for machine status codes in English (UI-18). */
function statusLabelsEN(): Record<string, string> {
  return {
    READY: "Ready", NEEDS_INPUT: "Needs input", BLOCKED: "Blocked", STALE: "Stale", CURRENT: "Current", DERIVED: "Derived", MISSING: "Missing",
    CONFLICT: "Conflict", UNKNOWN: "Unknown", UNDECLARED: "Undeclared", PARTIAL: "Partial", PASS: "Pass", FAIL: "Fail", WAIVED: "Waived",
    ACTIVE: "Active", DRAFT: "Draft", PAUSED: "Paused", RETIRED: "Retired", MATERIAL: "Material", INFORMATIONAL: "Informational",
    DECISION_READY: "Ready to decide", ACCEPTED_FOR_DEVELOPMENT: "Accepted for development", REJECTED: "Rejected", ISSUED: "Issued",
    ACCEPTED: "Accepted", ACCEPTED_FOR_STAGING: "Accepted for staging", CANDIDATE_REQUIRES_HUMAN_ACCEPTANCE: "Candidate requires human acceptance",
    CANDIDATE_ACCEPTABLE: "Candidate acceptable", NEEDS_EVIDENCE: "Needs evidence", NEEDS_REVIEW: "Needs review", ACCEPTED_FOR_PROMOTION: "Accepted for promotion",
    CANDIDATE_ACCEPTED: "Candidate accepted", CANDIDATE_SUBMITTED: "Candidate submitted", EVALUATED: "Evaluated", DEFERRED_TO_PROMOTION: "Deferred to promotion",
    GATE_BLOCKED: "Gate blocked", AWAITING_BUILD: "Awaiting build", READY_FOR_APPROVAL: "Ready for approval", APPROVED: "Approved", PROMOTED: "Promoted",
    PROMOTION_FAILED: "Promotion failed", PROMOTION_SIMULATED: "Promotion simulated", REPLAYED: "Replayed", NEEDS_APPROVAL: "Needs approval",
    READ_ONLY_BLOCKED_IN_P0: "Read-only, blocked in P0", RECOMMENDED: "Recommended", UNVERIFIED: "Unverified",
  };
}
