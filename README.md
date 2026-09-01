# lrc-translate

Multi-line synced lyrics translator. Lihat [plan.md](plan.md) untuk alur/fitur produk dan [plan-extended.md](plan-extended.md) untuk rencana implementasi teknis (stack, skema DB, milestone).

## Menjalankan secara lokal

### Backend (Go)

```
cd backend
cp .env.example .env   # sesuaikan kalau perlu
go run ./cmd/server
```

Berjalan di `http://localhost:8080`. Database SQLite dibuat otomatis di `backend/data/db.sqlite` (di-gitignore).

Test: `go test ./...`

### Frontend (React + Vite)

```
cd frontend
cp .env.example .env   # sesuaikan kalau perlu
npm install
npm run dev
```

Berjalan di `http://localhost:5173`.

## Status implementasi

- [x] Milestone 0 — Scaffolding (Go module + Vite React TS project)
- [x] Milestone 1 — Cari lagu → import LRC dari LRCLIB → edit manual per baris → copy/download `.lrc`
- [x] Milestone 2 — Terjemahan MT (LibreTranslate) + romanisasi (kagome/gojp-kana) + tag `method` + revert
- [x] Milestone 3 — Scraping + alignment heuristik (Cabang C)
- [ ] Milestone 4 — AI/LLM (Cabang B) — ditunda, skema data sudah siap (`method: "ai"`)

### Catatan desain Milestone 3 (scraping)

Rencana awal (`plan.md`) menyontohkan auto-scrape dari database lirik seperti Fandom Wiki. Saat implementasi, kandidat-kandidat itu ternyata tidak bisa dipakai:
- **lyricstranslate.com** — robots.txt-nya eksplisit melarang bot AI (`anthropic-ai`, `Claude-Web`, `ClaudeBot`, dll)
- **Fandom** — diproteksi Cloudflare, bahkan `robots.txt`-nya sendiri gagal diakses
- **Genius** — diblokir oleh tool fetch yang dipakai

Karena itu backend melakukan **cek robots.txt otomatis** untuk URL manapun sebelum scrape (`backend/internal/scrape/scrape.go`) — kalau situsnya melarang, request ditolak dengan pesan jelas. Ekstraksi tekstnya generik (readability-style: ambil teks dari elemen blok, buang nav/header/footer/script/style) untuk URL sembarang.

**utatime.com** kemudian ditambahkan sebagai sumber khusus (`backend/internal/scrape/utatime.go`) karena strukturnya sangat cocok: tiap halaman lirik punya blok `#Original` dan `#Translations` dengan baris `<span class="line-text">` yang sudah sejajar 1:1 per bahasa — parser presisi dipakai otomatis begitu URL-nya dikenali dari domain `utatime.com`. Tapi search API resmi mereka (`/music-api/site/v1/search`) dan sitemap-nya diproteksi Cloudflare bot-challenge, dan robots.txt-nya sendiri (walau permisif untuk `*`) ternyata dibarengi rate-limit agresif — beberapa kali fetch berturut-turut saat riset langsung kena 403 sementara. Karena itu discovery-nya **tebak URL dari judul+artist dulu (best-effort, sering meleset khususnya untuk nama artis Jepang yang di-romanisasi beda dari LRCLIB), fallback ke paste link manual kalau tebakannya gagal** — lihat tombol "Cari otomatis (utatime.com)" di `ScrapePanel.tsx`.

Heuristik alignment (`internal/align/align.go`) punya 3 tingkat: posisional 1:1 (kalau jumlah baris persis sama), block-based (kalau jumlah blok yang dipisah baris kosong sama), fallback proporsional (stretch/kompresi index, dengan rounding — bukan floor — biar duplikasi tidak menumpuk di baris-baris awal). Walau begitu, kalau sumber terjemahan menggabung beberapa baris asli jadi satu kalimat (umum terjadi), hasilnya akan drift dari titik itu — makanya **hasil Cabang C selalu ditandai "perlu dicek"**, tidak pernah dianggap final tanpa review manual.

