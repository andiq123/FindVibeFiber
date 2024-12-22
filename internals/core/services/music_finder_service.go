package services

import (
	"strings"

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
		sourceLink: "https://muzlen.me/?q=",
	}
}

func (m *MusicFinderService) FindMusic(query string) ([]models.Song, error) {
	c := scrapper.GetInstance()
	songs := make([]models.Song, 0, 40)

	c.OnHTML(".files-cols-2", func(e *colly.HTMLElement) {
		e.ForEach(".mp3", func(_ int, row *colly.HTMLElement) {
			image := row.ChildAttr("img", "data-src")
			link := row.ChildAttr("div[mp3source]", "mp3source")

			fullText := row.ChildText("span")
			artist, title := parseArtistAndTitle(fullText)

			song := models.NewSong(title, artist, image, link)
			songs = append(songs, *song)
		})
	})

	if err := c.Visit(m.sourceLink + query); err != nil {
		return nil, err
	}

	return songs, nil
}

func parseArtistAndTitle(text string) (string, string) {
	if artist, title, found := strings.Cut(text, " - "); found {
		return artist, title
	}
	return text, ""
}
