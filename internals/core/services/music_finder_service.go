package services

import (
	"github.com/andiq123/FindVibeFiber/internals/core/models"
	"github.com/andiq123/FindVibeFiber/internals/core/ports"
	"github.com/andiq123/FindVibeFiber/internals/scrapper"
	"github.com/gocolly/colly/v2"
)

type MusicFinderService struct {
	sourceLink string
}

var _ ports.IMusicFinderService = (*MusicFinderService)(nil)

func NewMusicFinderService() *MusicFinderService {
	return &MusicFinderService{
		sourceLink: "https://muzvibe.org/search/",
	}
}

func (m *MusicFinderService) FindMusic(query string) ([]models.Song, error) {
	c := scrapper.GetInstance()
	songs := make([]models.Song, 0, 40)

	c.OnHTML("#results", func(e *colly.HTMLElement) {
		e.ForEach(".item", func(_ int, row *colly.HTMLElement) {
			image := row.ChildAttr("img", "src")
			title := row.ChildText(".title")
			artist := row.ChildText(".artist")
			link := row.ChildAttr(".link", "href")

			song := models.NewSong(title, artist, image, link)
			songs = append(songs, *song)
		})
	})

	if err := c.Visit(m.sourceLink + query); err != nil {
		return nil, err
	}

	return songs, nil
}
