# Backend Fixes: Scrape Alignment & AI Reference (2026-08-25)

> Catatan perbaikan bug alignment scrape (Cabang C) yang dilaporkan user lewat contoh nyata lagu "Compared Child" (TUYU, scrape dari utatime.com), plus fitur baru yang lahir dari investigasi bug tersebut. Referensi historis — kalau ada regresi terkait area ini, mulai dari sini. Lihat juga [docs/frontend/fixes-2026-08-25-scrape-alignment.md](../frontend/fixes-2026-08-25-scrape-alignment.md) untuk sisi UI-nya.

## 1. Baris hasil scrape dobel/salah posisi (bug di `alignByBlock`)

**Gejala**: hasil scrape+align untuk lagu Kuraberarekko menampilkan baris yang identik dobel di 2 timestamp berbeda (`Nan to naku sa wo kanjite` muncul di `00:14.84` dan `00:19.32`), dan baris-baris setelahnya jadi salah posisi.

**Root cause**: [align.go](../../backend/internal/align/align.go)'s `alignByBlock` — waktu ukuran block scraped lebih kecil dari block original (mis. penerjemah menggabung 2 baris Jepang jadi 1 kalimat Inggris di tengah block, bukan cuma di ujung), fungsi lama meng-copy langsung `scrapedBlock[i]` untuk index awal, dan baru jatuh ke `proportionalIndex` untuk sisa index yang "kehabisan slot" di akhir. Ini bikin baris setelah titik penggabungan geser satu index terus-menerus, dan baris terakhir block ke-assign index yang sama dengan baris sebelum­nya (dobel).

**Fix**: `proportionalIndex` sekarang dipakai merata untuk **seluruh** posisi dalam block begitu ukurannya beda (bukan cuma untuk sisa index di ujung) — dibuktikan lewat hitung tangan bahwa ini menghasilkan pemetaan yang benar untuk kasus penggabungan di tengah block. Kalau ukuran block sama persis, tetap direct-copy 1:1 (gak ada pembulatan).

**Test baru**: `TestAlign_BlockHeuristicHandlesMergeInsideBlock` di [align_test.go](../../backend/internal/align/align_test.go) — mereproduksi kasus penggabungan di tengah block dan memastikan baris block berikutnya gak ikut kebocoran.

## 2. Strategi baru: positional match yang mempertahankan baris kosong

**Temuan awal (salah duga, lalu dikoreksi)**: sempat dicurigai bahwa `Align()` gagal mendeteksi kecocokan 1:1 karena strategi exact-match lama membandingkan `cleanedFlat` (baris kosong sudah dibuang) dengan `len(original)` (baris kosong masih ada) — hampir gak akan pernah sama persis kalau original punya instrumental gap.

**Fix**: strategi baru `blanksLineUp` + `trimAnnotationsKeepBlanks` — kalau scraped text (setelah annotation dibuang, tapi baris kosong dipertahankan) sama panjang dengan original DAN posisi baris kosongnya persis sejajar, langsung mapping 1:1 by index, prioritas tertinggi sebelum strategi block/proporsional.

**Catatan penting**: setelah diverifikasi ke data asli (lihat poin 3), strategi ini **tidak** yang memperbaiki kasus Kuraberarekko — sumber utatime.com untuk lagu itu ternyata TIDAK 1:1 sejajar (13 pemisah paragraf di scraped text vs cuma 4 gap instrumental beneran di LRC). Strategi ini tetap berguna untuk sumber yang genuinely sejajar per baris, tapi bukan solusi kasus ini.

## 3. Investigasi mendalam: skala masalah sebenarnya jauh lebih besar dari dugaan awal

Setelah audit manual seluruh 61 baris lagu Kuraberarekko (bandingkan original vs translation vs `scrape_context.prev`), ditemukan **~30 baris** (bukan cuma 6 yang keliatan dobel persis) sebenarnya salah posisi — mayoritas silent drift yang gak keliatan karena tiap baris individual tetap kalimat Inggris yang masuk akal, cuma dipasangkan ke baris Jepang yang salah.

**Root cause matematis**: `alignProportional` (fallback global, dipakai waktu jumlah block scraped ≠ jumlah block original) mengasumsikan kompresi (baris tergabung) tersebar RATA sepanjang lagu. Kenyataannya penggabungan menumpuk di section tertentu (mis. 2 penggabungan cuma dalam 8 baris pertama) — begitu rumus "kalah cepat" dari kompresi asli, index yang di-assign jadi overshoot dan gak "nyadar" sampai titik reset berikutnya (biasanya belasan baris kemudian).

