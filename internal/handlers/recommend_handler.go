package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/core/services"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
)

const (
	recommendResolveCap = 10 // Explore "Because" rails — longer browse before radio
	/** Radio wants a longer same-vibe batch so the queue doesn't pivot every extend. */
	radioResolveCap     = 12
	recommendSearchPeek = 8
	recommendTTL        = 6 * time.Hour
	recommendCacheCap   = 64

	// ponytail: region hard-coded until settings grow a country picker
	exploreCountry = "Romania"
	exploreTTL     = 6 * time.Hour
	exploreCap     = 10
	exploreFetch   = 16
)

type recommendEntry struct {
	songs []domain.Song
	at    time.Time
}

type ExploreSection struct {
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Subtitle string        `json:"subtitle"`
	Songs    []domain.Song `json:"songs"`
}

var (
	parenRe = regexp.MustCompile(`\([^)]*\)|\[[^\]]*\]`)
	featRe  = regexp.MustCompile(`(?i)\s*(feat\.?|ft\.?|featuring)\s+.*$`)
	junkRe  = regexp.MustCompile(`(?i)\b(original\s+mix|extended\s+mix|radio\s+edit|club\s+mix|remix|bootleg|edit|mix|version|remaster(ed)?|instrumental|karaoke|live|acoustic|dub)\b`)
	spaceRe = regexp.MustCompile(`\s+`)
)

type RecommendHandler struct {
	client *http.Client
	apiKey string
	search ports.ISearchService
	covers *services.CoverService

	exploreMu       sync.Mutex
	exploreSections []ExploreSection
	exploreAt       time.Time

	recommendMu    sync.Mutex
	recommendCache map[string]recommendEntry
}

func NewRecommendHandler(client *http.Client, apiKey string, search ports.ISearchService, covers *services.CoverService) *RecommendHandler {
	return &RecommendHandler{
		client: client, apiKey: apiKey, search: search, covers: covers,
		recommendCache: make(map[string]recommendEntry),
	}
}

// GET /explore?refresh=1 → chart shelves (Romania / Worldwide / vibes), server-cached 6h.
func (h *RecommendHandler) GetExplore(c fiber.Ctx) error {
	if h.apiKey == "" {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "LASTFM_API_KEY not set"})
	}
	refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")
	if !refresh {
		if sections, ok := h.exploreSnap(true); ok {
			c.Set("Cache-Control", "public, max-age=21600")
			return c.JSON(fiber.Map{"country": exploreCountry, "sections": sections, "cached": true})
		}
	}

	sections, err := h.buildExplore(c.Context())
	if err != nil || len(sections) == 0 {
		// Serve last good payload even past TTL — better than empty Explore.
		if stale, ok := h.exploreSnap(false); ok {
			c.Set("Cache-Control", "public, max-age=60")
			return c.JSON(fiber.Map{"country": exploreCountry, "sections": stale, "cached": true})
		}
		if err != nil {
			return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": "Couldn't load charts"})
		}
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "No playable charts right now"})
	}
	// ponytail: don't poison the 6h cache with a half-empty build
	if len(sections) >= 2 {
		h.exploreStore(sections)
	}
	c.Set("Cache-Control", "public, max-age=21600")
	return c.JSON(fiber.Map{"country": exploreCountry, "sections": sections, "cached": false})
}

type exploreJob struct {
	id, title, subtitle string
	q                   url.Values
}

