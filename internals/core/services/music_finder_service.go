package services

import (
	"strings"

	"github.com/andiq123/FindVibeFiber/internals/core/models"
	"github.com/andiq123/FindVibeFiber/internals/core/ports"
	"github.com/gocolly/colly/v2"
)

type MusicFinderService struct {
	sourceLink string
	colly      *colly.Collector
}

var _ ports.IMusicFinderService = (*MusicFinderService)(nil)

func NewMusicFinderService(colly *colly.Collector) *MusicFinderService {
	return &MusicFinderService{
		sourceLink: "https://muzsky.net/search/",
		colly:      colly,
	}
}

func (m *MusicFinderService) FindMusic(query string) ([]models.Song, error) {
	songs := make([]models.Song, 0, 40)

	m.colly.OnHTML("tbody", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(i int, e *colly.HTMLElement) {
			image := e.ChildAttr("img", "data-src")
			link := e.ChildAttr("div[data-id]", "data-id")

			artist, title, found := strings.Cut(e.ChildText("a"), " - ")
			if !found {
				title = ""
			}

			song := models.NewSong(title, artist, image, link)
			songs = append(songs, *song)
		})
	})

	err := m.colly.Visit(m.sourceLink + query)
	if err != nil {
		return nil, err
	}
	m.colly.Wait()

	return songs, nil
}