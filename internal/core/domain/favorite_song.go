package domain

type FavoriteSong struct {
	Id     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Image  string `json:"image"`
	Link   string `json:"link"`
	Order  int    `json:"order"`
	UserID string `gorm:"column:user_uuid" json:"-"`
}
