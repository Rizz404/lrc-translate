package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appdb "lrc-translate/backend/internal/db"
)

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
		err        error
	}
	outcomes := make([]outcome, len(targets))

	// queued is one line lined up for a TranslateBatch call: idx is its
	// position in targets/outcomes, so a batch's results (returned in the
	// same order they were sent) can be written back to the right line.
	type queued struct {
		idx  int
		text string
	}
	// Grouped by source language rather than one call for the whole track,
	// since TranslateSourceScrape lines can each carry their own detected
	// language (see below) — almost always just one group in practice
	// (TranslateSourceOriginal always shares trackLang), so this is usually
	// exactly one TranslateBatch call for the entire track.
	groups := map[string][]queued{}
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

		groups[lang] = append(groups[lang], queued{idx: i, text: text})
	}

	// One TranslateBatch call per source-language group, so an LLM backend
	// sees the whole song (or at least the whole group) together as context
	// instead of the old one-request-per-line design — see
	// llmprompt.BuildBatch and docs/backend/fixes-2026-08-31-batch-translate-context.md
	// for why translating each line blind to its neighbors was a problem.
	for lang, items := range groups {
		texts := make([]string, len(items))
		for j, it := range items {
			texts[j] = it.text
		}

		translated, err := s.translator.TranslateBatch(c.Request.Context(), texts, lang, req.TargetLang)
		if err != nil {
			for _, it := range items {
				outcomes[it.idx] = outcome{err: err}
			}
			continue
		}
		for j, it := range items {
			outcomes[it.idx] = outcome{translated: translated[j]}
		}
	}

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

		line.Translation = o.translated
		line.TranslationLang = ""
		if s.translatorIsLLM {
			line.Method = appdb.MethodAI
		} else {
			line.Method = appdb.MethodMT
		}
		// Fresh machine output, not yet seen by a human — flag it the same
		// way a scrape+align guess is (see handleAlignTrack), so it surfaces
		// in EditorPage's needs_review summary/highlighting instead of
		// silently looking "done". Cleared by handleUpdateLine once the user
		// actually edits/confirms the line (see its NeedsReview reset).
		line.NeedsReview = true
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

// translateOneCached checks translation_cache before calling out to the
// active Translator, and writes a cache entry on a fresh call. This is what
// makes repeat translate requests (same text/lang pair, even across
// different tracks) avoid hitting the external API again. The cache key is
// namespaced by s.translatorID so switching providers (e.g. libretranslate
// -> gemini) doesn't return a stale result translated by a different engine.
func (s *Server) translateOneCached(ctx context.Context, text, sourceLang, targetLang string) (translated string, cacheHit bool, err error) {
	key := translationCacheKey(text, sourceLang, targetLang, s.translatorID)

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
		Provider:       s.translatorID,
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
