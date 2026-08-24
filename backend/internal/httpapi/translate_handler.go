package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appdb "lrc-translate/backend/internal/db"
)

// Batch translate calls are throttled to avoid bursting a rate-limited
// LibreTranslate instance (public or self-hosted) — see plan-extended.md
// "Handling LibreTranslate rate-limit/downtime".
const maxConcurrentTranslations = 2

// POST /api/tracks/:id/translate { target_lang, line_ids?, source? }
func (s *Server) handleTranslateTrack(c *gin.Context) {
	track, err := s.loadTrack(c.Param("id"))
	if err != nil {
		respondTrackLookupError(c, err)
		return
	}

	var req TranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	source := req.Source
	if source == "" {
		source = TranslateSourceOriginal
	}

	// Whole-track guard: track.Language is one value for the entire track,
	// so a target_lang matching it means EVERY TranslateSourceOriginal line
	// would be a same-language no-op translate. Reject upfront with a clear
	// error rather than silently burning MT calls on it — the frontend
	// should offer "hapus translation" (see handleClearTranslation) instead.
	// (TranslateSourceScrape lines are checked per-line below instead, since
	// each line's scrape language can differ.)
	if source == TranslateSourceOriginal && track.Language != "" && track.Language == req.TargetLang {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_lang sama dengan bahasa lirik asli track ini — gunakan aksi hapus translation, bukan translate"})
		return
	}

	targets := selectLines(track.Lines, req.LineIDs)

	trackLang := track.Language
	if trackLang == "" {
		trackLang = "auto"
	}

	type outcome struct {
		translated string
		cacheHit   bool
		err        error
	}
	outcomes := make([]outcome, len(targets))

	sem := make(chan struct{}, maxConcurrentTranslations)
	var wg sync.WaitGroup
	for i, line := range targets {
		if line.Original == "" {
			continue
		}

		// Resolve per-line source text/language: TranslateSourceScrape
		// chains off this line's own already-scraped translation (e.g. EN
		// from utatime.com) when it has one, falling back to the original
		// lyric otherwise so the batch still completes for that line.
		text, lang := line.Original, trackLang
		if source == TranslateSourceScrape && line.Method == appdb.MethodScrape && line.Translation != "" {
			text = line.Translation
			lang = line.TranslationLang
			if lang == "" {
				lang = "auto"
			}
		}

		if lang != "" && lang != "auto" && lang == req.TargetLang {
			outcomes[i] = outcome{err: fmt.Errorf("baris ini sudah dalam bahasa target (%s)", lang)}
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(i int, text, lang string) {
			defer wg.Done()
			defer func() { <-sem }()
			translated, cacheHit, err := s.translateOneCached(c.Request.Context(), text, lang, req.TargetLang)
			outcomes[i] = outcome{translated: translated, cacheHit: cacheHit, err: err}
		}(i, text, lang)
	}
	wg.Wait()

	resp := TranslateResponse{Lines: []LineDTO{}}
	for i, line := range targets {
		if line.Original == "" {
			continue
		}
		o := outcomes[i]
		if o.err != nil {
			resp.Failed = append(resp.Failed, struct {
				LineID uint   `json:"line_id"`
				Error  string `json:"error"`
			}{LineID: line.ID, Error: o.err.Error()})
			continue
		}

		if o.cacheHit {
			resp.CacheHits++
		} else {
			resp.CacheMisses++
		}

		line.Translation = o.translated
		line.TranslationLang = ""
		line.Method = appdb.MethodMT
		if err := s.db.Save(&line).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save translated line: " + err.Error()})
			return
		}
		resp.Lines = append(resp.Lines, toLineDTO(line))
	}

	c.JSON(http.StatusOK, resp)
}

// POST /api/tracks/:id/translate/clear { line_ids? }
// Wipes Translation(/TranslationLang/Method) back to empty/"none" for the
// targeted lines, without touching Original/Romanized/Timestamp. This is
// the alternative action offered when a translate would otherwise be a
// same-language no-op (see the track.Language guard in handleTranslateTrack
// and the per-line guard above it) — deleting is the sensible move there,
// not translating.
func (s *Server) handleClearTranslation(c *gin.Context) {
	track, err := s.loadTrack(c.Param("id"))
	if err != nil {
		respondTrackLookupError(c, err)
		return
	}

	var req ClearTranslationRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	targets := selectLines(track.Lines, req.LineIDs)

	resp := ClearTranslationResponse{Lines: []LineDTO{}}
	for _, line := range targets {
		if line.Translation == "" && line.Method == appdb.MethodNone {
			continue // already clear, nothing to do
		}

		line.Translation = ""
		line.TranslationLang = ""
		line.Method = appdb.MethodNone
		line.SuggestedTranslation = ""
		line.SuggestedMethod = ""
		line.SuggestedTranslationLang = ""
		line.NeedsReview = false

		if err := s.db.Save(&line).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear line: " + err.Error()})
			return
		}
		resp.Lines = append(resp.Lines, toLineDTO(line))
	}

	c.JSON(http.StatusOK, resp)
}

// translateOneCached checks translation_cache before calling out to
// LibreTranslate, and writes a cache entry on a fresh call. This is what
// makes repeat translate requests (same text/lang pair, even across
// different tracks) avoid hitting the external API again.
func (s *Server) translateOneCached(ctx context.Context, text, sourceLang, targetLang string) (translated string, cacheHit bool, err error) {
	key := translationCacheKey(text, sourceLang, targetLang, "libretranslate")

	var cached appdb.TranslationCache
	err = s.db.Where("cache_key = ?", key).First(&cached).Error
	if err == nil {
		return cached.TranslatedText, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, err
	}

	translated, err = s.translator.Translate(ctx, text, sourceLang, targetLang)
	if err != nil {
		return "", false, err
	}

	// Best-effort cache write: a failure here shouldn't fail the translate
	// request itself, just means this result won't be reused next time.
	s.db.Create(&appdb.TranslationCache{
		CacheKey:       key,
		SourceText:     text,
		TranslatedText: translated,
		Provider:       "libretranslate",
	})

	return translated, false, nil
}

func translationCacheKey(text, sourceLang, targetLang, provider string) string {
	sum := sha256.Sum256([]byte(text + "|" + sourceLang + "|" + targetLang + "|" + provider))
	return hex.EncodeToString(sum[:])
}

// selectLines returns the lines matching ids, or all lines if ids is empty.
func selectLines(lines []appdb.Line, ids []uint) []appdb.Line {
	if len(ids) == 0 {
		out := make([]appdb.Line, len(lines))
		copy(out, lines)
		return out
	}
	idSet := make(map[uint]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var out []appdb.Line
	for _, l := range lines {
		if idSet[l.ID] {
			out = append(out, l)
		}
	}
	return out
}
