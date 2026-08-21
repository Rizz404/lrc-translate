# Plan Extended: Rencana Implementasi Multi-Line Synced Lyrics Translator

> Turunan teknis dari [plan.md](plan.md). Dokumen ini berisi keputusan stack dan langkah build konkret, sedangkan `plan.md` tetap jadi sumber kebenaran untuk alur/fitur produk.

## Context

Repo ini saat ini kosong — hanya berisi `plan.md`, dokumen desain alur kerja untuk tool personal: cari lagu → tarik LRC dari LRCLIB → edit manual → (opsional) terjemahkan lewat MT/AI/scraping → simpan hasil per baris dengan tag `method`. Dokumen ini menerjemahkan `plan.md` menjadi rencana implementasi konkret dari nol, mengikuti urutan prioritas build yang sudah ditetapkan di `plan.md` (fondasi dasar dulu, baru MT, baru scraping, AI ditunda).

Keputusan stack yang sudah dikonfirmasi user:
- **Platform**: Web app, frontend + backend terpisah
- **Backend**: **Go** — framework diserahkan ke Claude
- **MT (Cabang A)**: LibreTranslate
- **Storage**: SQLite, tapi **portable** — hindari fitur khas SQLite supaya gampang pindah ke PostgreSQL/MySQL nanti

Dua keputusan teknis di bawah sudah divalidasi lewat web search (bukan asumsi), karena mesin dev ini Windows tanpa C compiler terpasang dan backend pindah ke Go butuh pengganti kuroshiro/kuromoji (library JS):
- **`glebarez/sqlite`** — driver SQLite pure-Go untuk GORM (tanpa cgo), direkomendasikan resmi oleh tim GORM. Tidak butuh gcc/C compiler untuk build di mesin ini.
- **`ikawaha/kagome` + `gojp/kana`** — pengganti kuromoji+kuroshiro: kagome men-tokenize teks Jepang & punya field `Reading` (katakana) per token (dictionary ter-embed di binary), lalu `gojp/kana` mengonversi katakana → romaji.

## Stack & Alasan

| Layer | Pilihan | Alasan |
|---|---|---|
| Frontend | React + Vite + TypeScript | Ekosistem matang untuk UI editor per-baris; TanStack Query (fetch/cache) + Zustand tipis (state editor lokal) |
| Backend | **Go + Gin** | Validasi/binding JSON bawaan (`go-playground/validator` via tag), ekosistem besar, cocok untuk API CRUD-heavy. `net/http`+`chi` jadi alternatif setara kalau nanti mau lebih minim-magic. |
| DB | **SQLite via GORM + `glebarez/sqlite`** (pure-Go, no cgo) | GORM mengabstraksi perbedaan dialek SQL — ganti ke Postgres/MySQL nanti = ganti 1 baris dialector (`gorm.io/driver/postgres`/`mysql`), model & query code tidak berubah |
| Romanisasi | **kagome + gojp/kana**, server-side | Dictionary ter-embed di binary Go (tidak perlu load file ~15MB terpisah seperti kuromoji di Node); dipanggil sekali per aksi romanize, hasil disimpan ke DB |
| MT | HTTP client kustom Go ke LibreTranslate (retry/backoff) | Publik atau self-host, env var yang sama |
| Scraping | **goquery** (`PuerkitoBio/goquery`, setara cheerio) + `net/http` | Parsing HTML ringan, tanpa headless browser dulu |
| Validasi request | Struct tag `binding`/`validate` Gin, dishare ke frontend lewat tipe TS yang di-maintain manual (lihat catatan di bawah) | |

**Catatan penting soal shared types**: karena backend sekarang Go (bukan Node), tidak bisa lagi 1 folder `shared/` TypeScript dipakai bareng FE+BE. Frontend dan backend jadi dua project terpisah sepenuhnya (npm project vs Go module). Tipe response API di-maintain manual sebagai TS interface di `frontend/src/api/types.ts`, mencerminkan JSON tag struct Go — cukup untuk project personal skala ini; kalau nanti dirasa rawan drift, bisa upgrade ke OpenAPI spec + codegen.

## Struktur Project