func (h *RecommendHandler) buildExplore(ctx context.Context) ([]ExploreSection, error) {
	limit := strconv.Itoa(exploreFetch)
	jobs := []exploreJob{
		{"romania", "Romania", "Hot this week", url.Values{
			"method": {"geo.getTopTracks"}, "country": {exploreCountry}, "limit": {limit},
		}},
		// ponytail: no free TikTok RO API — YouTube Music trending RO is the closest viral proxy
		{"viral-ro", "Viral Romania", "Hot trends right now", nil},
		{"worldwide", "Worldwide", "Global chart", url.Values{
			"method": {"chart.getTopTracks"}, "limit": {limit},
		}},
		{"pop", "Pop", "Trending vibe", url.Values{
			"method": {"tag.getTopTracks"}, "tag": {"pop"}, "limit": {limit},
		}},
		{"electronic", "Electronic", "Trending vibe", url.Values{
			"method": {"tag.getTopTracks"}, "tag": {"electronic"}, "limit": {limit},
		}},
	}

	// ponytail: one shelf at a time — parallel resolve floods providers and returns 1 song.
	out := make([]ExploreSection, 0, len(jobs))
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		var pairs []lastfmPair
		var err error
		if job.id == "viral-ro" {
			pairs, err = h.kworbROViral(ctx)
		} else {
			pairs, err = h.lastfmTracks(ctx, job.q, "tracks")
		}
		if err != nil || len(pairs) == 0 {
			continue
		}
		songs := h.resolveN(ctx, pairs, lastfmPair{}, exploreCap)
		if len(songs) == 0 {
			continue
		}
		h.covers.FillSongs(ctx, songs)
		out = append(out, ExploreSection{
			ID: job.id, Title: job.title, Subtitle: job.subtitle, Songs: songs,
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// YouTube Music trending Romania via kworb (public HTML). Closest free stand-in for TikTok RO.
const kworbROViralURL = "https://kworb.net/youtube/trending/ro.html"

func (h *RecommendHandler) kworbROViral(ctx context.Context) ([]lastfmPair, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kworbROViralURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "FindVibe/1.0")
	req.Header.Set("Accept", "text/html")
	res, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kworb viral: HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return parseKworbMusicPairs(string(body)), nil
}

var kworbAnchorRe = regexp.MustCompile(`(?i)<a[^>]*>([^<]+)</a>`)

// parseKworbMusicPairs reads the music-only table (not overall trending with gaming/vlogs).
func parseKworbMusicPairs(html string) []lastfmPair {
	start := strings.Index(html, `class="music"`)
	if start < 0 {
		return nil
	}
	chunk := html[start:]
	if i := strings.Index(chunk, "</table>"); i >= 0 {
		chunk = chunk[:i]
	}
	seen := map[string]bool{}
	out := make([]lastfmPair, 0, exploreFetch)
	for _, m := range kworbAnchorRe.FindAllStringSubmatch(chunk, -1) {
		if len(out) >= exploreFetch {
			break
		}
		artist, title, ok := parseChartTitle(m[1])
		if !ok {
			continue
		}
		k := songKey(artist, title)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, lastfmPair{artist: artist, title: title})
	}
	return out
}

// "VANILLA x ALEX VELEA - 7 din 7 | Official Video" → ("VANILLA", "7 din 7")
func parseChartTitle(raw string) (artist, title string, ok bool) {
	raw = strings.TrimSpace(htmlUnescape(raw))
	if raw == "" {
		return "", "", false
	}
	if i := strings.Index(raw, "|"); i >= 0 {
		raw = strings.TrimSpace(raw[:i])
	}
	sep := " - "
	i := strings.Index(raw, sep)
	if i < 0 {
		sep = " – "
		i = strings.Index(raw, sep)
	}
	if i <= 0 {
		return "", "", false
	}
	artist = chartPrimaryArtist(strings.TrimSpace(raw[:i]))
	title = strings.TrimSpace(raw[i+len(sep):])
	title = parenRe.ReplaceAllString(title, " ")
	title = spaceRe.ReplaceAllString(strings.TrimSpace(title), " ")
	if artist == "" || title == "" {
		return "", "", false
	}
	return artist, title, true
}

func chartPrimaryArtist(a string) string {
	for _, sep := range []string{" x ", " ❌", " ✘", " × ", " & ", " feat.", " ft.", " featuring "} {
		if i := strings.Index(strings.ToLower(a), strings.ToLower(sep)); i > 0 {
			return strings.TrimSpace(a[:i])
		}
	}
	return strings.TrimSpace(a)
}

func htmlUnescape(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return s
}

func (h *RecommendHandler) exploreSnap(freshOnly bool) ([]ExploreSection, bool) {
	h.exploreMu.Lock()
	defer h.exploreMu.Unlock()
	if len(h.exploreSections) == 0 {
		return nil, false
	}
	if freshOnly && time.Since(h.exploreAt) > exploreTTL {
		return nil, false
	}
	out := make([]ExploreSection, len(h.exploreSections))
	copy(out, h.exploreSections)
	return out, true
}

func (h *RecommendHandler) exploreStore(sections []ExploreSection) {
	h.exploreMu.Lock()
	defer h.exploreMu.Unlock()
	h.exploreSections = append([]ExploreSection(nil), sections...)
	h.exploreAt = time.Now()
}

func (h *RecommendHandler) recommendSnap(key string) ([]domain.Song, bool) {
	if key == "" {
		return nil, false
	}
	h.recommendMu.Lock()
	defer h.recommendMu.Unlock()
	e, ok := h.recommendCache[key]
	if !ok || len(e.songs) == 0 || time.Since(e.at) > recommendTTL {
		return nil, false
	}
	out := make([]domain.Song, len(e.songs))
	copy(out, e.songs)
	return out, true
}

func (h *RecommendHandler) recommendStale(key string) ([]domain.Song, bool) {
	if key == "" {
		return nil, false
	}
	h.recommendMu.Lock()
	defer h.recommendMu.Unlock()
	e, ok := h.recommendCache[key]
	if !ok || len(e.songs) == 0 {
		return nil, false
	}
	out := make([]domain.Song, len(e.songs))
	copy(out, e.songs)
	return out, true
}

func (h *RecommendHandler) recommendStore(key string, songs []domain.Song) {
	if key == "" || len(songs) == 0 {
		return
	}
	h.recommendMu.Lock()
	defer h.recommendMu.Unlock()
	if h.recommendCache == nil {
		h.recommendCache = make(map[string]recommendEntry)
	}
	// ponytail: bounded map — drop expired, else wipe when full
	if len(h.recommendCache) >= recommendCacheCap {
		for k, e := range h.recommendCache {
			if time.Since(e.at) > recommendTTL {
				delete(h.recommendCache, k)
			}
		}
		if len(h.recommendCache) >= recommendCacheCap {
			h.recommendCache = make(map[string]recommendEntry, recommendCacheCap/2)
		}
	}
	cp := make([]domain.Song, len(songs))
	copy(cp, songs)
	h.recommendCache[key] = recommendEntry{songs: cp, at: time.Now()}
}

// GET /similar-artists?artist= → {artists: string[]} from Last.fm artist.getSimilar.
func (h *RecommendHandler) GetSimilarArtists(c fiber.Ctx) error {
	if h.apiKey == "" {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "LASTFM_API_KEY not set"})
	}
	artist := strings.TrimSpace(c.Query("artist"))
	if artist == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "artist required"})
	}
	names, err := h.lastfmSimilarArtists(c.Context(), artist)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": "Couldn't load similar artists"})
	}
	if len(names) > 12 {
		names = names[:12]
	}
	return c.JSON(fiber.Map{"artists": names})
}

