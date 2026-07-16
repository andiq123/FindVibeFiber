package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

var muzjamTotalRegex = regexp.MustCompile(`найдено\s+(\d+)\s+песен`)

const (
	muzjamOrigin = "https://muzjam.org"
	muzjamSearch = muzjamOrigin + "/search/"
)

type MuzJamProvider struct{ *BaseProvider }

func NewMuzJamProvider(client *http.Client) *MuzJamProvider {
	return &MuzJamProvider{BaseProvider: NewBaseProvider("MuzJam", 8, client)}
}

func (p *MuzJamProvider) SearchWithPage(ctx context.Context, query string, page int) ([]domain.ProviderResult, error) {
	if page < 1 {
		page = 1
	}

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

func (p *MuzJamProvider) parseResults(doc *goquery.Document, page int) []domain.ProviderResult {
	pagination := p.pagination(doc, page)
	results := make([]domain.ProviderResult, 0, 40)
	rank := 1

	doc.Find("#results .item").Each(func(_ int, s *goquery.Selection) {
		title := text(s.Find(".title"))
		artist := text(s.Find(".artist a"))
		if title == "" || artist == "" {
			return
		}
		song := domain.NewSong(
			title,
			artist,
			absoluteURL(s.Find(".cover img").AttrOr("src", "")),
			absoluteURL(s.Find("a.link").AttrOr("href", "")),
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