```
lrc-translate/
  plan.md                       # desain alur/fitur (sumber kebenaran produk)
  plan-extended.md              # dokumen ini
  frontend/                    # npm/Vite project, terpisah dari backend
    src/
      api/
        client.ts               # fetch wrapper + react-query hooks
        types.ts                 # TS interface manual, cermin JSON response Go
      pages/{SearchPage,EditorPage}.tsx
      components/{SongCandidateList,LineRow,TranslatePanel,RomanizeButton,CopyDownloadBar,MethodBadge,ScrapePanel}.tsx
      utils/lrcFormat.ts        # generate teks LRC di client (copy/download instan, tanpa round-trip)
      store/editorStore.ts       # zustand
  backend/                      # Go module terpisah
    go.mod / go.sum
    cmd/server/main.go           # entrypoint, wiring config+db+router
    internal/
      config/config.go           # env: DB_DRIVER, DB_DSN, LIBRETRANSLATE_URL, PORT, dst
      db/
        db.go                     # gorm.Open, dialector dipilih via DB_DRIVER (sqlite default)
        models.go                 # struct Track, Line, TranslationCache, ScrapeSource (GORM tags)
      lrclib/client.go            # client LRCLIB API
      lrc/lrc.go                  # parse/format teks LRC <-> []Line (fungsi murni + _test.go)
      libretranslate/client.go    # client MT + retry/backoff
      romanize/romanize.go        # pipeline kagome (tokenize+reading) -> gojp/kana (romaji)
      align/align.go              # heuristik alignment cabang C (fungsi murni + _test.go)
      scrape/fandom.go            # scraper goquery, mulai 1 situs
      httpapi/
        router.go                  # gin.Engine + middleware (cors, logger, recovery)
        dto.go                      # request/response structs, validator tags
        search_handler.go
        tracks_handler.go
        lines_handler.go
        translate_handler.go
        romanize_handler.go
        scrape_handler.go
        health_handler.go
    .env.example
  docker-compose.yml             # opsional: dokumentasi self-host LibreTranslate
```

## Skema Data (GORM models, portable)

Prinsip portabilitas — dijaga konsisten di semua model:
- PK `Track.ID` = **string UUID** yang di-generate di app layer (`google/uuid`), bukan mengandalkan ROWID/AUTOINCREMENT SQLite
- PK `Line.ID`/`TranslationCache.ID` boleh `uint` auto-increment biasa — GORM otomatis memetakan ke `INTEGER PRIMARY KEY AUTOINCREMENT` (SQLite) / `SERIAL` (Postgres) / `AUTO_INCREMENT` (MySQL) tanpa kode berbeda
- Tipe Go standar saja (`string`, `int64`, `bool`, `time.Time`) — GORM yang memetakan ke tipe kolom native per dialek, hindari `gorm:"type:..."` custom kecuali benar-benar perlu
- Schema dikelola via **`AutoMigrate`** (bukan raw DDL per-engine) — jalan konsisten di ketiga dialek
- **Dilarang** dipakai di kode: `PRAGMA`, fungsi JSON1 SQLite (`json_extract` dst), `WITHOUT ROWID`, FTS5, `ATTACH DATABASE`
- Query pakai GORM chainable builder (`Where`/`Joins`/`Order`), bukan raw SQL string — kalau butuh `Raw()`/`Exec()` sesekali, tetap tulis SQL standar ANSI

Model (ringkas):
- **Track**: ID (uuid), LrclibID, Title, Artist, Album, DurationMs, Language, Source (`lrclib`|`manual`), RawSyncedLrc, CreatedAt, UpdatedAt
- **Line**: ID, TrackID (FK), LineIndex, TimeMs, Timestamp, Original, Romanized, Translation, Method (`none`|`mt`|`ai`|`scrape`|`manual` — `ai` disiapkan dari awal walau belum dipakai), SuggestedTranslation, SuggestedMethod (snapshot sebelum edit manual → basis fitur "revert ke saran awal"), NeedsReview (bool), CreatedAt, UpdatedAt
- **TranslationCache**: CacheKey (unique, hash text+lang+provider), SourceText, TranslatedText, Provider, CreatedAt — cache lintas-lagu
- **ScrapeSource**: TrackID (FK), SourceURL, RawText, FetchedAt

Cache per-lagu (plan.md §8) otomatis terpenuhi karena `Line.Translation` persisten — buka ulang track tidak memicu API call ulang.

