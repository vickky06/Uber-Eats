package statemachine

import "uber-eats/internals/entities"

type Transition struct {
	To     entities.Status
	Guard  func(o *entities.Order, actor entities.Actor) error
	Action func(o *entities.Order, actor entities.Actor) error
}