**Keputusan**: TIDAK melakukan perbaikan algoritma lebih lanjut (opsi "MT-assisted alignment" pakai DP+similarity matching sempat dipertimbangkan, tapi ditunda) — pure line-counting heuristic gak bisa dibuat jauh lebih akurat tanpa sinyal isi teks. Sebagai gantinya, dibangun tooling review (lihat poin 4 & 5) yang membantu manusia mengoreksi, bukan mencoba menyelesaikan otomatis 100%.

## 4. Fitur baru: `AlignWithContext` — referensi baris tetangga dari scrape mentah

**Tujuan**: karena setiap hasil align heuristik tetap `needs_review=true` (gak pernah dianggap terverifikasi), reviewer butuh cara cepat lihat "apa sih tepatnya yang ada di raw scraped text di sekitar baris ini" tanpa harus scroll manual ke viewer teks mentah.

**Implementasi** ([align.go](../../backend/internal/align/align.go)):
- `Context{Prev, Matched, Next string}` — baris scraped tepat sebelum/pas/sesudah yang dipakai untuk satu posisi original.
- `AlignWithContext(original, scrapedRaw) ([]string, []Context)` — versi `Align()` yang juga mengembalikan context. `Align()` sekarang jadi wrapper tipis yang buang context-nya (backward compatible, semua test lama tetap pass tanpa perubahan).
- Semua strategi (positional/exact/block/proportional) dimodifikasi untuk juga melacak index ke `cleanedFlat`/`scrapedPositional` per posisi, dipakai `fillContext` buat ambil prev/next.

**Test baru**: `TestAlignWithContext_ProportionalGivesNeighborsForReview`, `TestAlignWithContext_BlankPositionsGetZeroValueContext`, `TestAlignWithContext_PositionalStrategyUsesAdjacentRawLines`.

**Verifikasi nyata**: dites lewat API asli (bukan cuma unit test) ke data lagu Kuraberarekko — chip "prev" terbukti berisi jawaban yang benar untuk beberapa baris yang salah (mis. `00:14.84` seharusnya "My left side hurts and it troubles me." — persis isi chip prev-nya).

## 5. Penyimpanan & eksposur context (`Line.ScrapeContext`)

- **DB**: kolom baru `Line.ScrapeContext string` (JSON text) di [models.go](../../backend/internal/db/models.go) — otomatis termigrasi lewat `AutoMigrate` yang sudah ada (gak perlu migration script terpisah).
- **DTO**: `LineScrapeContextDTO{Prev, Matched, Next}` dan `LineScrapeContextsDTO{Translation, Romanized, AI}` di [dto.go](../../backend/internal/httpapi/dto.go), diekspos di `LineDTO.ScrapeContext` (`json:"scrape_context,omitempty"`).
- **Penulisan**: `handleAlignTrack` di [scrape_handler.go](../../backend/internal/httpapi/scrape_handler.go) sekarang manggil `align.AlignWithContext` untuk translation **dan** romanized (independen, karena keduanya bisa punya breakdown baris berbeda), lalu marshal ke `line.ScrapeContext`. Replace wholesale tiap align call (bukan merge) — supaya gak pernah nampilin context dari scrape source yang beda dari yang lagi aktif.
- **Pembacaan**: `toLineDTO` di [tracks_handler.go](../../backend/internal/httpapi/tracks_handler.go) unmarshal JSON-nya; gagal parse dianggap "gak ada context" (bukan error fatal).

## 6. Fitur baru: endpoint `POST /tracks/:id/ai-reference`

**Latar belakang**: user melaporkan gak bisa baca huruf Jepang/romaji, jadi chip prev/next dari poin 4 gak banyak membantu kecuali kebetulan dobel persis — gak ada cara menilai "apakah translation ini cocok sama original" tanpa mengerti bahasa sumbernya.

**Ide**: tambah referensi ke-3 yang **selalu benar posisinya** (bukan tebakan alignment) — machine translation langsung per baris original, dalam bahasa target yang sama dengan translation yang lagi direview, sehingga reviewer bisa bandingkan Inggris-vs-Inggris tanpa perlu ngerti Jepang sama sekali.

