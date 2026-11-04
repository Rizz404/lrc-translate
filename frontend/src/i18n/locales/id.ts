// Source of truth for translation keys - src/i18n/i18next.d.ts derives the
// typed `t()` keys from this file, so a typo'd key fails at compile time.
// en.ts must mirror these keys (plural forms aside: Indonesian has only
// `_other`, English also needs `_one`).
export const id = {
  common: {
    language: "Bahasa",
  },

  search: {
    tagline:
      "Cari lagu, tarik lirik tersinkron, terjemahkan per baris - semua dalam satu editor.",
    placeholder: "Cari judul lagu…",
  },

  candidates: {
    empty: "Tidak ada hasil. Coba judul lain.",
    noAlbum: "tanpa album",
    noSyncedLyrics: "tanpa synced lyrics",
  },

  editor: {
    back: "Cari lagu lain",
    resetScrape: "Reset pencarian scraping",
    resetScrapeTitle:
      "Bersihkan hasil pencarian scraping (URL, teks mentah, status alignment) tanpa mengubah baris yang sudah ter-alignment",
    resetAll: "Reset semua baris",
    resetAllTitle:
      "Hapus semua romanisasi/terjemahan, kembalikan tiap baris ke LRC asli",
    resetAllConfirm:
      "Hapus semua romanisasi/terjemahan dan kembalikan tiap baris ke LRC asli? Aksi ini tidak bisa dibatalkan.",
    needsReview_other:
      "{{count}} baris perlu dicek manual - belum tentu semuanya salah, tapi belum ada yang mengonfirmasi. Bisa karena hasil scrape+align (posisinya cuma perkiraan, bisa salah pasangan dengan baris aslinya) atau hasil MT/AI (terjemahan otomatis, belum tentu 100% akurat maknanya). Ditandai border kuning di bawah, cek satu-satu - flag-nya hilang begitu baris itu diedit manual.",
  },

  line: {
    translationPlaceholder: "Terjemahan…",
    revertTitle: "Kembalikan ke saran awal (kalau ada)",
  },

  method: {
    none: "belum diterjemahkan",
    mt: "MT",
    ai: "AI",
    scrape: "scrape",
    manual: "manual",
  },

  copyBar: {
    copy: "Copy LRC",
    copied: "Tersalin!",
    download: "Download .lrc",
    options: "Opsi export",
    copyFailed: "Gagal menyalin ke clipboard. Pilih teksnya manual.",
    mainLyricHint:
      "Lirik utama ikut tampilan editor saat ini: <mode>{{mode}}</mode> - ganti lewat tombol Asli/Romaji di sebelah Romanize.",
    includeTranslation: "Sertakan terjemahan",
    includeTranslationHint:
      "Ditambahkan sebagai baris kedua di timestamp yang sama.",
    wrapInParens: 'Bungkus dengan kurung "( )"',
  },

  romanize: {
    button: "Romanize",
    original: "Asli",
    romaji: "Romaji",
    protectedTitle_other:
      "{{count}} baris sudah pakai romanisasi resmi/manual - itu tidak akan ditimpa, cuma baris lain yang diproses.",
    protectedNote_other:
      "{{count}} baris romanisasi resmi/manual - tidak akan ditimpa",
    kept_other: "{{count}} baris dipertahankan",
  },

  translate: {
    button: "Translate",
    sourceScrape: "Dari data scrape",
    sourceScrapeTitle:
      "Terjemahkan dari terjemahan hasil scrape yang sudah ada, bukan lirik asli",
    sourceOriginal: "Timpa dari lirik asli",
    sourceOriginalTitle: "Terjemahkan dari lirik asli, timpa apa pun yang ada",
    alreadyInLang: " (sudah dalam bahasa ini)",
    blockedOriginal: "sama dengan bahasa lirik asli",
    blockedScrape: "sama dengan bahasa hasil scrape",
    clear: "Hapus translation",
    clearTitle: "Target lang {{reason}} - hapus translation sebagai gantinya",
    inProgress: "Menerjemahkan… {{elapsed}}",
    captionLocalLlm:
      "Menerjemahkan pakai LLM lokal{{model}} - makin panjang lagunya, makin lama (bisa beberapa menit untuk lagu penuh).",
    captionGemini: "Menerjemahkan pakai Gemini{{model}}…",
    captionDefault: "Menerjemahkan…",
    result_other: "{{count}} baris diterjemahkan · selesai dalam {{elapsed}}",
    resultWithFailed_other:
      "{{count}} baris diterjemahkan, {{failed}} gagal · selesai dalam {{elapsed}}",
    errorAfter: "{{message}} (setelah {{elapsed}})",
    cleared_other: "{{count}} baris dihapus translation-nya",
  },

  scrape: {
    title: "Cari terjemahan dari web (scraping)",
    disclaimer:
      'Opsi terakhir, bukan yang pertama dicoba - pakai Translate (Gemini/LibreTranslate) di atas dulu. Alignment di sini cuma perkiraan posisi, dan pernah salah pasang baris tanpa kelihatan jelas per-baris (baris individual tetap masuk akal, cuma ketuker sama baris asli yang salah - lihat riwayat bug di <code>docs/backend/fixes-2026-08-25-scrape-alignment.md</code>). Hasilnya selalu ditandai <em>"perlu dicek"</em>, bukan hasil verifikasi.',
    auto: "Cari otomatis (utatime.com)",
    align: "Terapkan Alignment",
    languagePrompt:
      "Situs ini bukan utatime.com - bahasa hasil scrape gak bisa dideteksi otomatis. Isi kode bahasanya (mis. <code>en</code>) biar fitur translate tahu ini bahasa apa:",
    languagePlaceholder: "en",
    autoDiscovered: "Ditemukan otomatis: ",
    source: "Sumber: ",
    romanizedTaken:
      "Romanisasi resmi dari situs ini ikut terambil - akan menggantikan romanisasi otomatis pada baris yang ter-alignment.",
    showRaw_other: "Lihat teks mentah hasil scrape ({{count}} baris)",
    hideRaw_other: "Sembunyikan teks mentah hasil scrape ({{count}} baris)",
    aligned_other:
      "{{count}} baris ter-alignment (terjemahan) - cek satu-satu, baris ditandai amber di bawah.",
    alignedWithRomanized_other:
      "{{count}} baris ter-alignment (terjemahan + romanisasi) - cek satu-satu, baris ditandai amber di bawah.",
    pasteLink: "atau paste link halaman manual →",
    pasteLinkPrompt:
      "Cari sendiri halaman yang berisi terjemahan lagu ini, lalu paste link-nya:",
    urlPlaceholder: "https://…",
    scrapePage: "Scrape halaman ini",
  },
};