## API Endpoints (Gin routes)

```
GET  /api/search?title=&artist=                     -> proxy LRCLIB search
POST /api/tracks/import { lrclib_id }                -> fetch+parse LRC, buat track+lines
GET  /api/tracks/:id | GET /api/tracks | PUT | DELETE
PUT  /api/tracks/:id/lines/:lineId                    -> edit manual -> snapshot Suggested*, method="manual"
POST /api/tracks/:id/lines/:lineId/revert             -> kembalikan dari Suggested*
POST /api/tracks/:id/romanize                         -> kagome+gojp/kana untuk baris Jepang
POST /api/tracks/:id/translate { targetLang, lineIds? }  -> Cabang A, cek cache dulu, method="mt"
POST /api/tracks/:id/lines/:lineId/regenerate { method } -> regenerate satu baris saja
POST /api/tracks/:id/scrape { sourceUrl? }            -> Cabang C tahap 1: simpan raw_text
POST /api/tracks/:id/align { scrapeSourceId }         -> Cabang C tahap 2: align.go, method="scrape", needs_review=true
GET  /api/health
```

Export (copy/download LRC) **tidak butuh endpoint backend** — dibangkitkan client-side dari state yang sudah ada (`frontend/src/utils/lrcFormat.ts`), supaya alur "skip terjemahan → copy/download" instan tanpa round-trip, sesuai semangat "fondasi paling minimal".

## Milestone (ikuti urutan prioritas plan.md)

**M0 — Scaffolding**: `go mod init` di `backend/`, setup Gin + struktur folder `internal/`; `npm create vite` di `frontend/` (React+TS); `.env.example`, `.gitignore`, `git init`.

**M1 — Fondasi dasar** (cari lagu → LRC → copy/download, TANPA terjemahan): `db/models.go` (Track+Line), `db/db.go` (AutoMigrate), `lrclib/client.go`, `lrc/lrc.go` (+ `lrc_test.go`), `httpapi/search_handler.go` + `tracks_handler.go` + `router.go`, `pages/SearchPage.tsx` + `EditorPage.tsx`, `components/SongCandidateList.tsx` + `LineRow.tsx` (editable) + `CopyDownloadBar.tsx`, `utils/lrcFormat.ts`.
Verifikasi: `go run ./cmd/server`, test endpoint via curl/`Invoke-RestMethod` (`/api/search`, `/api/tracks/import`) sebelum FE dipakai; lalu di browser: cari lagu asli → pilih versi → edit satu baris → download `.lrc` → cek format `[mm:ss.xx]`.

**M2 — Cabang A (MT) + edit manual + tag method** (dibangun bersamaan — strukturnya dipakai semua cabang): tambah field `Method`/`Suggested*`/`NeedsReview`/`Language` ke model `Line` + model `TranslationCache`, re-`AutoMigrate`; `libretranslate/client.go` (retry/backoff on 429/5xx); `romanize/romanize.go` (kagome tokenizer + gojp/kana); `httpapi/translate_handler.go`, `lines_handler.go` (edit+revert), `romanize_handler.go`; deteksi bahasa Jepang via regex range Hiragana/Katakana/Kanji pada `Original`; `components/TranslatePanel.tsx`, side-by-side view di `LineRow.tsx` + `MethodBadge.tsx` + tombol revert, autosave debounced (~800ms) via PUT.
Verifikasi: panggil `/api/tracks/:id/translate` 2x pada track sama, cek panggilan kedua kena cache (lebih cepat, tidak keluar ke LibreTranslate); `go test ./internal/lrc/... ./internal/align/...`; di browser: edit baris → badge jadi "manual" → revert → refresh halaman → data persist (buka `backend/data/db.sqlite` pakai DB Browser for SQLite untuk cek isi tabel).

**M3 — Cabang C (scraping + alignment)**, setelah M2 stabil — **DESAIN BERUBAH dari rencana awal saat implementasi**: rencana awal menyontohkan auto-scrape dari database lirik tetap (Fandom Wiki dll), tapi kandidat-kandidat itu ternyata tidak bisa dipakai — lyricstranslate.com eksplisit melarang bot AI di robots.txt-nya (termasuk Claude), Fandom diproteksi Cloudflare berat, Genius diblokir oleh tool fetch. Desain final: **user paste URL sendiri** (bukan auto-crawl), backend cek robots.txt situs target sebelum scrape apa pun, dan ekstraksi teksnya generik (bukan parser 1 situs spesifik) supaya bekerja untuk URL manapun.

