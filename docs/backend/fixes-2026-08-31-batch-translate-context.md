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

## 6. Prompt nambah Rule "Diction" — biar kerasa lirik, bukan chat

Setelah dicoba pada lagu asli, hasil terjemahan Indonesia-nya kebaca terlalu "everyday conversation" (mis. gaya kontraksi/partikel chat kayak "udah", "gak", "kayak", "deh", "kok") padahal user maunya bahasa Indonesia yang baku secara ejaan/diksi — tapi tetap informal secara alamat (`aku`/`kamu`, bukan `saya`/`Anda`, yang memang sudah jadi Rule 1 sebelumnya). Dua sumbu ini (register alamat vs. diksi/ejaan) ternyata gampang ketuker kalau cuma dijelasin lewat satu rule.

[prompt.go](../../backend/internal/llmprompt/prompt.go): `Build`/`BuildBatch` nambah **Rule 2 baru — "Diction"** — di antara Rule 1 (Register) dan rule "translate FULL line" lama (nomornya jadi geser semua): pakai ejaan/diksi standar di bahasa target (bukan singkatan chat/SMS atau filler obrolan lisan), sambil tetap pertahankan alamat informal dari Rule 1. Kasih contoh konkret bahasa Indonesia (`tidak`/`sudah`/`seperti` alih-alih `nggak`/`gak`/`udah`/`kayak`, buang filler `deh`/`sih`/`dong`/`kok` kecuali baris sumbernya memang interjeksi lisan beneran) — sama pola dengan Rule 1 yang juga kasih contoh ID+FR biar model non-ID (mis. saat translate ke bahasa lain) tetap dapat instruksi yang jelas lewat analogi.

## 7. Frontend: counter waktu + pesan loading yang provider-aware

Karena batch translate sekarang bisa betulan makan waktu menit-an di LLM lokal (bukan lagi kedip cepat kayak libretranslate per-baris), spinner polos di tombol Translate gampang kebaca "hang" padahal masih jalan. Ditambah:

- **`GET /api/health`** ([health_handler.go](../../backend/internal/httpapi/health_handler.go)) sekarang ikut balikin `translate_provider` (isinya `s.translatorID`, sama yang dipakai buat namespace cache) — sekadar expose provider yang lagi aktif, bukan endpoint baru.
- **`useElapsedSeconds`** hook baru ([useElapsedSeconds.ts](../../frontend/src/hooks/useElapsedSeconds.ts)): hitung detik berjalan sejak sebuah mutation mulai pending, reset begitu selesai. Dihitung dari wall-clock (`Date.now()` diff), bukan sekadar hitung tick, biar gak ngaco kalau tab di-background dan `setInterval` di-throttle browser.
- **`TranslatePanel.tsx`**: tombol Translate pas pending sekarang nampilin `Menerjemahkan… mm:ss`, plus caption di bawahnya yang beda teks tergantung `translate_provider` — kasih peringatan eksplisit "bisa beberapa menit" khusus buat `localllm` (yang paling sering bikin orang kira macet), pesan lebih pendek buat `gemini`, generic buat sisanya.

## 8. `Method`/`needs_review` dibenerin, `translate_model` diekspos gantiin `translate_provider` mentah

User nemu tiga hal nyata pas ngetes ulang: (a) hasil translate lewat LLM (localllm/gemini) tetap ke-tag `method: "mt"`, padahal frontend udah lama punya badge ungu "AI" yang gak kepakai; (b) `needs_review` selalu `false` abis translate, padahal user mau itu `true` biar ke-flag buat direview; (c) nampilin string `"localllm"` mentah ke frontend bakal aneh kalau di-hosting buat orang lain (bukan brand/model, cuma nama internal package).

