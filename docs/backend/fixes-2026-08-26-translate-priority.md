# Backend Fixes: Translate Provider Priority (Gemini → LibreTranslate) (2026-08-26)

> Catatan perubahan hasil keputusan "KISS" user setelah sesi kemarin ([docs/backend/fixes-2026-08-25-scrape-alignment.md](fixes-2026-08-25-scrape-alignment.md)) dinilai kelewat rumit untuk masalah yang sebenarnya bisa dihindari: alih-alih membangun lebih banyak tooling review buat mengatasi scrape+align yang gak reliable, urutan sumber translate diprioritaskan supaya scrape sejarang mungkin dipakai. Lihat juga [docs/frontend/fixes-2026-08-26-translate-priority.md](../frontend/fixes-2026-08-26-translate-priority.md) untuk sisi UI-nya (comment-out tooling review + disclaimer).

## Latar belakang

Keputusan user: urutan prioritas sumber translate adalah **(1) Gemini API (LLM, natural) → (2) LibreTranslate (MT, fallback) → (3) scrape+align (last resort, manual)**. Diklarifikasi lewat pertanyaan eksplisit sebelum implementasi:
- Fallback Gemini→LibreTranslate: **urutan default konfigurasi saja**, bukan auto-retry di request yang sama — `TRANSLATE_PROVIDER` tetap satu switch statis per proses, yang berubah cuma cara resolve default-nya.
- Tooling review scrape (chip `ScrapeReference`, tombol/endpoint `ai-reference`) yang dibangun sesi kemarin: **dikomentari, bukan dihapus** — supaya gampang diaktifkan lagi kalau perlu di masa depan.

## 1. `TRANSLATE_PROVIDER` unset sekarang berarti "auto-prefer Gemini kalau ada key", bukan hardcode LibreTranslate

**Sebelumnya**: [config.go](../../backend/internal/config/config.go) default `TRANSLATE_PROVIDER` ke `"libretranslate"` secara hardcoded, dan [main.go](../../backend/cmd/server/main.go) switch statement men-treat `""` sama persis dengan `"libretranslate"` — jadi gak ada cara bikin Gemini jadi prioritas default tanpa eksplisit set env var, dan kalau default langsung diganti ke `"gemini"` di config, deployment yang belum punya `GEMINI_API_KEY` bakal `log.Fatalf` saat startup (regresi berbahaya buat env yang sudah jalan).

**Fix**: `""` (unset) sekarang jadi kondisi "auto-resolve", terpisah dari case `"libretranslate"` eksplisit:
- [config.go](../../backend/internal/config/config.go): default `getEnv("TRANSLATE_PROVIDER", "libretranslate")` → `getEnv("TRANSLATE_PROVIDER", "")`.
- [main.go](../../backend/cmd/server/main.go) switch: case `""` sekarang cek `cfg.GeminiAPIKey` — kalau terisi, pakai `gemini.New(...)`; kalau kosong, `libretranslate.New(...)` (behavior lama, gak berubah). Case `"gemini"` dan `"libretranslate"` eksplisit tetap seperti sebelumnya, termasuk `log.Fatalf` kalau `gemini` diminta eksplisit tanpa key — itu tetap masuk akal (beda dari auto-resolve yang diam-diam fallback tanpa bikin developer sadar keynya belum diisi).
- Provider yang benar-benar dipilih di-log saat startup (`log.Printf("translate provider: %s...")`) supaya gampang diverifikasi provider mana yang aktif tanpa harus baca kode.

## 2. Bug laten yang ketahuan pas ngerjain ini: cache namespace bakal salah kalau `TRANSLATE_PROVIDER` dibiarkan auto-resolve

Cache translation (`translation_cache` table) di-namespace per provider lewat `Server.translatorID` (lihat catatan di [router.go](../../backend/internal/httpapi/router.go) dan poin 8 doc kemarin soal alasannya) — sebelumnya `main.go` langsung passing `cfg.TranslateProvider` mentah ke `httpapi.NewServer(...)`. Begitu `cfg.TranslateProvider` bisa `""` (auto-resolve, poin 1), dua server yang sama-sama auto-resolve tapi beda hasil (satu punya `GEMINI_API_KEY`, satu enggak) bakal sama-sama pakai namespace cache `""` — persis bug yang sudah pernah diperbaiki sebelumnya ("cached results reused across different providers"), muncul lagi lewat jalur baru.

