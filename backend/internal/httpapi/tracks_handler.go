package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	appdb "lrc-translate/backend/internal/db"
	"lrc-translate/backend/internal/lrc"
)

// POST /api/tracks/import { lrclib_id }
// Fetches the track + synced lyrics from LRCLIB, parses the LRC, and
// persists a new Track with its Lines.
func (s *Server) handleImportTrack(c *gin.Context) {
	var req ImportTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remote, err := s.lrclib.GetByID(c.Request.Context(), req.LrclibID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch track from LRCLIB: " + err.Error()})
		return
	}
	if remote.SyncedLyrics == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "this track has no synced (timestamped) lyrics on LRCLIB"})
		return
	}

	parsedLines, err := lrc.Parse(remote.SyncedLyrics)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "failed to parse synced lyrics: " + err.Error()})
		return
	}

	track := appdb.Track{
		ID:           uuid.NewString(),
		LrclibID:     strconv.FormatInt(remote.ID, 10),
		Title:        remote.TrackName,
		Artist:       remote.ArtistName,
		Album:        remote.AlbumName,
		DurationMs:   int64(remote.Duration * 1000),
		Source:       appdb.SourceLRCLIB,
		RawSyncedLrc: remote.SyncedLyrics,
	}
	track.Language = detectLanguage(parsedLines)

	track.Lines = make([]appdb.Line, len(parsedLines))
	for i, pl := range parsedLines {
		track.Lines[i] = appdb.Line{
			TrackID:   track.ID,
			LineIndex: i,
			TimeMs:    pl.TimeMs,
			Timestamp: pl.Timestamp,
			Original:  pl.Text,
			Method:    appdb.MethodNone,
		}
	}

	if err := s.db.Create(&track).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save track: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toTrackDTO(track))
}

// GET /api/tracks
func (s *Server) handleListTracks(c *gin.Context) {
	var tracks []appdb.Track
	if err := s.db.Order("created_at desc").Find(&tracks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]TrackSummaryDTO, len(tracks))
	for i, t := range tracks {
		out[i] = TrackSummaryDTO{
			ID:         t.ID,
			Title:      t.Title,
			Artist:     t.Artist,
			Album:      t.Album,
			DurationMs: t.DurationMs,
			Language:   t.Language,
			Source:     string(t.Source),
		}
	}
	c.JSON(http.StatusOK, out)
}

// GET /api/tracks/:id
func (s *Server) handleGetTrack(c *gin.Context) {
	track, err := s.loadTrack(c.Param("id"))
	if err != nil {
		respondTrackLookupError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTrackDTO(*track))
}

// PUT /api/tracks/:id
func (s *Server) handleUpdateTrack(c *gin.Context) {
	track, err := s.loadTrack(c.Param("id"))
	if err != nil {
		respondTrackLookupError(c, err)
		return
	}

	var req UpdateTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title != nil {
		track.Title = *req.Title
	}
	if req.Artist != nil {
		track.Artist = *req.Artist
	}
	if req.Album != nil {
		track.Album = *req.Album
	}
	if req.Language != nil {
		track.Language = *req.Language
	}

	if err := s.db.Save(track).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toTrackDTO(*track))
}

