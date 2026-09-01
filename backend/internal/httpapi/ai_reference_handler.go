package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// aiReferenceConcurrency is deliberately lower than maxConcurrentTranslations
// (handleTranslateTrack's limit, tuned for self-hosted LibreTranslate's CPU
// headroom — see its doc comment). This endpoint is typically called once
// for an entire song's worth of needs_review lines in one go, with none of
// them cached yet, against Gemini's free-tier per-minute rate limit — a
// bottleneck concurrency makes *worse*, not better, since a bigger burst of
// simultaneous requests just exhausts that quota faster and triggers more
// 429s than translating serially would (each 429 costs a real multi-second
// backoff — see internal/gemini's exponentialBackoff — so a handful of
// well-spaced requests reliably beats a burst that mostly fails and retries
// into its own still-exhausted window).
const aiReferenceConcurrency = 1

// POST /api/tracks/:id/ai-reference { target_lang, line_ids? }
//
// Fills in LineDTO.ScrapeContext.AI for the targeted lines (default: every
// line with a non-empty Original) with a fresh machine translation of that
// line's Original text — see AIReferenceRequest/LineScrapeContextsDTO.AI's
// doc comments for why this exists: a scrape-aligned Translation is only a
// heuristic guess at position (see internal/align), and a reviewer who
// can't read the source language has no way to catch a wrong guess that
// isn't an exact duplicate. This gives them a same-language (their target
// language) reference to compare against instead, without touching
// Translation/Method/NeedsReview at all — purely additive, using
// translateOneCached's per-line caching (handleTranslateTrack no longer
// does — see TranslateResponse's doc comment) so re-running this (e.g.
// after fixing a few lines by hand) doesn't re-spend API calls on lines
// whose Original text hasn't changed.
func (s *Server) handleGetAIReference(c *gin.Context) {
	track, err := s.loadTrack(c.Param("id"))
	if err != nil {
		respondTrackLookupError(c, err)
		return
	}

	var req AIReferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	sem := make(chan struct{}, aiReferenceConcurrency)
	var wg sync.WaitGroup
	for i, line := range targets {
		if line.Original == "" {
			continue
		}
		if trackLang != "auto" && trackLang == req.TargetLang {
			outcomes[i] = outcome{err: fmt.Errorf("baris ini sudah dalam bahasa target (%s)", trackLang)}
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(i int, text string) {
			defer wg.Done()
			defer func() { <-sem }()
			translated, cacheHit, err := s.translateOneCached(c.Request.Context(), text, trackLang, req.TargetLang)
			outcomes[i] = outcome{translated: translated, cacheHit: cacheHit, err: err}
		}(i, line.Original)
	}
	wg.Wait()

	resp := AIReferenceResponse{Lines: []LineDTO{}}
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

		// Merge into whatever scrape context already exists (Translation/
		// Romanized from a prior align) rather than replacing it wholesale —
		// this endpoint only ever adds/refreshes the AI field.
		var ctx LineScrapeContextsDTO
		if line.ScrapeContext != "" {
			_ = json.Unmarshal([]byte(line.ScrapeContext), &ctx)
		}
		ctx.AI = o.translated
		if encoded, err := json.Marshal(ctx); err == nil {
			line.ScrapeContext = string(encoded)
		}

		if err := s.db.Save(&line).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save AI reference: " + err.Error()})
			return
		}
		resp.Lines = append(resp.Lines, toLineDTO(line))
	}

	c.JSON(http.StatusOK, resp)
}
