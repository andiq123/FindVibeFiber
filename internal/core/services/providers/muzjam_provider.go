package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

var muzjamTotalRegex = regexp.MustCompile(`найдено\s+(\d+)\s+песен`)

const (
	muzjamOrigin = "https://muzjam.org"
	muzjamSearch = muzjamOrigin + "/search/"
)

// Soft outbound pace — bursty parallel resolve is what trips their edge.
var muzjamPace = newPacer(180 * time.Millisecond)

type MuzJamProvider struct {
	*BaseProvider
	warm sync.Once
}

func NewMuzJamProvider(client *http.Client) *MuzJamProvider {
	return &MuzJamProvider{BaseProvider: NewBaseProvider("MuzJam", 8, client)}
}

// UseRotator enables proxy hop on 403/429/challenge pages.
func (p *MuzJamProvider) UseRotator(r proxyRotator) *MuzJamProvider {
	p.WithRotator(r)
	return p
}

func (p *MuzJamProvider) SearchWithPage(ctx context.Context, query string, page int) ([]domain.ProviderResult, error) {
	if page < 1 {
		page = 1
	}
	if err := muzjamPace.wait(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", p.Name(), err)
	}
	p.ensureWarm(ctx)

	apiURL := muzjamSearch + url.QueryEscape(query)
	if page > 1 {
		apiURL = fmt.Sprintf("%s/%d", apiURL, page)
	}

	doc, err := p.fetchDocument(ctx, apiURL, muzjamOrigin+"/")
	if err != nil {
		return nil, err
	}
	return p.parseResults(doc, page), nil
}

func (p *MuzJamProvider) ensureWarm(ctx context.Context) {
	p.warm.Do(func() {
		// Single homepage hit (no proxy burn) — seed cookies before /search.
		warmCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		defer cancel()
		_, _, _ = p.doFetch(warmCtx, muzjamOrigin+"/", "")
	})
}

func (p *MuzJamProvider) parseResults(doc *goquery.Document, page int) []domain.ProviderResult {
	pagination := p.pagination(doc, page)
	results := make([]domain.ProviderResult, 0, 40)
	rank := 1

	doc.Find("#results .item").Each(func(_ int, s *goquery.Selection) {
		title := text(s.Find(".title"))
		artist := text(s.Find(".artist a"))
		if artist == "" {
			artist = text(s.Find(".artist"))
		}
		link := playableLink(s.Find("a.link").AttrOr("href", ""))
		if title == "" || artist == "" || link == "" {
			return
		}
		song := domain.NewSong(
			title,
			artist,
			absoluteURL(s.Find(".cover img").AttrOr("src", "")),
			link,
		)
		results = append(results, domain.NewProviderResult(*song, p.Name(), rank, pagination))
		rank++
	})
	return results
}

func (p *MuzJamProvider) pagination(doc *goquery.Document, page int) *domain.PaginationInfo {
	info := &domain.PaginationInfo{
		CurrentPage: page,
		HasPrevPage: page > 1,
		TotalPages:  1,
	}
	if m := muzjamTotalRegex.FindStringSubmatch(doc.Find(".h1 h1").Text()); len(m) > 1 {
		info.TotalResults, _ = strconv.Atoi(m[1])
	}
	maxPage := page
	doc.Find("#paginator .page_number, #paginator .active_page").Each(func(_ int, s *goquery.Selection) {
		if n, err := strconv.Atoi(strings.TrimSpace(s.Text())); err == nil && n > maxPage {
			maxPage = n
		}
	})
	info.TotalPages = maxPage
	info.HasNextPage = page < maxPage
	return info
}

// pacer serializes bursts so parallel explore/radio resolves don't stampede MuzJam.
type pacer struct {
	mu     sync.Mutex
	last   time.Time
	minGap time.Duration
}

func newPacer(minGap time.Duration) *pacer {
	return &pacer{minGap: minGap}
}

func (p *pacer) wait(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.last.IsZero() {
		wait := p.minGap - time.Since(p.last)
		if wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	p.last = time.Now()
	return nil
}
