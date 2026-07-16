package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
)

const (
	recommendResolveCap = 5
	recommendSearchPeek = 8
)

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
}

func NewRecommendHandler(client *http.Client, apiKey string, search ports.ISearchService) *RecommendHandler {
	return &RecommendHandler{client: client, apiKey: apiKey, search: search}
}

// GET /recommend?artist=&title= → Song[] (Last.fm similar tracks / artists → unique playable hits).
func (h *RecommendHandler) GetRecommend(c fiber.Ctx) error {
	artist := strings.TrimSpace(c.Query("artist"))
	title := strings.TrimSpace(c.Query("title"))
	if artist == "" || title == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "artist and title are required"})
	}
	if h.apiKey == "" {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "LASTFM_API_KEY not set"})
	}

	seed := lastfmPair{artist: artist, title: title}
	artists := artistCandidates(artist)

	pairs, err := h.collectPairs(c.Context(), seed, artists)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": "Radio lookup failed. Try again."})
	}

	songs := h.resolve(c.Context(), pairs, seed)
	// ponytail: Last.fm often misses collab credits — fall back to our search by artist name.
	if len(songs) == 0 {
		songs = h.searchFallback(c.Context(), artists, seed)
	}
	if len(songs) == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Couldn't build a radio for this track"})
	}
	return c.JSON(songs)
}

type lastfmPair struct{ artist, title string }

func (h *RecommendHandler) collectPairs(ctx context.Context, seed lastfmPair, artists []string) ([]lastfmPair, error) {
	var pairs []lastfmPair

	for _, a := range artists {
		similar, err := h.lastfmSimilar(ctx, a, seed.title)
		if err != nil {
			return nil, err
		}
		pairs = uniquePairs(append(pairs, similar...), seed)
		if len(pairs) >= recommendResolveCap {
			break
		}
	}

	for _, a := range artists {
		if len(pairs) >= recommendResolveCap {
			break
		}
		extra, err := h.pairsFromSimilarArtists(ctx, a, seed.title)
		if err != nil {
			return nil, err
		}
		pairs = uniquePairs(append(pairs, extra...), seed)
	}

	for _, a := range artists {
		if len(pairs) >= recommendResolveCap {
			break
		}
		tops, err := h.lastfmArtistTop(ctx, a, seed.title)
		if err != nil {
			return nil, err
		}
		pairs = uniquePairs(append(pairs, tops...), seed)
	}

	others := filterNotSeedArtists(pairs, artists)
	if len(others) >= 3 {
		return others, nil
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
	if len(pairs) > recommendResolveCap*2 {
		pairs = pairs[:recommendResolveCap*2]
	}

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
			song, ok := h.resolveOne(ctx, p, seed)
			ch <- slot{i: i, song: song, ok: ok}
		}(i, p)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	byIdx := make(map[int]domain.Song, len(pairs))
	for s := range ch {
		if s.ok {
			byIdx[s.i] = s.song
		}
	}

	seen := map[string]bool{songKey(seed.artist, seed.title): true}
	seenArtists := map[string]bool{}
	out := make([]domain.Song, 0, recommendResolveCap)
	for i := 0; i < len(pairs) && len(out) < recommendResolveCap; i++ {
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
		if seenArtists[art] && len(out) < recommendResolveCap-1 && i < len(pairs)-1 {
			continue
		}
		seen[k] = true
		seenArtists[art] = true
		out = append(out, song)
	}
	return out
}

func (h *RecommendHandler) resolveOne(ctx context.Context, want, seed lastfmPair) (domain.Song, bool) {
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
		if artistOK && fallback.Link == "" {
			fallback = s
		}
	}
	if fallback.Link != "" {
		return fallback, true
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
