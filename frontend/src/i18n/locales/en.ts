export const en = {
  common: {
    language: "Language",
  },

  search: {
    tagline:
      "Find a song, pull its synced lyrics, translate line by line - all in one editor.",
    placeholder: "Search song title…",
  },

  candidates: {
    empty: "No results. Try another title.",
    noAlbum: "no album",
    noSyncedLyrics: "no synced lyrics",
  },

  editor: {
    back: "Search another song",
    resetScrape: "Reset scrape search",
    resetScrapeTitle:
      "Clear the scrape search results (URL, raw text, alignment status) without changing lines that are already aligned",
    resetAll: "Reset all lines",
    resetAllTitle:
      "Remove all romanization/translation and restore every line to the original LRC",
    resetAllConfirm:
      "Remove all romanization/translation and restore every line to the original LRC? This cannot be undone.",
    needsReview_one:
      "{{count}} line needs a manual check - not necessarily wrong, but nobody has confirmed it yet. It may come from scrape+align (the position is only a guess and could be paired with the wrong original line) or from MT/AI (automatic translation, meaning not guaranteed to be 100% accurate). Marked with a yellow border below, check them one by one - the flag disappears once the line is edited manually.",
    needsReview_other:
      "{{count}} lines need a manual check - not necessarily all wrong, but nobody has confirmed them yet. They may come from scrape+align (the position is only a guess and could be paired with the wrong original line) or from MT/AI (automatic translation, meaning not guaranteed to be 100% accurate). Marked with a yellow border below, check them one by one - the flag disappears once a line is edited manually.",
  },

  line: {
    translationPlaceholder: "Translation…",
    revertTitle: "Revert to the initial suggestion (if any)",
  },

  method: {
    none: "not translated",
    mt: "MT",
    ai: "AI",
    scrape: "scrape",
    manual: "manual",
  },

  copyBar: {
    copy: "Copy LRC",
    copied: "Copied!",
    download: "Download .lrc",
    options: "Export options",
    copyFailed: "Failed to copy to clipboard. Select the text manually.",
    mainLyricHint:
      "The main lyric follows what the editor currently shows: <mode>{{mode}}</mode> - change it with the Original/Romaji toggle next to Romanize.",
    includeTranslation: "Include translation",
    includeTranslationHint: "Added as a second line at the same timestamp.",
    wrapInParens: 'Wrap in parentheses "( )"',
  },

  romanize: {
    button: "Romanize",
    original: "Original",
    romaji: "Romaji",
    protectedTitle_one:
      "{{count}} line already uses official/manual romanization - it won't be overwritten, only the other lines are processed.",
    protectedTitle_other:
      "{{count}} lines already use official/manual romanization - they won't be overwritten, only the other lines are processed.",
    protectedNote_one:
      "{{count}} line with official/manual romanization - won't be overwritten",
    protectedNote_other:
      "{{count}} lines with official/manual romanization - won't be overwritten",
    kept_one: "{{count}} line kept",
    kept_other: "{{count}} lines kept",
  },

  translate: {
    button: "Translate",
    sourceScrape: "From scrape data",
    sourceScrapeTitle:
      "Translate from the existing scraped translation, not the original lyrics",
    sourceOriginal: "Overwrite from original lyrics",
    sourceOriginalTitle:
      "Translate from the original lyrics, overwriting whatever is there",
    alreadyInLang: " (already in this language)",
    blockedOriginal: "is the same as the original lyrics' language",
    blockedScrape: "is the same as the scraped result's language",
    clear: "Clear translation",
    clearTitle: "Target language {{reason}} - clear the translation instead",
    inProgress: "Translating… {{elapsed}}",
    captionLocalLlm:
      "Translating with a local LLM{{model}} - the longer the song, the longer it takes (can be several minutes for a full song).",
    captionGemini: "Translating with Gemini{{model}}…",
    captionDefault: "Translating…",
    result_one: "{{count}} line translated · done in {{elapsed}}",
    result_other: "{{count}} lines translated · done in {{elapsed}}",
    resultWithFailed_one:
      "{{count}} line translated, {{failed}} failed · done in {{elapsed}}",
    resultWithFailed_other:
      "{{count}} lines translated, {{failed}} failed · done in {{elapsed}}",
    errorAfter: "{{message}} (after {{elapsed}})",
    cleared_one: "Translation cleared on {{count}} line",
    cleared_other: "Translation cleared on {{count}} lines",
  },

  scrape: {
    title: "Find translations on the web (scraping)",
    disclaimer:
      'Last resort, not the first thing to try - use Translate (Gemini/LibreTranslate) above first. Alignment here is only a position guess, and has paired lines wrongly before without it being obvious per line (each line still looks sensible, it\'s just swapped with the wrong original line - see the bug history in <code>docs/backend/fixes-2026-08-25-scrape-alignment.md</code>). The result is always marked <em>"needs review"</em>, never verified.',
    auto: "Auto search (utatime.com)",
    align: "Apply Alignment",
    languagePrompt:
      "This site isn't utatime.com - the scraped language can't be detected automatically. Enter its language code (e.g. <code>en</code>) so the translate feature knows which language this is:",
    languagePlaceholder: "en",
    autoDiscovered: "Found automatically: ",
    source: "Source: ",
    romanizedTaken:
      "The site's official romanization was captured too - it will replace the automatic romanization on aligned lines.",
    showRaw_one: "Show raw scraped text ({{count}} line)",
    showRaw_other: "Show raw scraped text ({{count}} lines)",
    hideRaw_one: "Hide raw scraped text ({{count}} line)",
    hideRaw_other: "Hide raw scraped text ({{count}} lines)",
    aligned_one:
      "{{count}} line aligned (translation) - check them one by one, lines are marked amber below.",
    aligned_other:
      "{{count}} lines aligned (translation) - check them one by one, lines are marked amber below.",
    alignedWithRomanized_one:
      "{{count}} line aligned (translation + romanization) - check them one by one, lines are marked amber below.",
    alignedWithRomanized_other:
      "{{count}} lines aligned (translation + romanization) - check them one by one, lines are marked amber below.",
    pasteLink: "or paste a page link manually →",
    pasteLinkPrompt:
      "Find a page with this song's translation yourself, then paste its link:",
    urlPlaceholder: "https://…",
    scrapePage: "Scrape this page",
  },
};