Selain terjemahan, halaman utatime.com juga punya romanisasi resmi (tab `#Romaji`) yang kualitasnya lebih baik dari romanizer kita sendiri (kagome+gojp/kana) — jadi begitu align dijalankan, `Line.Romanized` ikut di-align dari sumber itu dan **menggantikan** hasil romanizer otomatis pada baris yang berhasil di-align (independen dari alignment terjemahan, karena pembagian barisnya bisa beda).

## Deploy dengan Docker (Plesk)

Repo ini punya satu `Dockerfile` (root) yang build frontend (Vite) dan backend (Go) lalu menghasilkan **satu image**: binary Go yang sekaligus menyajikan hasil build frontend (static files + SPA fallback ke `index.html`, lihat `STATIC_DIR` di `backend/internal/httpapi/router.go`). `docker-compose.yml` (root) mendefinisikan stack produksinya: service `app` (image di atas) + service `libretranslate` (self-hosted, supaya translate tidak bergantung ke API key berpusat libretranslate.com — lihat bagian di bawah).

### Via Plesk Docker extension

1. Push/clone repo ini ke server (atau upload lewat File Manager Plesk).
2. Buka **Docker → Stacks** di Plesk, buat stack baru, arahkan ke `docker-compose.yml` di direktori project (atau tempel isinya).
3. Sebelum deploy, siapkan `.env` di direktori yang sama dengan `docker-compose.yml`:
   ```
   cp .env.example .env   # sesuaikan APP_PORT dll bila perlu
   ```
4. Deploy stack. First run akan build image (bisa beberapa menit) dan libretranslate akan mengunduh model bahasa (bisa beberapa GB, cukup sekali karena disimpan di named volume `libretranslate-data`).
5. Container `app` publish ke `127.0.0.1:${APP_PORT:-8080}` (host). Di **Websites & Domains**, pilih domain yang mau dipakai → set domain itu ke mode proxy/reverse-proxy ke `127.0.0.1:8080` (atau tambahkan Nginx directive tambahan seperti `proxy_pass http://127.0.0.1:8080;`) supaya lalu lintas HTTPS Plesk diteruskan ke container.
6. Data (SQLite + database lirik) persisten di named volume `app-data` (`/app/data/db.sqlite` di dalam container) — aman terhadap rebuild/redeploy image, cuma hilang kalau volume-nya sendiri dihapus.

### Via CLI (SSH ke server)

```
cp .env.example .env   # sesuaikan bila perlu
docker compose up -d --build
```

Update ke versi baru: `git pull && docker compose up -d --build`.

## ⚠️ LibreTranslate: instance publik sekarang butuh API key

Per Agustus 2026, `https://libretranslate.com` **menolak request tanpa API key berbayar** (`Visit https://portal.libretranslate.com to get an API key`). Beberapa mirror publik lama (`translate.argosopentech.com`, `translate.terraprint.co`, dll) sudah mati/tidak stabil. Client sudah menangani ini dengan benar (error 400 tidak di-retry, pesan error jelas ditampilkan ke user) — ini keterbatasan layanan eksternal, bukan bug di kode.

Opsi:
1. **Self-host via Docker untuk dev lokal** (butuh Docker terpasang — belum ada di mesin dev ini). `docker-compose.yml` di root berisi stack produksi lengkap (app + libretranslate — lihat bagian "Deploy dengan Docker" di atas), dengan port libretranslate **tidak** dipublish ke host secara default (biar tidak ke-expose publik saat production). Untuk dev lokal, jalankan service libretranslate-nya saja dan publish portnya, backend/frontend tetap jalan native seperti biasa (`go run`, `npm run dev`):
   ```
   docker compose run -d --name libretranslate-dev -p 5000:5000 libretranslate
   ```
   (atau uncomment blok `ports:` di `docker-compose.yml` lalu `docker compose up -d libretranslate`). Set `LIBRETRANSLATE_URL=http://localhost:5000` di `backend/.env`. First-run akan mengunduh model bahasa (butuh waktu & disk cukup besar).
