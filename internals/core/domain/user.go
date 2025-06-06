package domain

import "github.com/google/uuid"

type User struct {
	Id   string `json:"id"`
	Name string `json:"username"`
}

func NewUser(name string) *User {
	return &User{
		Id:   uuid.New().String(),
		Name: name,
	}
}
