# Backend Feature: Local LLM Translate Provider (2026-08-31)

> Lanjutan dari [docs/backend/fixes-2026-08-26-translate-priority.md](fixes-2026-08-26-translate-priority.md), yang menetapkan pola "auto-resolve priority berdasarkan config yang tersedia, bukan runtime fallback per-request". Perubahan sesi ini menambah satu provider baru di depan urutan itu, memakai pola yang sama persis.

## Latar belakang

User punya LLM sendiri yang di-host lokal via LM Studio dan diekspos lewat tunnel ngrok, dan minta itu jadi provider translate utama, dengan Gemini sebagai fallback. Dicek dulu sebelum implementasi:
- `GET /v1/models` di server user (butuh `Authorization: Bearer <token>`, LM Studio-nya diset "require API key") menunjukkan 3 model chat (`qwen3.6-35b-a3b-uncensored-hauhaucs-aggressive` / alias `... (modified)`, `google/gemma-4-26b-a4b-qat`) + 1 model embedding (tidak relevan). User pilih varian `... (modified)` — preset yang sudah dia atur sendiri di LM Studio, jadi id modelnya harus persis string itu (dengan spasi & tanda kurung), bukan slug-nya.
- Test manual `POST /v1/chat/completions` ke model itu mengungkap satu hal penting: **ini reasoning model** (field `reasoning_content` terpisah dari `content`, gaya Qwen3 "thinking"). Dengan `max_tokens: 100`, seluruh budget habis buat reasoning dan `content` jadi kosong (`finish_reason: "length"`) — menerjemahkan "I love you" saja butuh 224 token reasoning. Ini bukan hal yang bisa diabaikan: kalau budget token kekecilan, tiap baris lirik bisa balik kosong.
- Klarifikasi makna "fallback" ke Gemini: disamakan dengan pola KISS yang sudah ada di [fixes-2026-08-26](fixes-2026-08-26-translate-priority.md) — urutan prioritas default konfigurasi saat startup, bukan auto-retry runtime dalam request yang sama. Konsisten dengan keputusan sebelumnya (dan nama commit historisnya sendiri: "prioritize Gemini over LibreTranslate").

## 1. Provider baru: `backend/internal/localllm`