File utama:
- `scrape/scrape.go`: `Scrape(ctx, rawURL) (string, error)` — cek robots.txt via `temoto/robotstxt` (User-Agent jujur `lrc-translate-bot/0.1`, fail-open kalau robots.txt tidak ada/gagal diakses, block kalau eksplisit disallow), lalu fetch + ekstrak teks generik via goquery (strip `script/style/nav/header/footer/aside/form`, ambil teks dari elemen blok, dedupe baris berurutan yang sama)
- `align/align.go` sebagai fungsi murni `Align(original, scrapedRaw []string) []string`, dengan strategi:
  1. Bersihkan teks scraped (buang baris kosong/marker `[Chorus]` dll via regex, blok dipisah baris kosong — original juga otomatis punya "blok" dari baris kosong LRC instrumental gap)
  2. Kalau jumlah baris cleaned sama dengan original → mapping posisional 1:1
  3. Kalau jumlah blok sama tapi isi beda → alignment per-blok, proporsional di dalam blok kalau perlu
  4. Kalau tidak cocok sama sekali → fallback pemetaan indeks proporsional global
  5. Semua hasil Cabang C **selalu** ditandai `NeedsReview=true` oleh handler, apa pun strateginya
- `httpapi/scrape_handler.go`: `POST /tracks/:id/scrape {source_url}` (stage 1, simpan raw text ke `ScrapeSource`) + `POST /tracks/:id/align {scrape_source_id}` (stage 2, jalankan `align.Align`, tulis ke `Line.Translation`/`Method=scrape`/`NeedsReview=true`)
- `components/ScrapePanel.tsx`: input URL collapsible, tombol "Scrape halaman" → "Terapkan Alignment", preview teks mentah, baris hasil align otomatis kena highlight amber dari `LineRow.tsx` yang sudah ada (needs_review sejak M2)

Verifikasi: `go test ./internal/align/...` (kasus exact match, block match, proportional fallback, instrumental gap, nil input) dan `go test ./internal/scrape/...` (extractText via HTML statis, checkRobots via `httptest` mock server untuk allow/disallow) — semua tanpa jaringan asli; lalu manual end-to-end di browser: scrape URL nyata (diuji dengan artikel Wikipedia) → cek raw text preview → Terapkan Alignment → cek badge "scrape · perlu dicek" dan border amber muncul di baris yang ter-mapping.

**M4 — Cabang B (AI)**: DITUNDA, tidak diimplementasikan. Yang sudah siap dari M2: enum `Method` di DB sudah menerima `"ai"`. Frontend cukup tampilkan tab "AI — Coming Soon" dalam keadaan disabled, tanpa handler baru.

## Handling LibreTranslate rate-limit/downtime
Env var `LIBRETRANSLATE_URL`/`LIBRETRANSLATE_API_KEY` (publik atau self-host, kode sama); timeout ±10s + retry/backoff 2-3x untuk 429/5xx (`net/http` client + `time.Sleep` backoff atau lib `github.com/avast/retry-go`); throttle concurrency (`golang.org/x/sync/errgroup` + semaphore, mis. maks 2 concurrent) saat batch-translate satu track; cache agresif lewat `TranslationCache`+`Line.Translation`; fallback self-host didokumentasikan via `docker-compose.yml`/README (`docker run -p 5000:5000 libretranslate/libretranslate`) — **tidak divalidasi di mesin ini** karena Docker belum terpasang.

## Testing/Verifikasi Umum
- Backend: `go test ./...` untuk fungsi murni rawan bug (`internal/lrc`, `internal/align`) — table-driven test khas Go, tanpa mock DB/jaringan; endpoint baru dicek manual via curl/`Invoke-RestMethod` sebelum dikonsumsi FE
- Frontend: manual browser testing end-to-end tiap milestone dengan lagu nyata
- DB: cek `backend/data/db.sqlite` via `sqlite3 db.sqlite ".schema"`/`.tables` atau DB Browser for SQLite setelah tiap milestone, pastikan field `Method`/`Suggested*`/`NeedsReview` sesuai desain
