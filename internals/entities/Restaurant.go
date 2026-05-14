package entities

import "github.com/google/uuid"

type Restaurant struct {
	Id      uuid.UUID
	Name    string
	Address string
	Menu    []Item
}