Client baru untuk server OpenAI-compatible chat completions (LM Studio, Ollama's OpenAI shim, dll), strukturnya sengaja dibuat semirip mungkin dengan `internal/libretranslate`/`internal/gemini` yang sudah ada (retry loop 3x dengan backoff linear+jitter pada 429/5xx, tidak retry pada 4xx lain):
- **`maxTokens = 2048`** — jauh lebih besar dari Gemini punya (300), khusus karena reasoning model bisa menghabiskan ratusan token cuma buat "mikir" sebelum nulis jawaban asli (lihat temuan di atas).
- Response kosong dengan `finish_reason: "length"` ditangani sebagai kondisi **retryable** tersendiri (bukan cuma 429/5xx) — kemungkinan besar itu tanda budget kehabisan di tengah reasoning, dan `temperature: 0.3` bikin retry punya peluang beda hasil (reasoning lebih pendek).
- `stripThinking()` menghapus blok `<think>...</think>` dari `message.content` kalau ada — server LM Studio yang dites taruh reasoning di field `reasoning_content` terpisah (jadi ini gak pernah kepakai di setup itu), tapi backend/model lain kadang inline reasoning ke `content` pakai chat template `<think>` — ini jaring pengaman biar gak ketinggalan tag itu.
- Header `Authorization: Bearer <token>` cuma dikirim kalau `apiKey` tidak kosong — server yang gak pasang "require API key" (default LM Studio) tetap kompatibel.
- Timeout HTTP **120s** (vs LibreTranslate/Gemini 30s) — inferensi lokal CPU/GPU-bound tanpa batching skala cloud, dan reasoning model butuh waktu ekstra buat "mikir".

## 2. Prompt LLM diekstrak ke `backend/internal/llmprompt` (shared)

`internal/gemini` sebelumnya punya `buildPrompt`/`langNames`/`langName` sendiri buat instruksi "terjemahin kayak lirik lagu, bukan literal". Karena `localllm` butuh instruksi yang sama persis (sama-sama LLM, beda backend API doang), kode itu dipindah apa adanya ke package baru `internal/llmprompt` (fungsi `Build`), dan `internal/gemini` diupdate untuk memanggilnya — menghindari duplikasi instruksi yang sama di dua tempat.

## 3. Urutan auto-resolve provider diperbarui: `localllm` → `gemini` → `libretranslate`

Sama seperti pola poin 1 di [fixes-2026-08-26](fixes-2026-08-26-translate-priority.md), cuma menambah satu tingkat prioritas baru di depan:
- [config.go](../../backend/internal/config/config.go): field baru `LocalLLMURL`, `LocalLLMAPIKey`, `LocalLLMModel` (env `LOCAL_LLM_URL`/`LOCAL_LLM_API_KEY`/`LOCAL_LLM_MODEL`), default model di-hardcode ke id persis yang user pilih (`Qwen3.6 35B A3B Uncensored HauhauCS Aggressive Q5 K P (modified)`).
- [main.go](../../backend/cmd/server/main.go) switch: tambah case eksplisit `"localllm"` (fatal kalau `LOCAL_LLM_URL` kosong, sama seperti case `"gemini"` fatal tanpa key), dan case `""` (auto) sekarang cek `LocalLLMURL` dulu sebelum `GeminiAPIKey`, baru default ke libretranslate. `resolvedProvider` (buat namespace cache, lihat alasan di fixes-2026-08-26 poin 2) ikut tercakup untuk `"localllm"`.
- Tidak ada retry otomatis runtime antar provider — kalau `localllm` gagal (server/tunnel down) di tengah sesi, pemulihan tetap `TRANSLATE_PROVIDER=gemini` (atau `libretranslate`) + restart, sama seperti sebelumnya untuk Gemini→LibreTranslate.

## 4. Dokumentasi konfigurasi diselaraskan

- [backend/.env.example](../../backend/.env.example) dan root [.env.example](../../.env.example) (dipakai docker-compose): tambah `LOCAL_LLM_URL`/`LOCAL_LLM_API_KEY`/`LOCAL_LLM_MODEL`, urutan komentar provider diurutkan ulang sesuai prioritas baru.
- [docker-compose.yml](../../docker-compose.yml): tambah env `LOCAL_LLM_URL`/`LOCAL_LLM_API_KEY`/`LOCAL_LLM_MODEL` ke service `app`, comment diupdate.
- [README.md](../../README.md): section "Prioritas provider translate" diganti jadi "LLM lokal → Gemini → LibreTranslate → scrape", jelasin cara pakai `localllm` (URL server, token opsional, id model harus persis `GET /v1/models`).

## Yang sengaja tidak diubah

- Pola "startup priority, bukan runtime fallback per-request" — dipertahankan sesuai keputusan sebelumnya, meski kata "fallback" yang dipakai user secara sepintas bisa dibaca sebagai retry otomatis. Diselaraskan lewat konteks: ini istilah yang sama dipakai untuk Gemini→LibreTranslate di sesi sebelumnya, dan itu memang priority order.
- `maxConcurrentTranslations`/`aiReferenceConcurrency` (`translate_handler.go`/`ai_reference_handler.go`) — tetap seperti sebelumnya. Server lokal yang cuma proses 1 request inferensi sekaligus akan meng-queue request yang datang bersamaan (lebih lambat, tapi tetap benar), jadi belum ada alasan kuat buat mengubah konstanta yang ditun untuk CPU headroom LibreTranslate.
- `translate_handler.go`, `Translator` interface — tidak berubah sama sekali, `localllm.Client` cuma implementasi baru dari interface yang sudah ada.

## File yang berubah

**Baru**:
- `backend/internal/llmprompt/prompt.go`
- `backend/internal/localllm/client.go`
- `backend/internal/localllm/client_test.go`

**Modifikasi**:
- `backend/internal/gemini/client.go` (pakai `llmprompt.Build`, hapus duplikat)
- `backend/internal/config/config.go`
- `backend/cmd/server/main.go`
- `backend/.env.example`
- `.env.example`
- `docker-compose.yml`
- `README.md`

Diverifikasi dengan `go build ./...`, `go vet ./...`, dan `go test ./...` (semua sukses, gak ada regresi) — termasuk test manual `curl` langsung ke server LM Studio user (list model, translate satu baris pendek) sebelum coding buat memastikan bentuk response API dan kuirk reasoning-model-nya kepegang di implementasi.
