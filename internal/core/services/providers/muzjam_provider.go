package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

var totalResultsRegex = regexp.MustCompile(`найдено\s+(\d+)\s+песен`)

const (
	muzjamOrigin = "https://muzjam.org"
	muzjamSearch = muzjamOrigin + "/search/"
)

type MuzJamProvider struct {
	*BaseProvider
}

func NewMuzJamProvider(client *http.Client) *MuzJamProvider {
	return &MuzJamProvider{
		BaseProvider: NewBaseProvider("MuzJam", 8, client),
	}
}

func (p *MuzJamProvider) SearchWithPage(ctx context.Context, query string, page int) ([]domain.ProviderResult, error) {
	if page < 1 {
		page = 1
	}

	req, err := p.createSearchRequest(ctx, query, page)
	if err != nil {
		return nil, err
	}

	resp, err := p.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("muzjam: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("muzjam: unexpected status code: %d", resp.StatusCode)
	}

	return p.parseResults(resp.Body, page)
}

func (p *MuzJamProvider) createSearchRequest(ctx context.Context, query string, page int) (*http.Request, error) {
	apiURL := muzjamSearch + url.QueryEscape(query)
	if page > 1 {
		apiURL = fmt.Sprintf("%s/%d", apiURL, page)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("muzjam: request creation failed: %w", err)
	}

	p.AddBrowserHeaders(req, muzjamOrigin+"/")
	return req, nil
}

func (p *MuzJamProvider) parseResults(body io.Reader, currentPage int) ([]domain.ProviderResult, error) {
	doc, err := goquery.NewDocumentFromReader(io.LimitReader(body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("muzjam: failed to parse HTML: %w", err)
	}

	paginationInfo := p.extractPaginationInfo(doc, currentPage)
	results := make([]domain.ProviderResult, 0, 40)
	rank := 1

	doc.Find("#results .item").Each(func(_ int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".title").Text())
		artist := strings.TrimSpace(s.Find(".artist a").Text())
		image := absoluteURL(s.Find(".cover img").AttrOr("src", ""))
		link := absoluteURL(s.Find("a.link").AttrOr("href", ""))

		if title == "" || artist == "" {
			return
		}

		song := domain.NewSong(title, artist, image, link)
		results = append(results, *domain.NewProviderResultWithPagination(
			*song,
			p.Name(),
			0,
			rank,
			paginationInfo,
		))
		rank++
	})

	return results, nil
}

func (p *MuzJamProvider) extractPaginationInfo(doc *goquery.Document, currentPage int) *domain.PaginationInfo {
	info := &domain.PaginationInfo{
		CurrentPage:  currentPage,
		TotalResults: 0,
		HasNextPage:  false,
		HasPrevPage:  currentPage > 1,
		TotalPages:   1,
	}

	h1Text := doc.Find(".h1 h1").Text()
	if matches := totalResultsRegex.FindStringSubmatch(h1Text); len(matches) > 1 {
		if total, err := strconv.Atoi(matches[1]); err == nil {
			info.TotalResults = total
		}
	}

	maxPage := currentPage
	doc.Find("#paginator .page_number, #paginator .active_page").Each(func(_ int, s *goquery.Selection) {
		pageText := strings.TrimSpace(s.Text())
		if pageNum, err := strconv.Atoi(pageText); err == nil && pageNum > maxPage {
			maxPage = pageNum
		}
	})

	info.TotalPages = maxPage
	info.HasNextPage = currentPage < maxPage

	return info
}

func absoluteURL(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	return raw
}
