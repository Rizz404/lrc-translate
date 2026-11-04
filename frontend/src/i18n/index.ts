import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";
import { id } from "./locales/id";
import { en } from "./locales/en";

export const SUPPORTED_LANGUAGES = [
  { code: "id", label: "Indonesia" },
  { code: "en", label: "English" },
] as const;

// Resources are bundled (two small files), so init is synchronous and no
// component needs Suspense. Indonesian is the fallback because it's what the
// app shipped in before i18n — a browser in an unsupported language keeps
// seeing exactly what it always did. A choice made via LanguageSwitcher is
// cached in localStorage and wins over the browser language on later visits.
i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: { id: { translation: id }, en: { translation: en } },
    supportedLngs: SUPPORTED_LANGUAGES.map((l) => l.code),
    nonExplicitSupportedLngs: true, // en-US / id-ID -> en / id
    fallbackLng: "id",
    interpolation: { escapeValue: false }, // React already escapes
    detection: {
      order: ["localStorage", "navigator"],
      lookupLocalStorage: "lrc-translate:lang",
      caches: ["localStorage"],
    },
  });

// Keep <html lang> in step (screen readers, browser hyphenation/translation
// prompts). index.html hard-codes lang="id" as the pre-JS default.
function syncHtmlLang(lng: string) {
  document.documentElement.lang = lng.split("-")[0];
}
syncHtmlLang(i18n.resolvedLanguage ?? i18n.language);
i18n.on("languageChanged", syncHtmlLang);

export default i18n;
