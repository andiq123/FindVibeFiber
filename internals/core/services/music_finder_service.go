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
	songChannel := make(chan models.Song)
	errorChannel := make(chan error)
	done := make(chan struct{})
	songs := make([]models.Song, 0, 40)

	go func() {
		defer close(songChannel)
		defer close(errorChannel)
		defer close(done)

		m.colly.OnHTML("tbody", func(e *colly.HTMLElement) {
			e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
				image := row.ChildAttr("img", "data-src")
				link := row.ChildAttr("div[data-id]", "data-id")

				fullText := row.ChildText("a")
				artist, title := parseArtistAndTitle(fullText)

				song := models.NewSong(title, artist, image, link)
				songChannel <- *song
			})
		})

		if err := m.colly.Visit(m.sourceLink + query); err != nil {
			errorChannel <- err
			return
		}

		m.colly.Wait()
		done <- struct{}{}
	}()

	go func() {
		for song := range songChannel {
			songs = append(songs, song)
		}
	}()

	select {
	case <-done:
		return songs, nil
	case err := <-errorChannel:
		return nil, err
	}
}

func parseArtistAndTitle(text string) (string, string) {
	if artist, title, found := strings.Cut(text, " - "); found {
		return artist, title
	}
	return text, ""
}
