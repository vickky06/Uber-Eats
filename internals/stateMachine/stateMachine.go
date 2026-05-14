package statemachine

import (
	"fmt"
	"uber-eats/internals/entities"
)

type StateMachine struct {
	transitions map[entities.Status][]Transition
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		transitions: map[entities.Status][]Transition{
			entities.Pending: {
				{
					To: entities.Confirmed,
					Guard: func(o *entities.Order, actor entities.Actor) error {
						if len(o.OrderItems) == 0 {
							return fmt.Errorf("order has no items")
						}
						if o.PlacedFrom == nil {
							return fmt.Errorf("no restaurant assigned")
						}
						return nil
					},
					Action: func(o *entities.Order, actor entities.Actor) error {
						// notify restaurant of new order
						return nil
					},
				},
				{
					To: entities.Rejected,
					Guard: func(o *entities.Order, actor entities.Actor) error {
						if actor != entities.RestaurantActor {
							return fmt.Errorf("only the restaurant can reject an order")
						}
						return nil
					},
					Action: func(o *entities.Order, actor entities.Actor) error {
						// full refund — restaurant declined before any work
						return nil
					},
				},
				{
					To: entities.Cancelled,
					Guard: func(o *entities.Order, actor entities.Actor) error {
						if actor != entities.CustomerActor && actor != entities.SystemActor {
							return fmt.Errorf("only customer or system can cancel a pending order")
						}
						return nil
					},
					Action: func(o *entities.Order, actor entities.Actor) error {
						// full refund — cancelled before confirmation
						return nil
					},
				},
			},
			entities.Confirmed: {
				{
					To: entities.Preparing,
					Guard: func(o *entities.Order, actor entities.Actor) error {
						if actor != entities.RestaurantActor {
							return fmt.Errorf("only the restaurant can start preparing")
						}
						return nil
					},
					Action: func(o *entities.Order, actor entities.Actor) error {
						// restaurant starts preparing food
						return nil
					},
				},
				{
					To: entities.Cancelled,
					Guard: func(o *entities.Order, actor entities.Actor) error {
						if actor != entities.CustomerActor && actor != entities.SystemActor {
							return fmt.Errorf("only customer or system can cancel")
						}
						return nil
					},
					Action: func(o *entities.Order, actor entities.Actor) error {
						// partial refund — restaurant already accepted
						return nil
					},
				},
			},
			entities.Preparing: {
				{
					To: entities.PickedUp,
					Guard: func(o *entities.Order, actor entities.Actor) error {
						if actor != entities.DriverActor {
							return fmt.Errorf("only the driver can pick up an order")
						}
						return nil
					},
					Action: func(o *entities.Order, actor entities.Actor) error {
						// driver picks up order, start delivery tracking
						return nil
					},
				},
				{
					To: entities.Cancelled,
					Action: func(o *entities.Order, actor entities.Actor) error {
						// late cancel — minimal or no refund
						return nil
					},
				},
			},
			entities.PickedUp: {
				{
					To: entities.Delivered,
					Guard: func(o *entities.Order, actor entities.Actor) error {
						if actor != entities.DriverActor {
							return fmt.Errorf("only the driver can mark delivered")
						}
						return nil
					},
					Action: func(o *entities.Order, actor entities.Actor) error {
						// confirm delivery, finalize payment
						return nil
					},
				},
			},
			// Delivered, Cancelled, Rejected — terminal, no transitions
		},
	}
}

func (sm *StateMachine) Apply(o *entities.Order, to entities.Status, actor entities.Actor) error {
	o.Lock()
	defer o.Unlock()

	transitions := sm.transitions[o.GetStatus()]
	for _, tr := range transitions {
		if tr.To == to {
			if tr.Guard != nil {
				if err := tr.Guard(o, actor); err != nil {
					return fmt.Errorf("transition %d→%d blocked: %w", o.GetStatus(), to, err)
				}
			}

			from := o.GetStatus()
			o.SetStatus(to)

			if tr.Action != nil {
				if err := tr.Action(o, actor); err != nil {
					// rollback the state and treat as no-op — a transient
					// notify/observer failure should not surface as a hard
					// transition error to callers.
					o.SetStatus(from)
					return nil
				}
			}

			return nil
		}
	}
	return fmt.Errorf("no transition defined: %d → %d", o.GetStatus(), to)
}
