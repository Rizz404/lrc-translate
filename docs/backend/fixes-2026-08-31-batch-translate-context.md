# Backend Fix: Whole-track batch translate for LLM context (2026-08-31)

> Lanjutan dari [docs/backend/fixes-2026-08-31-local-llm-provider.md](fixes-2026-08-31-local-llm-provider.md). Poin "Yang sengaja tidak diubah" di doc itu bilang `translate_handler.go`/`Translator` interface tidak berubah — sesi ini justru mengubah keduanya, karena ternyata desain lama yang dipertahankan itu sendiri yang jadi masalah.

## Latar belakang

User laporan local LLM-nya (Qwen3.6 35B via LM Studio) butuh ~15 menit buat translate satu lagu dan gak kelar-kelar. Diagnosis awal soal tuning LM Studio (context_length 128K kegedean, presence_penalty 1.5 kegedean untuk reasoning model, thinking mode dsb — lihat percakapan sesi ini). Tapi di tengah diagnosis, kebongkar hal yang lebih fundamental: `handleTranslateTrack` ternyata mengirim **satu request HTTP terpisah per baris lirik**, masing-masing independen — bukan cuma alasan lambat (reasoning-model overhead paid per line, bukan sekali), tapi juga alasan kualitas: `llmprompt.Build`'s prompt sendiri secara eksplisit bilang ke model "you won't see the other lines". Efeknya translation antar baris kadang gak nyambung (register/pronoun gak konsisten, terjemahan frasa yang sama beda-beda) karena model betulan gak pernah lihat konteks lagu.

User minta dirombak: kirim semua baris sekaligus supaya AI-nya paham konteks penuh, dan cache untuk translate di-drop dulu (mau dibenerin belakangan, di luar sesi ini).

## 1. `Translator` interface nambah `TranslateBatch`

[router.go](../../backend/internal/httpapi/router.go): interface `Translator` sekarang punya dua method — `Translate` (dipertahankan, dipakai `handleGetAIReference` yang butuh reference per-baris + cache per-baris) dan `TranslateBatch(ctx, lines []string, sourceLang, targetLang string) ([]string, error)` yang baru — dipakai `handleTranslateTrack`.

## 2. Prompt batch: `llmprompt.BuildBatch` + `llmprompt.ParseBatch`

[prompt.go](../../backend/internal/llmprompt/prompt.go): `BuildBatch` mirip `Build` (aturan register/kelengkapan/makna-vs-literal yang sama), tapi mengirim semua baris sekaligus (dinomori dalam prompt) dan minta balikan **JSON array of strings**, panjang & urutan persis sama dengan input — supaya bisa di-zip balik ke baris asalnya secara posisional. `ParseBatch` men-decode balikan itu, toleran terhadap markdown code fence (```json ... ```, sering muncul meski sudah diminta jangan), dan **error keras kalau jumlah elemen gak pas** — gak ada cara aman nebak baris mana hilang/gabung kalau count-nya beda, jadi lebih baik gagal jelas daripada translation kegeser satu baris ke bawah untuk sisa lagu.

## 3. Tiga backend implement `TranslateBatch`

- **`gemini`**: satu `generateContent` call untuk seluruh batch, `maxOutputTokens` diskalakan (`batchMaxOutputTokens`: ~300/baris, floor 1024, ceiling 8192). Mismatch count dari `ParseBatch` diperlakukan **retryable** (temperature 0.3 kasih kans hasil beda tiap attempt) memakai backoff yang sama dengan `Translate`. `doTranslate` lama dipecah jadi `doRequest` (HTTP + parsing candidate text mentah) dipakai bareng oleh `Translate`/`TranslateBatch`.
- **`localllm`**: sama polanya, `max_tokens` diskalakan (`batchMaxTokens`: `maxTokens(2048) * jumlah baris`, floor `maxTokens`, ceiling 32768) — reasoning model butuh budget reasoning kurang lebih segitu dikali jumlah baris. **Timeout HTTP terpisah**: `httpBatch` (15 menit) vs `http` (120s) yang lama — satu request batch sekarang menanggung generasi+reasoning seluruh lagu sebelum return, jauh lewat 120s di hardware lokal yang pas-pasan. `stripThinking` tetap dipakai sebelum `ParseBatch`.
- **`libretranslate`**: NMT biasa, gak ada "context" yang bisa dipakai dari batching — tapi tetap implement `TranslateBatch` pakai dukungan native API-nya (`q` sebagai array, balikannya `translatedText` juga array) supaya caller (`handleTranslateTrack`) gak perlu percabangan khusus per provider. `doTranslate` lama dipecah jadi `doRequest` (kirim request, balikin raw body) dipakai bareng oleh `Translate`/`TranslateBatch`.