- **Method AI vs MT** ([db/models.go](../../backend/internal/db/models.go), [main.go](../../backend/cmd/server/main.go), [translate_handler.go](../../backend/internal/httpapi/translate_handler.go)): `MethodAI` ternyata udah lama ada di enum + [MethodBadge.tsx](../../frontend/src/components/MethodBadge.tsx) (badge fuchsia "AI") — sisa dari rencana lama "Cabang B (AI)" yang gak jadi dibikin sebagai endpoint terpisah. Sekarang `main.go` hitung `resolvedIsLLM` (true buat gemini/localllm, false buat libretranslate) dan lempar ke `Server`; `handleTranslateTrack` pakai itu buat milih `MethodAI` vs `MethodMT`.
- **`needs_review` di-set `true` abis translate** ([translate_handler.go](../../backend/internal/httpapi/translate_handler.go)) — sama kayak hasil scrape+align, karena sama-sama "output mesin yang belum dilihat manusia". Supaya flag-nya beneran bisa clear (bukan cuma nyala terus), `handleUpdateLine` ([tracks_handler.go](../../backend/internal/httpapi/tracks_handler.go)) sekarang set `false` begitu user edit manual, dan `handleRevertLine` ([lines_handler.go](../../backend/internal/httpapi/lines_handler.go)) set balik ke `true` (revert = balik ke saran otomatis yang belum direview lagi). Banner ringkasan di [EditorPage.tsx](../../frontend/src/pages/EditorPage.tsx) diupdate teksnya biar gak lagi ngasumsikan penyebabnya selalu scrape+align.
- **`translate_model` di health** ([health_handler.go](../../backend/internal/httpapi/health_handler.go)): `main.go` sekarang juga nyimpen `resolvedModel` (nilai `LOCAL_LLM_MODEL`/`GEMINI_MODEL` yang beneran dipakai, kosong buat libretranslate) dan expose lewat `GET /api/health`. [TranslatePanel.tsx](../../frontend/src/components/TranslatePanel.tsx) pakai ini buat nampilin nama model asli di caption loading, bukan hardcode "LLM lokal" doang atau nama brand yang di-guess — jadi tetep akurat kalau modelnya diganti nanti tanpa perlu ubah kode lagi.

## Yang sengaja tidak diubah

- LM Studio config tuning (context_length, presence_penalty, thinking mode dsb) — itu rekomendasi terpisah ke user, bukan perubahan kode.
- `handleGetAIReference`/`ai_reference_handler.go` — tetap per-baris + cache, route-nya juga masih nonaktif. Doc comment-nya diupdate biar gak lagi salah ngasumsikan `handleTranslateTrack` masih pakai cache yang sama.
- Cache untuk batch translate — sengaja di-drop, bukan di-redesign. Kalau nanti mau dibenerin, perlu skema baru (cache key per grup/song, bukan per baris) karena satu `TranslateBatch` call sekarang mewakili banyak baris sekaligus.

## File yang berubah

**Baru**:
- `backend/internal/llmprompt/prompt_test.go`
- `backend/internal/gemini/client_test.go`
- `frontend/src/hooks/useElapsedSeconds.ts`
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
- `backend/internal/httpapi/health_handler.go` (`translate_provider` + `translate_model` di response)
- `backend/internal/db/models.go` (`MethodAI` doc comment diupdate — bukan lagi "reserved, not implemented")
- `backend/internal/httpapi/lines_handler.go` (`handleRevertLine` set `NeedsReview = true`)
- `frontend/src/api/types.ts` (`TranslateResponse` kehilangan cache_hits/cache_misses; `HealthResponse` baru + `translate_model`)
- `frontend/src/api/client.ts` (`api.health()`)
- `frontend/src/components/TranslatePanel.tsx` (tampilan hasil translate disesuaikan; counter waktu + caption provider+model-aware saat pending)
- `frontend/src/pages/EditorPage.tsx` (teks banner needs_review gak lagi ngasumsikan selalu dari scrape+align)

Diverifikasi dengan `go build ./...`, `go vet ./...`, `go test ./...` (semua sukses) dan `npx tsc -b` di frontend (gak ada type error). Belum diverifikasi manual ke server LM Studio user yang sesungguhnya — perlu dicoba langsung untuk konfirmasi soal waktu total & konsistensi hasil.