2. **Beli API key** dari [portal.libretranslate.com](https://portal.libretranslate.com) dan set `LIBRETRANSLATE_API_KEY` di `backend/.env`.
3. Cari mirror publik gratis yang masih hidup (cek [community.libretranslate.com](https://community.libretranslate.com)).

## Prioritas provider translate: LLM lokal → Gemini → LibreTranslate → scrape (KISS, 2026-08-31)

LibreTranslate itu NMT klasik — gratis dan self-hostable, tapi hasilnya cenderung harfiah kata-per-kata, kurang cocok buat lirik lagu yang butuh nuansa/gaya bahasa natural. Backend punya dua provider LLM alternatif, keduanya pakai prompt yang sama (`backend/internal/llmprompt`) yang diarahkan buat nerjemahin "kayak lirik lagu", bukan literal:
- `localllm` (`backend/internal/localllm`) — server OpenAI-compatible yang kamu host sendiri (LM Studio, Ollama, dll), biasanya diakses lewat tunnel (ngrok) karena gak publicly routable sendiri. **Prioritas pertama** kalau dikonfigurasi: gratis/unlimited karena jalan di hardware sendiri, gak kena rate limit/kuota harian kayak layanan cloud.
- `gemini` (`backend/internal/gemini`) — LLM Gemini via [Google AI Studio](https://aistudio.google.com/apikey) (ada free tier). Prioritas kedua, dipakai kalau `localllm` gak dikonfigurasi.

Cara pakai `localllm`: set `LOCAL_LLM_URL=<url server kamu>` (misal URL ngrok, tanpa `/v1` di akhir), `LOCAL_LLM_API_KEY` kalau server-nya butuh Bearer token (mis. LM Studio dengan "require API key" aktif), dan `LOCAL_LLM_MODEL` sesuai id model persis seperti yang muncul di `GET /v1/models` server kamu. Cara pakai `gemini`: set `GEMINI_API_KEY=<key kamu>` (di `backend/.env` untuk lokal, atau di env vars stack Portainer untuk production — **jangan** commit API key ke git). `GEMINI_MODEL` opsional, default `gemini-2.5-flash`.

**Biarkan `TRANSLATE_PROVIDER` kosong/unset** — backend otomatis pakai `localllm` kalau `LOCAL_LLM_URL` terisi, jatuh ke `gemini` kalau tidak tapi `GEMINI_API_KEY` terisi, dan jatuh ke LibreTranslate kalau keduanya kosong (lihat `backend/cmd/server/main.go`). Ini cuma keputusan startup, bukan retry otomatis saat runtime — kalau provider yang aktif bermasalah di tengah sesi (server lokal/tunnel down, atau Gemini kehabisan kuota harian — lihat `docs/backend/fixes-2026-08-25-scrape-alignment.md` poin 8), pemulihan tercepat adalah set `TRANSLATE_PROVIDER=<provider lain>` eksplisit lalu restart server, bukan menunggu auto-retry.

Scrape+align (import terjemahan dari situs lirik eksternal, lihat `ScrapePanel` di editor) tetap ada sebagai **opsi terakhir, manual** — dipakai kalau Gemini/LibreTranslate belum cukup (mis. butuh terjemahan siap-pakai yang sudah published). Hasilnya selalu ditandai "perlu dicek": alignment itu heuristik posisi, pernah menghasilkan baris salah pasang tanpa kelihatan jelas per-baris (lihat `docs/backend/fixes-2026-08-25-scrape-alignment.md` poin 3 untuk detail insidennya).

Cache translation (`translation_cache` table) di-namespace per provider yang benar-benar dipakai (bukan raw `TRANSLATE_PROVIDER`, yang bisa kosong saat auto-resolve), jadi gonta-ganti provider gak bakal ke-mix atau kepakein hasil cache dari provider lain — translate ulang otomatis dipicu untuk baris yang belum pernah diterjemahkan oleh provider yang lagi aktif.