## 4. `handleTranslateTrack` dirombak: grouping per-lang, bukan goroutine per-baris

[translate_handler.go](../../backend/internal/httpapi/translate_handler.go): baris target dikelompokkan berdasarkan source lang (hampir selalu cuma 1 grup — `TranslateSourceOriginal` selalu pakai `trackLang` yang sama; `TranslateSourceScrape` bisa lebih dari 1 kalau baris-baris scrape punya `TranslationLang` beda), lalu **satu `TranslateBatch` call per grup**. Baris yang sudah same-language-as-target tetap di-skip sebelum masuk grup (perilaku lama dipertahankan). `maxConcurrentTranslations` (goroutine+semaphore lama) dihapus — gak relevan lagi, jumlah grup biasanya 1 dan tiap grup cuma 1 HTTP call (yang sudah retry sendiri di client-nya).

## 5. Cache di-drop dari `handleTranslateTrack` (sementara)

Sesuai permintaan eksplisit user. `translateOneCached`/`translationCacheKey` **tidak dihapus** — masih dipakai `handleGetAIReference` (fitur terpisah, per-baris, route-nya sendiri masih nonaktif — lihat router.go) — cuma `handleTranslateTrack` berhenti memanggilnya. `TranslateResponse` (dto.go) kehilangan field `cache_hits`/`cache_misses`; frontend (`types.ts`, `TranslatePanel.tsx`) disesuaikan (nampilin jumlah baris diterjemahkan alih-alih itu). `AIReferenceResponse` gak disentuh, masih punya `cache_hits`/`cache_misses` sendiri.

## Yang sengaja tidak diubah

- LM Studio config tuning (context_length, presence_penalty, thinking mode dsb) — itu rekomendasi terpisah ke user, bukan perubahan kode.
- `handleGetAIReference`/`ai_reference_handler.go` — tetap per-baris + cache, route-nya juga masih nonaktif. Doc comment-nya diupdate biar gak lagi salah ngasumsikan `handleTranslateTrack` masih pakai cache yang sama.
- Cache untuk batch translate — sengaja di-drop, bukan di-redesign. Kalau nanti mau dibenerin, perlu skema baru (cache key per grup/song, bukan per baris) karena satu `TranslateBatch` call sekarang mewakili banyak baris sekaligus.

## File yang berubah

**Baru**:
- `backend/internal/llmprompt/prompt_test.go`
- `backend/internal/gemini/client_test.go`
- `docs/backend/fixes-2026-08-31-batch-translate-context.md` (dokumen ini)

**Modifikasi**:
- `backend/internal/httpapi/router.go` (`Translator` interface)
- `backend/internal/httpapi/translate_handler.go` (grouping per-lang, drop cache/goroutine)
- `backend/internal/httpapi/ai_reference_handler.go` (doc comment saja)
- `backend/internal/httpapi/dto.go` (`TranslateResponse` kehilangan cache_hits/cache_misses)
- `backend/internal/llmprompt/prompt.go` (`BuildBatch`, `ParseBatch`)
- `backend/internal/gemini/client.go` (`TranslateBatch`, `doRequest` refactor, `baseURL` field buat testability)
- `backend/internal/localllm/client.go` (`TranslateBatch`, `doRequest` refactor, `httpBatch` client)
- `backend/internal/localllm/client_test.go` (test `TranslateBatch`)
- `backend/internal/libretranslate/client.go` (`TranslateBatch`, `doRequest` refactor)
- `backend/internal/libretranslate/client_test.go` (test `TranslateBatch`)
- `frontend/src/api/types.ts` (`TranslateResponse` kehilangan cache_hits/cache_misses)
- `frontend/src/components/TranslatePanel.tsx` (tampilan hasil translate disesuaikan)

Diverifikasi dengan `go build ./...`, `go vet ./...`, `go test ./...` (semua sukses) dan `npx tsc -b` di frontend (gak ada type error). Belum diverifikasi manual ke server LM Studio user yang sesungguhnya — perlu dicoba langsung untuk konfirmasi soal waktu total & konsistensi hasil.
