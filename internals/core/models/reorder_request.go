package models

type ReorderRequest struct {
	SongId string `json:"songId"`
	Order  int    `json:"order"`
}
