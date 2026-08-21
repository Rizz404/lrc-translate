# Alur Pengerjaan — Multi-Line Synced Lyrics (Versi Simpel)

## Diagram Alur

```
[User cari lagu]
       │
       ▼
[Query LRCLIB] ──(ambigu?)──> [User pilih versi yang benar]
       │
       ▼
[LRC dasar tampil di editor: timestamp + lirik asli]
       │
       ▼
[User bisa edit lirik asli / timestamp secara manual]
       │
       ▼
[User pilih: lanjut terjemahkan, atau langsung copy/download LRC apa adanya]
       │
       ├──> Skip terjemahan → langsung Copy / Download LRC (selesai, tidak lanjut ke bawah)
       │
       └──> Lanjut terjemahkan → [User pilih metode terjemahan]
                │
                ├──> Cabang A: Translate biasa (MT)
                │        │
                │        ▼
                │    Kirim tiap baris ke API MT (Google/DeepL/LibreTranslate)
                │        │
                │        ▼
                │    Hasil otomatis 1:1 sejajar baris (tanpa alignment)
                │
                ├──> Cabang B: AI (LLM) — 🔜 COMING SOON, belum dibangun di rilis awal
                │
                └──> Cabang C: Cari terjemahan (scraping)
                         │
                         ▼
                     Scraping web lirik (Fandom Wiki, dll)
                         │
                         ▼
                     Hasil teks lepas (belum per-baris)
                         │
                         ▼
                     Alignment heuristik posisi/urutan
                         │
                         ▼
                     Ditandai "perlu dicek" (paling rawan meleset)
       │
       ▼
[Terjemahan muncul side-by-side, otomatis di bawah baris asli]
       │
       ▼
[User review & edit manual per baris]
       │
       ▼
[Simpan: timestamp + asli + romaji (jika Jepang) + terjemahan + method per baris]
```

## Rincian Tiap Tahap

### 1. Cari Lagu
- Input: judul + artist (dari user)
- Proses: query ke LRCLIB API
- Kalau hasil ambigu (banyak versi/remix/live) → tampilkan kandidat, user pilih manual
- Output: LRC dasar (timestamp `[mm:ss.xx]` + lirik asli)

### 2. Tampilkan & Edit LRC Dasar
- Lirik asli + timestamp ditampilkan di editor
- User bisa koreksi manual di tahap ini (sebelum lanjut ke terjemahan) kalau ada yang salah dari sumber LRCLIB

### 3. Romanisasi (khusus lagu Jepang)
- Kanji/Kana → Romaji otomatis (Kuromoji/Kuroshiro)
- Kalau lagu Inggris, langkah ini di-skip

### 4. (Opsional) Skip Terjemahan
- Kalau user cuma butuh LRC asli tanpa terjemahan, bisa langsung copy teks LRC atau download file-nya di tahap ini — alur berhenti sampai sini, tidak perlu lanjut ke bawah

### 5. Pilih Metode Terjemahan (kalau lanjut)

| Cabang             | Cara Kerja                                  | Alignment Dibutuhkan?                       | Tag Method | Status        |
| ------------------ | ------------------------------------------- | ------------------------------------------- | ---------- | ------------- |
| A. Translate biasa | API MT per baris                            | Tidak (otomatis 1:1)                        | `mt`       | Rilis awal    |
| B. AI              | LLM per baris/batch + validasi jumlah baris | Minim (hanya validasi, retry jika mismatch) | `ai`       | 🔜 Coming soon |
| C. Cari terjemahan | Scraping web lirik                          | Ya (heuristik posisi)                       | `scrape`   | Rilis awal    |

### 6. Tampilkan Side-by-Side
- Hasil terjemahan otomatis muncul di bawah tiap baris asli, di editor yang sama
- Baris dari cabang C diberi highlight khusus (mis. warna beda) sebagai penanda "perlu dicek duluan"

### 7. Review & Edit Manual
- User bisa ubah baris terjemahan mana pun
- Baris yang diedit user → tag method berubah jadi `manual`
- Simpan draft otomatis sebelum edit (supaya bisa "revert ke saran awal" kalau user salah edit)

### 8. Simpan
- Data final per baris: `time_ms`, `timestamp`, `original`, `romanized` (jika ada), `translation`, `method`
- User bisa regenerate ulang per baris (ganti metode) tanpa perlu proses ulang seluruh lagu
- Cache hasil per lagu supaya tidak manggil API MT/AI berulang untuk lagu yang sama

## Skema Data (Ringkas — DATA DUMMY, bukan lagu/lirik nyata)

> Catatan: seluruh nilai di bawah ini contoh placeholder untuk menunjukkan struktur field saja, bukan kutipan dari lagu manapun.

```json
{
  "track_id": "dummy-track-0001",
  "title": "<contoh judul lagu>",
  "artist": "<contoh nama artis>",
  "lines": [
    {
      "time_ms": 0,
      "timestamp": "[00:00.00]",
      "original": "<contoh baris lirik asli>",
      "romanized": "<contoh romanisasi jika Jepang>",
      "translation": "<contoh baris terjemahan>",
      "method": "mt"
    }
  ]
}
```

## Catatan Prioritas Build

1. Alur dasar: cari lagu → tarik LRC → copy/download langsung (tanpa terjemahan) — ini fondasi paling minimal, bangun paling pertama
2. Cabang A (MT) — paling gampang di antara opsi terjemahan, bangun berikutnya sebagai baseline
3. Fitur edit manual + tag `method` — bangun bareng cabang A karena strukturnya dipakai semua cabang
4. Cabang C (scraping + alignment) — bangun setelah A stabil, karena paling kompleks dan paling jarang jadi pilihan utama user
5. Cabang B (AI) — 🔜 coming soon, ditunda ke rilis berikutnya setelah A & C stabil