// GET /resolve?artist=&title=&strict=1 → one playable Song (search + fuzzy match).
// strict=1: require artist+title overlap (Spotify import) — no "first hit" fallback.
func (h *RecommendHandler) GetResolve(c fiber.Ctx) error {
	artist := strings.TrimSpace(c.Query("artist"))
	title := strings.TrimSpace(c.Query("title"))
	if artist == "" || title == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "artist and title required"})
	}
	strict := c.Query("strict") == "1" || strings.EqualFold(c.Query("strict"), "true")
	song, ok := h.resolveOne(c.Context(), lastfmPair{artist: artist, title: title}, lastfmPair{}, strict)
	if !ok {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "no match"})
	}
	// Spotify import + play-from-resolve — fill art before the client saves to vault.
	songs := []domain.Song{song}
	h.covers.FillSongs(c.Context(), songs)
	return c.JSON(songs[0])
}

// GET /recommend?artist=&title=&mode=radio&offset=N
// Cached 6h per normalized seed (+ mode/offset). Explore "because" stays diverse;
// mode=radio keeps same-artist + similar so the station doesn't pivot genres.
func (h *RecommendHandler) GetRecommend(c fiber.Ctx) error {
	artist := strings.TrimSpace(c.Query("artist"))
	title := strings.TrimSpace(c.Query("title"))
	if artist == "" || title == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "artist and title are required"})
	}
	if h.apiKey == "" {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "LASTFM_API_KEY not set"})
	}

	radio := strings.EqualFold(c.Query("mode"), "radio")
	offset, _ := strconv.Atoi(c.Query("offset"))
	if offset < 0 {
		offset = 0
	}
	capN := recommendResolveCap
	if radio {
		capN = radioResolveCap
	}

	refresh := c.Query("refresh") == "1" || strings.EqualFold(c.Query("refresh"), "true")
	key := songKey(artist, title)
	if radio {
		key = key + "|radio|" + strconv.Itoa(offset)
	}
	if !refresh {
		if songs, ok := h.recommendSnap(key); ok {
			c.Set("Cache-Control", "private, max-age=21600")
			return c.JSON(songs)
		}
	}

	seed := lastfmPair{artist: artist, title: title}
	artists := artistCandidates(artist)

	pairs, err := h.collectPairs(c.Context(), seed, artists, radio)
	if err != nil {
		if stale, ok := h.recommendStale(key); ok {
			c.Set("Cache-Control", "private, max-age=60")
			return c.JSON(stale)
		}
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": "Radio lookup failed. Try again."})
	}
	if radio && offset > 0 && len(pairs) > 0 {
		off := offset % len(pairs)
		pairs = append(pairs[off:], pairs[:off]...)
	}

	songs := h.resolveN(c.Context(), pairs, seed, capN)
	// ponytail: Last.fm often misses collab credits — fall back to our search by artist name.
	if len(songs) == 0 {
		songs = h.searchFallback(c.Context(), artists, seed)
		if radio && len(songs) > capN {
			songs = songs[:capN]
		}
	}
	if len(songs) == 0 {
		if stale, ok := h.recommendStale(key); ok {
			c.Set("Cache-Control", "private, max-age=60")
			return c.JSON(stale)
		}
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Couldn't build a radio for this track"})
	}
	h.covers.FillSongs(c.Context(), songs)
	h.recommendStore(key, songs)
	c.Set("Cache-Control", "private, max-age=21600")
	return c.JSON(songs)
}