**Implementasi** ([ai_reference_handler.go](../../backend/internal/httpapi/ai_reference_handler.go), baru):
- `POST /tracks/:id/ai-reference { target_lang, line_ids? }` — reuse `translateOneCached` (cache yang sama dipakai `handleTranslateTrack`, jadi baris yang berulang, mis. chorus, gratis di-request ulang) dan pola concurrency `maxConcurrentTranslations` yang sudah ada.
- **Tidak menyentuh** `Translation`/`Method`/`NeedsReview` sama sekali — murni additif, di-merge ke `ScrapeContext.AI` (unmarshal-modify-remarshal, supaya gak menghapus context Translation/Romanized yang udah ada dari align sebelumnya).
- Response `AIReferenceResponse{Lines, CacheHits, CacheMisses, Failed}` — tiap baris gagal dilaporkan per-line (bukan bikin seluruh request gagal), supaya batch besar tetap dapet hasil parsial.
- Route didaftarkan di [router.go](../../backend/internal/httpapi/router.go).

**Verifikasi nyata**: dites ke line yang beneran drift (`00:35.39`, original "A, B, C, D, E, F, G") — scrape translation salah total ("No matter what i chose"), AI reference benar ("A, B, C, D, E, F,") — kelihatan jelas mismatch-nya cuma dari teks Inggris, tanpa baca Jepang sama sekali.

## 7. Bug: `.env` gak pernah kebaca oleh proses backend

**Gejala**: `TRANSLATE_PROVIDER=gemini` di `backend/.env` gak ngefek — server tetap pakai LibreTranslate default (endpoint publik yang butuh API key, gagal dengan pesan "Visit https://portal.libretranslate.com to get an API key").

**Root cause**: [config.go](../../backend/internal/config/config.go) cuma baca `os.LookupEnv` langsung — gak ada loader `.env` (`godotenv` dkk) sama sekali. File `.env` di folder backend gak pernah benar-benar masuk ke environment proses kecuali di-export manual ke shell sebelum `air`/`go run` dijalankan.

**Fix**: [main.go](../../backend/cmd/server/main.go) — `godotenv.Load()` (dependency baru: `github.com/joho/godotenv`) dipanggil di awal `main()`, sebelum `config.Load()`. Gak override env var yang udah ada di environment asli (behavior default godotenv), dan gagal-diam kalau `.env` gak ada (kasus normal di produksi/Docker).

## 8. Tuning retry/backoff Gemini client — dua iterasi, verifikasi langsung ke API asli

**Iterasi 1 (salah duga)**: awalnya dikira 31/61 baris gagal karena LibreTranslate/Gemini kena rate limit sesaat akibat request beruntun — `maxAttempts` dinaikkan dari 3 (backoff linear ~1-3s) ke 6 dengan backoff eksponensial sampai ~62s total.

**Ditemukan lewat testing langsung**: satu baris yang masih gagal dites terisolasi — **tetap gagal setelah nunggu 64.5 detik penuh** (attempt ke-6, semua backoff kepakai). Ini bukan pola "pulih setelah nunggu bentar" — kemungkinan besar kuota **harian** API key Gemini yang abis (bukan rate limit per-menit), yang gak akan pulih walau di-backoff berjam-jam dalam sesi yang sama.

**Iterasi 2 (koreksi)**: [client.go](../../backend/internal/gemini/client.go) — `maxAttempts` diturunkan ke 4, backoff diringkas ke 1s/2s/4s (~7s total). Diverifikasi: baris yang sama yang tadinya 64.5s sekarang gagal dalam **9.5 detik** — outcome-nya tetap gagal (karena penyebab aslinya kuota, bukan burst), tapi gak lagi bikin user nunggu lama buat tahu hasilnya. Dites juga batch 27 baris yang masih gagal — semua tetap gagal walau di-retry selama ~3m46s, mengonfirmasi ini beneran kuota, bukan burst sesaat.

## File yang berubah

**Modifikasi**:
- `backend/internal/align/align.go`
- `backend/internal/align/align_test.go`
- `backend/internal/db/models.go`
- `backend/internal/gemini/client.go`
- `backend/internal/httpapi/dto.go`
- `backend/internal/httpapi/router.go`
- `backend/internal/httpapi/scrape_handler.go`
- `backend/internal/httpapi/tracks_handler.go`
- `backend/cmd/server/main.go`
- `backend/go.mod`, `backend/go.sum` (dependency baru: `github.com/joho/godotenv`)

**Baru**:
- `backend/internal/httpapi/ai_reference_handler.go`

Diverifikasi dengan `go build ./...` dan `go test ./...` (semua sukses, gak ada regresi) di setiap tahap, **plus** verifikasi langsung ke server `air` yang jalan lokal (curl ke endpoint asli, query database SQLite langsung) — bukan cuma unit test, karena bug awalnya baru kelihatan lewat data produksi nyata.