// DELETE /api/tracks/:id
func (s *Server) handleDeleteTrack(c *gin.Context) {
	track, err := s.loadTrack(c.Param("id"))
	if err != nil {
		respondTrackLookupError(c, err)
		return
	}
	if err := s.db.Select("Lines").Delete(track).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// PUT /api/tracks/:id/lines/:lineId
// Milestone 1 scope: manual edits to the original lyric text and/or its
// timestamp. Translation editing (with suggested_* snapshot + method
// transition to "manual") is added in Milestone 2.
func (s *Server) handleUpdateLine(c *gin.Context) {
	trackID := c.Param("id")

	var line appdb.Line
	if err := s.db.Where("id = ? AND track_id = ?", c.Param("lineId"), trackID).First(&line).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "line not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req UpdateLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Original != nil {
		line.Original = *req.Original
	}
	if req.Romanized != nil {
		line.Romanized = *req.Romanized
		line.RomanizedSource = "manual"
	}
	if req.Timestamp != nil {
		line.Timestamp = *req.Timestamp
	}
	if req.TimeMs != nil {
		line.TimeMs = *req.TimeMs
	}
	if req.Translation != nil {
		// Snapshot the last auto-generated suggestion before overwriting it,
		// so the user can revert. Only snapshot once: if the line is already
		// "manual", the existing Suggested* still points at the original
		// mt/scrape/ai suggestion and must not be clobbered by this edit.
		if line.Method != appdb.MethodManual {
			line.SuggestedTranslation = line.Translation
			line.SuggestedMethod = line.Method
			line.SuggestedTranslationLang = line.TranslationLang
		}
		line.Translation = *req.Translation
		line.Method = appdb.MethodManual
		line.TranslationLang = "" // no longer attributable to a scrape source
		// A human just looked at and fixed this line, which is exactly what
		// needs_review was asking for — see handleTranslateTrack and
		// handleAlignTrack for where it gets set true.
		line.NeedsReview = false
	}

	if err := s.db.Save(&line).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toLineDTO(line))
}

func (s *Server) loadTrack(id string) (*appdb.Track, error) {
	var track appdb.Track
	err := s.db.Preload("Lines", func(db *gorm.DB) *gorm.DB {
		return db.Order("line_index asc")
	}).First(&track, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func respondTrackLookupError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "track not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func toTrackDTO(t appdb.Track) TrackDTO {
	lines := make([]LineDTO, len(t.Lines))
	for i, l := range t.Lines {
		lines[i] = toLineDTO(l)
	}
	return TrackDTO{
		ID:         t.ID,
		LrclibID:   t.LrclibID,
		Title:      t.Title,
		Artist:     t.Artist,
		Album:      t.Album,
		DurationMs: t.DurationMs,
		Language:   t.Language,
		Source:     string(t.Source),
		Lines:      lines,
	}
}

func toLineDTO(l appdb.Line) LineDTO {
	dto := LineDTO{
		ID:              l.ID,
		LineIndex:       l.LineIndex,
		TimeMs:          l.TimeMs,
		Timestamp:       l.Timestamp,
		Original:        l.Original,
		Romanized:       l.Romanized,
		RomanizedSource: l.RomanizedSource,
		Translation:     l.Translation,
		TranslationLang: l.TranslationLang,
		Method:          string(l.Method),
		NeedsReview:     l.NeedsReview,
	}
	if l.ScrapeContext != "" {
		var ctx LineScrapeContextsDTO
		// A decode failure here means stored JSON we can no longer parse
		// (shouldn't happen — this app is the only writer) — surfacing it
		// as a missing context is harmless (same as never having one), so
		// it's ignored rather than failing the whole line/track fetch over
		// what's purely a "nice to have" review aid.
		if err := json.Unmarshal([]byte(l.ScrapeContext), &ctx); err == nil {
			dto.ScrapeContext = &ctx
		}
	}
	return dto
}

// detectLanguage makes a best-effort guess ("ja" or "") by checking whether
// any line contains Hiragana/Katakana/Kanji characters. Good enough to decide
// whether to surface the "Romanize" action in the UI (Milestone 2) — not a
// general-purpose language detector.
func detectLanguage(lines []lrc.Line) string {
	for _, l := range lines {
		for _, r := range l.Text {
			switch {
			case r >= 0x3040 && r <= 0x309F: // Hiragana
				return "ja"
			case r >= 0x30A0 && r <= 0x30FF: // Katakana
				return "ja"
			case r >= 0x4E00 && r <= 0x9FFF: // CJK Unified Ideographs (kanji)
				return "ja"
			}
		}
	}
	return ""
}