type lastfmPair struct{ artist, title string }

func (h *RecommendHandler) collectPairs(ctx context.Context, seed lastfmPair, artists []string, radio bool) ([]lastfmPair, error) {
	pairCap := recommendResolveCap * 2
	if radio {
		pairCap = radioResolveCap * 3 // more candidates so offset rotates through a neighborhood
	}
	var pairs []lastfmPair

	for _, a := range artists {
		similar, err := h.lastfmSimilar(ctx, a, seed.title)
		if err != nil {
			return nil, err
		}
		pairs = uniquePairs(append(pairs, similar...), seed)
		if len(pairs) >= pairCap {
			break
		}
	}

	for _, a := range artists {
		if len(pairs) >= pairCap {
			break
		}
		extra, err := h.pairsFromSimilarArtists(ctx, a, seed.title)
		if err != nil {
			return nil, err
		}
		pairs = uniquePairs(append(pairs, extra...), seed)
	}

	for _, a := range artists {
		if len(pairs) >= pairCap {
			break
		}
		tops, err := h.lastfmArtistTop(ctx, a, seed.title)
		if err != nil {
			return nil, err
		}
		pairs = uniquePairs(append(pairs, tops...), seed)
	}

	// Explore "because": prefer other artists. Radio: keep same-artist tops in the mix.
	if !radio {
		others := filterNotSeedArtists(pairs, artists)
		if len(others) >= 3 {
			return others, nil
		}
	}
	return pairs, nil
}

