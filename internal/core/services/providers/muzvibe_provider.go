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

type MuzVibeProvider struct {
	*BaseProvider
	sourceURL string
}

func NewMuzVibeProvider(client *http.Client) *MuzVibeProvider {
	return &MuzVibeProvider{
		BaseProvider: NewBaseProvider("MuzVibe", 8, client),
		sourceURL:    "https://muzvibe.org/search/",
	}
}

func (mvp *MuzVibeProvider) Search(ctx context.Context, query string) ([]domain.ProviderResult, error) {
	return mvp.SearchWithPage(ctx, query, 1)
}

func (mvp *MuzVibeProvider) SearchWithPage(ctx context.Context, query string, page int) ([]domain.ProviderResult, error) {
	if page < 1 {
		page = 1
	}

	req, err := mvp.createSearchRequest(ctx, query, page)
	if err != nil {
		return nil, err
	}

	resp, err := mvp.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("muzvibe: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("muzvibe: unexpected status code: %d", resp.StatusCode)
	}

	return mvp.parseResults(resp.Body, query, page)
}

func (mvp *MuzVibeProvider) createSearchRequest(ctx context.Context, query string, page int) (*http.Request, error) {
	apiURL := mvp.sourceURL + url.QueryEscape(query)
	if page > 1 {
		apiURL = fmt.Sprintf("%s/%d", apiURL, page)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("muzvibe: request creation failed: %w", err)
	}

	mvp.AddBrowserHeaders(req, "https://muzvibe.org/")
	return req, nil
}

func (mvp *MuzVibeProvider) parseResults(body io.Reader, query string, currentPage int) ([]domain.ProviderResult, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("muzvibe: failed to parse HTML: %w", err)
	}

	paginationInfo := mvp.extractPaginationInfo(doc, currentPage)
	results := make([]domain.ProviderResult, 0, 40)
	rank := 1

	doc.Find("#results .item").Each(func(_ int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".title").Text())
		artist := strings.TrimSpace(s.Find(".artist a").Text())
		image := s.Find(".cover img").AttrOr("src", "")
		link := s.Find("a.link").AttrOr("href", "")

		if title == "" || artist == "" {
			return
		}

		if image != "" && len(image) >= 2 && image[:2] == "//" {
			image = "https:" + image
		}
		if link != "" && len(link) >= 2 && link[:2] == "//" {
			link = "https:" + link
		}

		song := domain.NewSong(title, artist, image, link)
		matchScore := mvp.CalculateBasicMatchScore(*song, query)

		results = append(results, *domain.NewProviderResultWithPagination(
			*song,
			mvp.Name(),
			matchScore,
			rank,
			paginationInfo,
		))
		rank++
	})

	return results, nil
}

func (mvp *MuzVibeProvider) extractPaginationInfo(doc *goquery.Document, currentPage int) *domain.PaginationInfo {
	info := &domain.PaginationInfo{
		CurrentPage:  currentPage,
		TotalResults: 0,
		HasNextPage:  false,
		HasPrevPage:  currentPage > 1,
		TotalPages:   1,
	}

	h1Text := doc.Find(".h1 h1").Text()
	totalResultsRegex := regexp.MustCompile(`найдено\s+(\d+)\s+песен`)
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
