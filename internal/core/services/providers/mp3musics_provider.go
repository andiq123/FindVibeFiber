package providers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

const (
	mp3musicsOrigin    = "https://mp3musics.pro"
	mp3musicsStreamAPI = "https://ytsapi.cc/hsl.php"
)

var mp3musicsPace = newPacer(150 * time.Millisecond)

// Mp3musicsProvider scrapes https://mp3musics.pro (YouTube→MP3 front).
// Search returns durable /file/{hex} links (hex = youtubeId|checksum).
// Clients (or ResolveFile) turn those into a playable stream via ytsapi.cc — same as their web player.
type Mp3musicsProvider struct{ *BaseProvider }

func NewMp3musicsProvider(client *http.Client) *Mp3musicsProvider {
	return &Mp3musicsProvider{BaseProvider: NewBaseProvider("Mp3musics", 5, client)}
}

func (p *Mp3musicsProvider) UseRotator(r proxyRotator) *Mp3musicsProvider {
	p.WithRotator(r)
	return p
}

func (p *Mp3musicsProvider) SearchWithPage(ctx context.Context, query string, page int) ([]domain.ProviderResult, error) {
	if page > 1 {
		return nil, nil
	}
	if page < 1 {
		page = 1
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if err := mp3musicsPace.wait(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", p.Name(), err)
	}

	searchURL := mp3musicsOrigin + "/search.php?" + url.Values{"q": {query}}.Encode()
	doc, err := p.fetchDocument(ctx, searchURL, mp3musicsOrigin+"/")
	if err != nil {
		return nil, err
	}

	hits := parseMp3musicsHits(doc)
	results := make([]domain.ProviderResult, 0, len(hits))
	for i, h := range hits {
		song := domain.NewSong(h.title, h.artist, mp3musicsThumb(h.ytID), h.fileURL)
		results = append(results, domain.NewProviderResult(*song, p.Name(), i+1, nil))
	}
	return results, nil
}

// ResolveFile turns /file/{hex} (or bare hex) into a short-lived HTTPS audio URL.
func (p *Mp3musicsProvider) ResolveFile(ctx context.Context, fileRef string) (string, error) {
	ytID, ok := youtubeIDFromMp3musicsFile(fileRef)
	if !ok {
		// bare hex
		ytID, ok = youtubeIDFromMp3musicsFile("/file/" + strings.TrimPrefix(strings.TrimSpace(fileRef), "/file/"))
	}
	if !ok {
		return "", fmt.Errorf("%s: invalid file id", p.Name())
	}

	apiHash, err := p.fetchAPIHash(ctx)
	if err != nil {
		return "", err
	}
	return p.resolveStream(ctx, apiHash, ytID)
}

func (p *Mp3musicsProvider) fetchAPIHash(ctx context.Context) (string, error) {
	doc, err := p.fetchDocument(ctx, mp3musicsOrigin+"/search.php?q=a", mp3musicsOrigin+"/")
	if err != nil {
		return "", err
	}
	hash := strings.TrimSpace(doc.Find("body").AttrOr("data-api-hash", ""))
	if hash == "" {
		return "", fmt.Errorf("%s: missing api hash", p.Name())
	}
	return hash, nil
}

type mp3musicsHit struct {
	artist  string
	title   string
	ytID    string
	fileURL string
}

func parseMp3musicsHits(doc *goquery.Document) []mp3musicsHit {
	out := make([]mp3musicsHit, 0, 20)
	doc.Find("div.tc").Each(func(_ int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find("a.tc-title").AttrOr("title", ""))
		if label == "" {
			label = text(s.Find("a.tc-title").First())
		}
		artist, title := splitMp3musicsLabel(label)
		if artist == "" || title == "" {
			return
		}
		href := strings.TrimSpace(s.Find("a.tc-dl").AttrOr("href", ""))
		ytID, ok := youtubeIDFromMp3musicsFile(href)
		if !ok {
			return
		}
		path := href
		if u, err := url.Parse(href); err == nil && u.Path != "" {
			path = u.Path
		}
		if !strings.HasPrefix(path, "/file/") {
			return
		}
		out = append(out, mp3musicsHit{
			artist:  artist,
			title:   title,
			ytID:    ytID,
			fileURL: mp3musicsOrigin + path,
		})
	})
	return out
}

func splitMp3musicsLabel(label string) (artist, title string) {
	label = strings.TrimSpace(label)
	if label == "" {
		return "", ""
	}
	if i := strings.Index(label, " - "); i > 0 {
		artist = strings.TrimSpace(label[:i])
		title = strings.TrimSpace(label[i+3:])
	} else if i := strings.Index(label, " – "); i > 0 {
		artist = strings.TrimSpace(label[:i])
		title = strings.TrimSpace(label[i+len(" – "):])
	} else {
		return "", label
	}
	if j := strings.Index(title, " | "); j > 0 {
		title = strings.TrimSpace(title[:j])
	}
	return artist, title
}

// /file/{hex} → hex("youtubeId|checksum")
func youtubeIDFromMp3musicsFile(href string) (string, bool) {
	href = strings.TrimSpace(href)
	if href == "" {
		return "", false
	}
	if u, err := url.Parse(href); err == nil && u.Path != "" {
		href = u.Path
	}
	const prefix = "/file/"
	if !strings.HasPrefix(href, prefix) {
		return "", false
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(href, prefix))
	if err != nil || len(raw) == 0 {
		return "", false
	}
	parts := strings.SplitN(string(raw), "|", 2)
	id := strings.TrimSpace(parts[0])
	if len(id) < 6 || len(id) > 20 {
		return "", false
	}
	return id, true
}

func mp3musicsThumb(ytID string) string {
	ytID = strings.TrimSpace(ytID)
	if ytID == "" {
		return ""
	}
	return "https://i.ytimg.com/vi/" + ytID + "/hqdefault.jpg"
}

type mp3musicsStreamResp struct {
	Link string `json:"link"`
}

func (p *Mp3musicsProvider) resolveStream(ctx context.Context, apiHash, ytID string) (string, error) {
	ytID = strings.TrimSpace(ytID)
	apiHash = strings.TrimSpace(apiHash)
	if ytID == "" || apiHash == "" {
		return "", fmt.Errorf("%s: missing stream credentials", p.Name())
	}

	u := mp3musicsStreamAPI + "?" + url.Values{
		"token": {apiHash},
		"id":    {ytID},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", mp3musicsOrigin+"/")
	req.Header.Set("Origin", mp3musicsOrigin)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: stream: %w", p.Name(), err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: stream status %d", p.Name(), resp.StatusCode)
	}
	var parsed mp3musicsStreamResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("%s: stream json: %w", p.Name(), err)
	}
	link := playableLink(parsed.Link)
	if link == "" {
		return "", fmt.Errorf("%s: empty stream link", p.Name())
	}
	return link, nil
}

// Mp3musicsFileRef reports whether link is a deferred mp3musics.pro /file/ URL.
func Mp3musicsFileRef(link string) (filePath string, ok bool) {
	link = strings.TrimSpace(link)
	u, err := url.Parse(link)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Host)
	if host != "mp3musics.pro" && !strings.HasSuffix(host, ".mp3musics.pro") {
		return "", false
	}
	if !strings.HasPrefix(u.Path, "/file/") {
		return "", false
	}
	if _, ok := youtubeIDFromMp3musicsFile(u.Path); !ok {
		return "", false
	}
	return u.Path, true
}