// artistCandidates: "Satoshi, magnat, feoctist" → Satoshi first (Last.fm / search), then others.
func artistCandidates(artist string) []string {
	full := strings.TrimSpace(artist)
	if full == "" {
		return nil
	}
	parts := splitArtistParts(full)
	out := make([]string, 0, len(parts)+1)
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		k := utils.NormalizeString(s)
		if seen[k] {
			return
		}
		seen[k] = true
		out = append(out, s)
	}
	for _, p := range parts {
		add(p)
	}
	add(full)
	if len(out) > 4 {
		out = out[:4]
	}
	return out
}

func splitArtistParts(artist string) []string {
	s := artist
	for _, sep := range []string{" feat. ", " feat ", " ft. ", " ft ", " featuring ", " x ", " X ", " / ", ";", "&"} {
		s = strings.ReplaceAll(s, sep, ",")
	}
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func filterNotSeedArtists(pairs []lastfmPair, seedArtists []string) []lastfmPair {
	block := map[string]bool{}
	for _, a := range seedArtists {
		block[utils.NormalizeString(a)] = true
	}
	out := make([]lastfmPair, 0, len(pairs))
	for _, p := range pairs {
		if block[utils.NormalizeString(p.artist)] {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (h *RecommendHandler) searchFallback(ctx context.Context, artists []string, seed lastfmPair) []domain.Song {
	seedKey := songKey(seed.artist, seed.title)
	seen := map[string]bool{seedKey: true}
	out := make([]domain.Song, 0, recommendResolveCap)

	for _, a := range artists {
		if len(out) >= recommendResolveCap {
			break
		}
		resp, err := h.search.Search(ctx, a, 1)
		if err != nil || resp == nil {
			continue
		}
		for _, s := range resp.Songs {
			k := songKey(s.Artist, s.Title)
			if k == "" || seen[k] || s.Link == "" {
				continue
			}
			seen[k] = true
			out = append(out, s)
			if len(out) >= recommendResolveCap {
				break
			}
		}
	}
	return out
}

func (h *RecommendHandler) lastfmSimilar(ctx context.Context, artist, title string) ([]lastfmPair, error) {
	return h.lastfmTracks(ctx, url.Values{
		"method": {"track.getSimilar"},
		"artist": {artist},
		"track":  {title},
		"limit":  {"25"},
	}, "similartracks")
}

func (h *RecommendHandler) lastfmArtistTop(ctx context.Context, artist, skipTitle string) ([]lastfmPair, error) {
	pairs, err := h.lastfmTracks(ctx, url.Values{
		"method": {"artist.getTopTracks"},
		"artist": {artist},
		"limit":  {"12"},
	}, "toptracks")
	if err != nil {
		return nil, err
	}
	out := make([]lastfmPair, 0, len(pairs))
	skipCore := coreTitle(skipTitle)
	for _, p := range pairs {
		if coreTitle(p.title) == skipCore {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// pairsFromSimilarArtists: artist.getSimilar → one top track each (different artists by design).
func (h *RecommendHandler) pairsFromSimilarArtists(ctx context.Context, artist, skipTitle string) ([]lastfmPair, error) {
	names, err := h.lastfmSimilarArtists(ctx, artist)
	if err != nil || len(names) == 0 {
		return nil, err
	}
	if len(names) > recommendResolveCap+2 {
		names = names[:recommendResolveCap+2]
	}

	type slot struct {
		i    int
		pair lastfmPair
		ok   bool
	}
	ch := make(chan slot, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			tops, err := h.lastfmArtistTop(ctx, name, skipTitle)
			if err != nil || len(tops) == 0 {
				ch <- slot{i: i}
				return
			}
			ch <- slot{i: i, pair: tops[0], ok: true}
		}(i, name)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	byIdx := make(map[int]lastfmPair, len(names))
	for s := range ch {
		if s.ok {
			byIdx[s.i] = s.pair
		}
	}
	out := make([]lastfmPair, 0, len(byIdx))
	for i := 0; i < len(names); i++ {
		if p, ok := byIdx[i]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (h *RecommendHandler) lastfmSimilarArtists(ctx context.Context, artist string) ([]string, error) {
	q := url.Values{
		"method":  {"artist.getSimilar"},
		"artist":  {artist},
		"api_key": {h.apiKey},
		"format":  {"json"},
		"limit":   {"10"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ws.audioscrobbler.com/2.0/?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var data struct {
		SimilarArtists struct {
			Artist json.RawMessage `json:"artist"`
		} `json:"similarartists"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	artists, err := decodeLastfmArtistList(data.SimilarArtists.Artist)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(artists))
	seed := utils.NormalizeString(artist)
	for _, a := range artists {
		n := strings.TrimSpace(a.Name)
		if n == "" || utils.NormalizeString(n) == seed {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

func (h *RecommendHandler) lastfmTracks(ctx context.Context, q url.Values, root string) ([]lastfmPair, error) {
	q.Set("api_key", h.apiKey)
	q.Set("format", "json")
	u := "https://ws.audioscrobbler.com/2.0/?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	block, ok := raw[root]
	if !ok {
		return nil, nil
	}
	var wrap struct {
		Track json.RawMessage `json:"track"`
	}
	if err := json.Unmarshal(block, &wrap); err != nil || len(wrap.Track) == 0 {
		return nil, nil
	}

	tracks, err := decodeLastfmTrackList(wrap.Track)
	if err != nil {
		return nil, err
	}
	out := make([]lastfmPair, 0, len(tracks))
	for _, t := range tracks {
		a, n := strings.TrimSpace(t.Artist.Name), strings.TrimSpace(t.Name)
		if a == "" || n == "" {
			continue
		}
		out = append(out, lastfmPair{a, n})
	}
	return out, nil
}

type lastfmTrack struct {
	Name   string `json:"name"`
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
}

type lastfmArtist struct {
	Name string `json:"name"`
}

func decodeLastfmTrackList(raw json.RawMessage) ([]lastfmTrack, error) {
	var many []lastfmTrack
	if err := json.Unmarshal(raw, &many); err == nil {
		return many, nil
	}
	var one lastfmTrack
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, err
	}
	if one.Name == "" {
		return nil, nil
	}
	return []lastfmTrack{one}, nil
}

func decodeLastfmArtistList(raw json.RawMessage) ([]lastfmArtist, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var many []lastfmArtist
	if err := json.Unmarshal(raw, &many); err == nil {
		return many, nil
	}
	var one lastfmArtist
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, err
	}
	if one.Name == "" {
		return nil, nil
	}
	return []lastfmArtist{one}, nil
}

func uniquePairs(pairs []lastfmPair, seed lastfmPair) []lastfmPair {
	seen := map[string]bool{songKey(seed.artist, seed.title): true}
	out := make([]lastfmPair, 0, len(pairs))
	for _, p := range pairs {
		k := songKey(p.artist, p.title)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, p)
	}
	return out
}

// songKey collapses remix/feat variants: "Collide (Extended Mix)" ≈ "Collide".
func songKey(artist, title string) string {
	a, t := utils.NormalizeString(artist), coreTitle(title)
	if a == "" || t == "" {
		return ""
	}
	return a + "|" + t
}

func coreTitle(title string) string {
	s := utils.NormalizeString(title)
	s = parenRe.ReplaceAllString(s, " ")
	s = featRe.ReplaceAllString(s, " ")
	s = junkRe.ReplaceAllString(s, " ")
	s = spaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
	return s
}

func (h *RecommendHandler) resolve(ctx context.Context, pairs []lastfmPair, seed lastfmPair) []domain.Song {
	return h.resolveN(ctx, pairs, seed, recommendResolveCap)
}

func (h *RecommendHandler) resolveN(ctx context.Context, pairs []lastfmPair, seed lastfmPair, cap int) []domain.Song {
	if cap < 1 {
		cap = recommendResolveCap
	}
	if len(pairs) > cap*2 {
		pairs = pairs[:cap*2]
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type slot struct {
		i    int
		song domain.Song
		ok   bool
	}
	ch := make(chan slot, len(pairs))
	var wg sync.WaitGroup
	for i, p := range pairs {
		wg.Add(1)
		go func(i int, p lastfmPair) {
			defer wg.Done()
			song, ok := h.resolveOne(ctx, p, seed, false)
			ch <- slot{i: i, song: song, ok: ok}
		}(i, p)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	byIdx := make(map[int]domain.Song, len(pairs))
	decided := make([]bool, len(pairs))
	for s := range ch {
		decided[s.i] = true
		if s.ok {
			byIdx[s.i] = s.song
		}
		// ponytail: only cut once a prefix is fully decided — keeps Last.fm order.
		prefix := 0
		for prefix < len(pairs) && decided[prefix] {
			prefix++
		}
		if out := pickResolved(byIdx, prefix, seed, cap); len(out) >= cap {
			cancel()
			return out
		}
	}
	return pickResolved(byIdx, len(pairs), seed, cap)
}

func pickResolved(byIdx map[int]domain.Song, n int, seed lastfmPair, cap int) []domain.Song {
	seen := map[string]bool{songKey(seed.artist, seed.title): true}
	seenArtists := map[string]bool{}
	out := make([]domain.Song, 0, cap)
	for i := 0; i < n && len(out) < cap; i++ {
		song, ok := byIdx[i]
		if !ok {
			continue
		}
		k := songKey(song.Artist, song.Title)
		if k == "" || seen[k] {
			continue
		}
		art := utils.NormalizeString(song.Artist)
		// Soft diversity: skip second track from same artist while we still have room to fill later.
		if seenArtists[art] && len(out) < cap-1 && i < n-1 {
			continue
		}
		seen[k] = true
		seenArtists[art] = true
		out = append(out, song)
	}
	return out
}

func (h *RecommendHandler) resolveOne(ctx context.Context, want, seed lastfmPair, strict bool) (domain.Song, bool) {
	resp, err := h.search.Search(ctx, want.artist+" "+want.title, 1)
	if err != nil || resp == nil || len(resp.Songs) == 0 {
		return domain.Song{}, false
	}

	wantCore := coreTitle(want.title)
	wantArt := utils.NormalizeString(want.artist)
	seedKey := songKey(seed.artist, seed.title)
	n := recommendSearchPeek
	if n > len(resp.Songs) {
		n = len(resp.Songs)
	}

	var fallback domain.Song
	for i := 0; i < n; i++ {
		s := resp.Songs[i]
		if songKey(s.Artist, s.Title) == seedKey {
			continue
		}
		gotCore := coreTitle(s.Title)
		art := utils.NormalizeString(s.Artist)
		artistOK := art == wantArt || strings.Contains(art, wantArt) || strings.Contains(wantArt, art)
		titleOK := gotCore == wantCore || strings.Contains(gotCore, wantCore) || strings.Contains(wantCore, gotCore)
		if artistOK && titleOK && !isRemixy(s.Title) {
			return s, true
		}
		if artistOK && titleOK && fallback.Link == "" {
			fallback = s
		}
		// Loose path only: artist match alone can fill fallback (radio/explore).
		if !strict && artistOK && fallback.Link == "" {
			fallback = s
		}
	}
	if fallback.Link != "" {
		return fallback, true
	}
	if strict {
		return domain.Song{}, false
	}
	// Last: first result that isn't the seed song.
	for _, s := range resp.Songs[:n] {
		if songKey(s.Artist, s.Title) != seedKey {
			return s, true
		}
	}
	return domain.Song{}, false
}

func isRemixy(title string) bool {
	t := utils.NormalizeString(title)
	return junkRe.MatchString(t) || parenRe.MatchString(t)
}
