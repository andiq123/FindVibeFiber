package models

import "github.com/google/uuid"

type Song struct {
	Id     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Image  string `json:"image"`
	Link   string `json:"link"`
}

func NewSong(title string, artist string, image string, link string) *Song {
	return &Song{
		Id:     uuid.New().String(),
		Title:  title,
		Artist: artist,
		Image:  image,
		Link:   link,
	}
}
