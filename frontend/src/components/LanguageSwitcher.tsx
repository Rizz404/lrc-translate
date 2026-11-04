import { useTranslation } from "react-i18next";
import { SUPPORTED_LANGUAGES } from "../i18n";

export function LanguageSwitcher() {
  const { t, i18n } = useTranslation();
  const current = i18n.resolvedLanguage;

  return (
    <div
      role="group"
      aria-label={t("common.language")}
      className="absolute top-3 right-3 z-30 inline-flex rounded-lg border border-neutral-800 bg-neutral-900/60 p-0.5 text-xs backdrop-blur"
    >
      {SUPPORTED_LANGUAGES.map((l) => (
        <button
          key={l.code}
          onClick={() => i18n.changeLanguage(l.code)}
          aria-pressed={current === l.code}
          title={l.label}
          className={`rounded-md px-2 py-1 font-medium uppercase transition-colors ${
            current === l.code ? "bg-primary-600 text-white" : "text-neutral-400 hover:text-white"
          }`}
        >
          {l.code}
        </button>
      ))}
    </div>
  );
}
