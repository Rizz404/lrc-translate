# Frontend Fixes: Scrape Alignment Review UI (2026-08-25)

> Catatan fitur baru di UI editor untuk membantu review hasil scrape+align, lahir dari investigasi bug dobel/salah-posisi yang dilaporkan user (lihat [docs/backend/fixes-2026-08-25-scrape-alignment.md](../backend/fixes-2026-08-25-scrape-alignment.md) untuk root cause & fix backend-nya). Referensi historis — kalau ada regresi terkait area ini, mulai dari sini.

## Latar belakang

Setelah root cause bug alignment ketemu di backend, muncul pertanyaan user: gimana caranya review baris `needs_review` yang salah kalau dia gak bisa baca huruf Jepang/romaji? Chip perbandingan aja gak cukup kalau reviewer gak bisa menilai mana yang cocok secara makna. Ini melahirkan 2 lapis fitur UI baru di bawah.

## 1. Komponen baru: `ScrapeReference` — chip perbandingan raw scrape

**Tujuan**: tampilkan baris scraped mentah (prev/matched/next) di bawah tiap field (lirik saat mode Romaji, dan terjemahan) yang sumbernya dari scrape — supaya mismatch antara original dan translation kelihatan tanpa perlu buka viewer teks mentah terpisah.

**Implementasi** ([ScrapeReference.tsx](../../frontend/src/components/ScrapeReference.tsx), baru):
- Terima `context: LineScrapeContext` (`{prev, matched, next}`) dari `Line.scrape_context` (lihat [types.ts](../../frontend/src/api/types.ts)) dan `current` (nilai field yang lagi aktif) buat nandain chip mana yang udah dipakai (ring hijau).
- Tiap chip klik langsung apply teksnya ke field terkait (lewat `onPick` callback) — reuse jalur `updateLine` yang sudah ada, otomatis snapshot ke `SuggestedTranslation` + aktifkan tombol Revert (mekanisme "pilih scrape atau manual" yang sebenarnya sudah ada, cuma dibikin actionable).
- **Responsive**: `flex-wrap` + `max-w-[55vw] sm:max-w-[240px] truncate` — di HP chip wrap ke baris baru (bukan overflow horizontal), `title` attribute nampilin teks penuh saat hover di desktop.
- **Hint discoverability**: label "mentah" punya `title` tooltip penjelasan, tiap chip punya `title="Klik untuk pakai: ..."` — ditambahkan setelah user eksplisit minta info kalau chip itu klik-able.

**Terintegrasi di** [LineRow.tsx](../../frontend/src/components/LineRow.tsx): chip translation muncul di bawah textarea terjemahan (selalu, kalau ada context), chip lyric/romanized muncul di bawah textarea lirik (cuma saat mode Romaji aktif, karena original Jepang itu ground truth LRC, bukan hasil align).

## 2. Temuan: chip prev/next gak cukup buat user yang gak baca Jepang

Setelah audit manual seluruh lagu (lihat catatan backend poin 3), ketahuan chip di atas cuma membantu kalau reviewer bisa menilai kecocokan makna Jepang↔Inggris. User eksplisit bilang gak bisa baca romaji-nya — jadi chip "prev"/"next" (yang isinya bahasa sumber scrape, biasanya sama bahasa dengan translation, misal Inggris — tapi gak selalu solve masalahnya kalau reviewer gak bisa nilai baris mana yang "lebih pas" secara makna) gak banyak membantu di luar kasus dobel-persis yang gampang keliatan.

**Keputusan**: bangun referensi ke-3 yang independen dari heuristik alignment — lihat poin 3.

## 3. Chip "AI" — referensi machine translation, selalu benar posisinya

**Cara kerja**: [ScrapeReference.tsx](../../frontend/src/components/ScrapeReference.tsx) diperluas terima prop opsional `aiText` — dirender sebagai chip ke-4 dengan warna beda (sky/biru, bukan amber/slate seperti chip prev/matched/next) dan label "AI" di depan teksnya, supaya jelas kelihatan beda sumber (MT langsung, bukan tebakan alignment scrape).

**Trigger**: tombol baru "Bandingkan dengan AI" di [EditorPage.tsx](../../frontend/src/pages/EditorPage.tsx), muncul di banner `needsReviewCount > 0` — sengaja **on-demand** (bukan otomatis jalan pas align), karena manggil MT beneran costs waktu/API call:
- `aiReferenceMutation` — infer `target_lang` dari `translation_lang` baris `needs_review` pertama yang punya nilai (fallback `"en"`), kirim `line_ids` cuma baris `needs_review` (efisien, gak translate ulang baris yang udah beres).
- Status sukses/gagal ditampilkan inline — termasuk **tombol "coba lagi baris yang gagal"** yang cuma re-request baris di `failed[]` (bukan full re-run), dan pesan yang jujur soal kemungkinan kuota provider abis (bukan sekadar "coba lagi sebentar lagi" — lihat catatan backend poin 8 kenapa pesan ini direvisi).

**API client** ([client.ts](../../frontend/src/api/client.ts), [types.ts](../../frontend/src/api/types.ts)): `api.aiReference(trackId, {target_lang, line_ids})` + tipe `AIReferenceRequest`/`AIReferenceResponse`, `LineScrapeContexts.ai?: string`.

**Verifikasi nyata**: dites ke server asli — contoh baris `00:35.39` (original "A, B, C, D, E, F, G"), chip translation (scrape, salah): "No matter what i chose"; chip AI (benar): "A, B, C, D, E, F," — mismatch-nya kelihatan jelas cuma dari teks Inggris, gak perlu baca huruf Jepang sama sekali. Ini yang memvalidasi keseluruhan pendekatan fitur ini.

## File yang berubah

**Modifikasi**:
- `frontend/src/api/client.ts`
- `frontend/src/api/types.ts`
- `frontend/src/components/LineRow.tsx`
- `frontend/src/pages/EditorPage.tsx`

**Baru**:
- `frontend/src/components/ScrapeReference.tsx`

Diverifikasi dengan `npx tsc -b`, `npx oxlint`, dan `npm run build` (semua sukses tanpa error/warning baru) di setiap tahap, **plus** verifikasi visual lewat screenshot asli dari user dan pengecekan manual ke response API sungguhan (bukan cuma type-check) — termasuk trace lengkap kenapa proses "Bandingkan dengan AI" sempat kelamaan (lihat catatan backend poin 8, root cause-nya di backend tapi gejalanya muncul lewat tombol ini).
