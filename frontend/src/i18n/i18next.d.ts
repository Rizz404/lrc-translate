import type { id } from "./locales/id";

// Makes t("some.key") compile-time checked against the Indonesian resource.
declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: "translation";
    resources: { translation: typeof id };
  }
}