**Fix**: [main.go](../../backend/cmd/server/main.go) sekarang melacak `resolvedProvider` (nilai aktual: `"gemini"` atau `"libretranslate"`, gak pernah `""`) terpisah dari `cfg.TranslateProvider` (bisa `""`), dan `resolvedProvider` itu yang di-passing ke `NewServer(...)` sebagai `translatorID` — bukan `cfg.TranslateProvider`.

## 3. Endpoint `POST /tracks/:id/ai-reference` dinonaktifkan (bukan dihapus)

Sesuai keputusan user untuk comment-out tooling review scrape (lihat Latar belakang), registrasi route-nya di [router.go](../../backend/internal/httpapi/router.go) dikomentari — endpoint sekarang balikin 404. Handler-nya sendiri ([ai_reference_handler.go](../../backend/internal/httpapi/ai_reference_handler.go)) **tidak disentuh sama sekali**, cuma jadi gak ke-route lagi, supaya tinggal uncomment satu baris kalau mau diaktifkan lagi nanti. Ini selaras dengan sisi frontend: tombol "Bandingkan dengan AI" yang manggil endpoint ini juga dikomentari (lihat doc frontend).

## 4. Dokumentasi konfigurasi diselaraskan

- [.env.example](../../.env.example): comment `TRANSLATE_PROVIDER` dijelaskan ulang — kosongkan untuk auto (Gemini kalau `GEMINI_API_KEY` diisi, else LibreTranslate), atau set eksplisit untuk memaksa satu provider. Default value di file diubah dari `TRANSLATE_PROVIDER=libretranslate` jadi `TRANSLATE_PROVIDER=` (kosong).
- [docker-compose.yml](../../docker-compose.yml): **bug yang ketauan sekalian** — `TRANSLATE_PROVIDER: ${TRANSLATE_PROVIDER:-libretranslate}` bakal selalu maksa string literal `"libretranslate"` ke proses Go kalau env var host-nya kosong/gak diset, membuat auto-resolve di poin 1 gak akan pernah ke-trigger lewat jalur Docker Compose/Portainer (yang notabene jalur deployment production sesuai README). Diperbaiki jadi `${TRANSLATE_PROVIDER:-}` supaya string kosong beneran nyampe ke `config.Load()`. Comment section juga diupdate.
- [README.md](../../README.md): section "Provider translate alternatif: Gemini" diganti jadi "Prioritas provider translate: Gemini → LibreTranslate → scrape (KISS, 2026-08-26)" — menjelaskan urutan prioritas baru, cara pakai (`TRANSLATE_PROVIDER` dibiarkan kosong), cara pulih kalau Gemini kehabisan kuota harian (set eksplisit ke libretranslate + restart, bukan auto-retry), dan posisi scrape sebagai opsi terakhir.

## Yang sengaja tidak diubah

- `translate_handler.go`, `Translator` interface — tetap satu provider aktif per proses, sesuai keputusan "urutan default saja" (bukan runtime fallback per-request).
- `align.go` (algoritma alignment scrape) — gak disentuh, sesuai keputusan sesi sebelumnya untuk gak lanjut ke MT-assisted alignment.
- `ai_reference_handler.go`, `scrape_handler.go`, `Line.ScrapeContext` (DB/DTO) — semua tetap utuh, cuma jalur pemanggilannya (route + frontend trigger) yang dimatikan.

## File yang berubah

**Modifikasi**:
- `backend/internal/config/config.go`
- `backend/cmd/server/main.go`
- `backend/internal/httpapi/router.go`
- `.env.example`
- `docker-compose.yml`
- `README.md`

Diverifikasi dengan `go build ./...`, `go vet ./...`, dan `go test ./...` (semua sukses, gak ada regresi). Manual: server tetap start normal tanpa `GEMINI_API_KEY` maupun `TRANSLATE_PROVIDER` (fallback ke LibreTranslate, behavior lama gak berubah).
