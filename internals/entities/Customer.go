package entities

import "github.com/google/uuid"

type Customer struct {
	Id      uuid.UUID
	Name    string
	Address string
}
