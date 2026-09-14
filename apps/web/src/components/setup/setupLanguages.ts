// Design ref: ux-design.pen Screen N1-D (dzgq9) · N5-D (CWh3E)
// One list for both ends of the wizard: N1 picks a language, N5 reads it back.
// N5 shows the NAME (繁體中文), never the code (zh-TW) — the summary is read by
// the person who just picked it, not by the API.
export const SETUP_LANGUAGES = [
  { code: 'zh-TW', label: '繁體中文' },
  { code: 'en', label: 'English' },
  { code: 'ja', label: '日本語' },
] as const;

export function languageLabel(code: string): string {
  return SETUP_LANGUAGES.find((lang) => lang.code === code)?.label ?? code;
}
