package services

import (
	"strings"

	"github.com/andiq123/FindVibeFiber/internals/core/models"
	"github.com/andiq123/FindVibeFiber/internals/core/ports"
	"github.com/gocolly/colly/v2"
)

type MusicFinderService struct {
	sourceLink string
}

var _ ports.IMusicFinderService = (*MusicFinderService)(nil)

func NewMusicFinderService() *MusicFinderService {
	return &MusicFinderService{
		sourceLink: "https://muzsky.net/search/",
	}
}

func (m *MusicFinderService) FindMusic(query string) ([]models.Song, error) {
	collector := colly.NewCollector()

	songs := make([]models.Song, 0, 40)

	collector.OnHTML("tbody", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
			image := row.ChildAttr("img", "data-src")
			link := row.ChildAttr("div[data-id]", "data-id")

			fullText := row.ChildText("a")
			artist, title := parseArtistAndTitle(fullText)

			song := models.NewSong(title, artist, image, link)
			songs = append(songs, *song)
		})
	})

	if err := collector.Visit(m.sourceLink + query); err != nil {
		return nil, err
	}

	collector.Wait()
	return songs, nil
}

func parseArtistAndTitle(text string) (string, string) {
	if artist, title, found := strings.Cut(text, " - "); found {
		return artist, title
	}
	return text, ""
}
