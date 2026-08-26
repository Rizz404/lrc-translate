# Frontend Fixes: Translate Provider Priority — Comment-out Review Tooling (2026-08-26)

> Catatan perubahan UI hasil keputusan "KISS" user: chip `ScrapeReference` dan tombol "Bandingkan dengan AI" yang dibangun sesi kemarin ([docs/frontend/fixes-2026-08-25-scrape-alignment.md](fixes-2026-08-25-scrape-alignment.md)) dinilai kelewat rumit untuk masalah yang sebenarnya lebih baik dihindari daripada dikompensasi dengan tooling review. Lihat [docs/backend/fixes-2026-08-26-translate-priority.md](../backend/fixes-2026-08-26-translate-priority.md) untuk sisi backend-nya (prioritas provider Gemini→LibreTranslate, endpoint ai-reference dinonaktifkan).

## Latar belakang

Backend sekarang memprioritaskan Gemini (LLM, natural) di atas LibreTranslate (MT literal), dengan scrape+align jadi opsi terakhir yang manual. Karena scrape sekarang jauh lebih jarang jadi sumber utama translation, tooling review yang khusus dibangun buat mengompensasi ketidakakuratan alignment scrape (chip prev/matched/next + referensi AI on-demand) jadi kurang perlu dipertahankan aktif — user minta **dikomentari, bukan dihapus**, supaya tetap ada sebagai referensi/bisa diaktifkan lagi kapan-kapan.

## 1. Chip `ScrapeReference` (prev/matched/next + AI) dinonaktifkan

**Sebelumnya**: [LineRow.tsx](../../frontend/src/components/LineRow.tsx) merender `<ScrapeReference>` di bawah textarea lirik (mode Romaji) dan di bawah textarea terjemahan — tiap chip klik-able buat langsung apply teks scrape mentah lewat handler `pickLyric`/`pickTranslation`.

**Fix**: import `ScrapeReference`, kedua pemakaiannya (blok `usingRomanized && line.scrape_context?.romanized`, dan blok `line.scrape_context?.translation || line.scrape_context?.ai`), serta `pickLyric`/`pickTranslation` (satu-satunya pemakai handler-handler itu) — semua dikomentari, bukan dihapus. Setiap blok diberi komentar `KISS 2026-08-26: disabled...` yang menunjuk balik ke komentar di import untuk konteks kenapa dan cara uncomment lagi. Komponen [ScrapeReference.tsx](../../frontend/src/components/ScrapeReference.tsx) sendiri **tidak disentuh** — filenya tetap ada, cuma gak dipanggil dari mana pun sekarang.

## 2. Tombol "Bandingkan dengan AI" (`ai-reference`) dinonaktifkan

**Sebelumnya**: [EditorPage.tsx](../../frontend/src/pages/EditorPage.tsx) punya `aiReferenceMutation` (manggil `api.aiReference`) dan tombol trigger-nya di banner `needsReviewCount > 0`, lengkap dengan status sukses/gagal dan tombol retry-baris-gagal.

**Fix**: seluruh `aiReferenceMutation` (termasuk komentar penjelas panjang di atasnya yang tetap dipertahankan sebagai konteks historis) dan blok JSX tombol+status+retry-nya dikomentari. Import `Sparkles` dari `lucide-react` ikut dihapus dari daftar import (satu-satunya pemakainya adalah tombol yang sekarang nonaktif) dengan komentar penjelas kenapa. Endpoint backend-nya (`POST /tracks/:id/ai-reference`) juga dinonaktifkan sisi router (lihat doc backend) — jadi tombol ini gak cuma disembunyikan dari UI, API-nya sendiri sudah gak reachable.

## 3. Disclaimer diperkuat (bukan tooling baru — cuma copy)

Sesuai instruksi user "kalau scrape kasih desclaimer aja" — bukan bikin heuristik/tooling baru, cuma pertegas kalimat yang sudah ada:

- [ScrapePanel.tsx](../../frontend/src/components/ScrapePanel.tsx): teks sebelum tombol "Cari otomatis (utatime.com)" diganti dari sekadar *"Hasil alignment selalu ditandai 'perlu dicek' — posisinya cuma perkiraan, bukan hasil verifikasi"* jadi eksplisit bilang **scrape itu opsi terakhir** (pakai Translate/Gemini/LibreTranslate dulu), dan alignment *pernah* salah pasang baris tanpa kelihatan jelas per-baris — dengan pointer ke `docs/backend/fixes-2026-08-25-scrape-alignment.md` buat riwayat insiden nyatanya.
- [EditorPage.tsx](../../frontend/src/pages/EditorPage.tsx): banner ringkasan `needsReviewCount > 0` copy-nya diperkuat senada (sebelumnya cuma *"...perlu dicek manual — ditandai border kuning di bawah"*), dan referensi ke "cek chip biru AI" dihapus dari copy karena fitur itu sudah dikomentari (poin 2) — jangan menunjuk ke sesuatu yang sudah gak ada di UI.

## Yang sengaja tidak diubah

- `MethodBadge.tsx` — badge `scrape` memang sudah sengaja disembunyikan sejak sesi sebelumnya (redundant dengan border amber), gak perlu diubah lagi.
- `ScrapeReference.tsx` — file komponen tetap utuh, cuma gak dipanggil.
- `ScrapePanel.tsx`'s alur scrape+align itu sendiri (search/fetch/align) — gak disentuh, cuma copy disclaimer-nya.

## File yang berubah

**Modifikasi**:
- `frontend/src/components/LineRow.tsx`
- `frontend/src/pages/EditorPage.tsx`
- `frontend/src/components/ScrapePanel.tsx`

**Tidak ada file baru maupun file yang dihapus** — sesuai keputusan "comment saja, bukan dihapus".

Diverifikasi dengan `npx tsc -b`, `npx oxlint`, dan `npm run build` (semua sukses, dua warning yang muncul di `SearchPage.tsx`/`TranslatePanel.tsx` sudah ada sebelum perubahan ini, tidak terkait).
