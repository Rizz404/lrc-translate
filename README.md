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
- [ ] Milestone 3 — Scraping + alignment heuristik (Cabang C)
- [ ] Milestone 4 — AI/LLM (Cabang B) — ditunda, skema data sudah siap (`method: "ai"`)

## ⚠️ LibreTranslate: instance publik sekarang butuh API key

Per Agustus 2026, `https://libretranslate.com` **menolak request tanpa API key berbayar** (`Visit https://portal.libretranslate.com to get an API key`). Beberapa mirror publik lama (`translate.argosopentech.com`, `translate.terraprint.co`, dll) sudah mati/tidak stabil. Client sudah menangani ini dengan benar (error 400 tidak di-retry, pesan error jelas ditampilkan ke user) — ini keterbatasan layanan eksternal, bukan bug di kode.

Opsi:
1. **Self-host via Docker** (butuh Docker terpasang — belum ada di mesin dev ini):
   ```
   docker compose up -d
   ```
   Lalu set `LIBRETRANSLATE_URL=http://localhost:5000` di `backend/.env`. First-run akan mengunduh model bahasa (butuh waktu & disk cukup besar).
2. **Beli API key** dari [portal.libretranslate.com](https://portal.libretranslate.com) dan set `LIBRETRANSLATE_API_KEY` di `backend/.env`.
3. Cari mirror publik gratis yang masih hidup (cek [community.libretranslate.com](https://community.libretranslate.com)).
