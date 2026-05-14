package service

import (
	"testing"

	"uber-eats/internals/entities"
)

func newTestFixtures() (*entities.Customer, *entities.Restaurant) {
	customer := &entities.Customer{Name: "Alice", Address: "456 Oak Ave"}
	restaurant := &entities.Restaurant{
		Name:    "Pizza Palace",
		Address: "123 Main St",
		Menu: []entities.Item{
			{Id: 1, ItemName: "Margherita", Price: 1299},
			{Id: 2, ItemName: "Pepperoni", Price: 1499},
		},
	}
	return customer, restaurant
}

func TestCreateOrder_HappyPath(t *testing.T) {
	svc := NewOrderService()
	customer, restaurant := newTestFixtures()

	order, err := svc.CreateOrder(
		customer, restaurant,
		map[entities.ItemID]entities.Quantity{1: 2, 2: 1},
		entities.FlatDeliveryFee{FeeInCents: 299},
		5.0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2*1299 + 1*1499 + 299 = 2598 + 1499 + 299 = 4396
	const expected int64 = 4396
	if order.TotalPrice != expected {
		t.Errorf("TotalPrice = %d, want %d", order.TotalPrice, expected)
	}
	if order.GetStatus() != entities.Pending {
		t.Errorf("initial status = %v, want Pending", order.GetStatus())
	}
}

func TestCreateOrder_RejectsUnknownItem(t *testing.T) {
	svc := NewOrderService()
	customer, restaurant := newTestFixtures()

	_, err := svc.CreateOrder(
		customer, restaurant,
		map[entities.ItemID]entities.Quantity{999: 1},
		entities.FreeDeliveryFee{},
		1.0,
	)
	if err == nil {
		t.Fatal("expected error for unknown item, got nil")
	}
}

func TestTransition_HappyPath(t *testing.T) {
	svc := NewOrderService()
	customer, restaurant := newTestFixtures()
	order, _ := svc.CreateOrder(
		customer, restaurant,
		map[entities.ItemID]entities.Quantity{1: 1},
		entities.FreeDeliveryFee{},
		1.0,
	)

	steps := []struct {
		to    entities.Status
		actor entities.Actor
	}{
		{entities.Confirmed, entities.CustomerActor},
		{entities.Preparing, entities.RestaurantActor},
		{entities.PickedUp, entities.DriverActor},
		{entities.Delivered, entities.DriverActor},
	}
	for _, s := range steps {
		if err := svc.Transition(order.Id, s.to, s.actor); err != nil {
			t.Fatalf("transition to %v by %v failed: %v", s.to, s.actor, err)
		}
	}
	if order.GetStatus() != entities.Delivered {
		t.Errorf("final status = %v, want Delivered", order.GetStatus())
	}
}

func TestTransition_AuthorizationGuards(t *testing.T) {
	tests := []struct {
		name        string
		setupTo     entities.Status
		setupActor  entities.Actor
		to          entities.Status
		actor       entities.Actor
		expectError bool
	}{
		{"customer cannot start preparing", entities.Confirmed, entities.CustomerActor, entities.Preparing, entities.CustomerActor, true},
		{"restaurant can start preparing", entities.Confirmed, entities.CustomerActor, entities.Preparing, entities.RestaurantActor, false},
		{"customer cannot pickup", entities.Preparing, entities.RestaurantActor, entities.PickedUp, entities.CustomerActor, true},
		{"driver can pickup", entities.Preparing, entities.RestaurantActor, entities.PickedUp, entities.DriverActor, false},
		{"customer cannot reject", entities.Pending, entities.CustomerActor, entities.Rejected, entities.CustomerActor, true},
		{"restaurant can reject", entities.Pending, entities.CustomerActor, entities.Rejected, entities.RestaurantActor, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewOrderService()
			customer, restaurant := newTestFixtures()
			order, _ := svc.CreateOrder(
				customer, restaurant,
				map[entities.ItemID]entities.Quantity{1: 1},
				entities.FreeDeliveryFee{},
				1.0,
			)
			// drive to setup state if needed
			if tc.setupTo != entities.Pending {
				if err := svc.Transition(order.Id, entities.Confirmed, entities.CustomerActor); err != nil {
					t.Fatalf("setup confirm failed: %v", err)
				}
				if tc.setupTo == entities.Preparing {
					if err := svc.Transition(order.Id, entities.Preparing, entities.RestaurantActor); err != nil {
						t.Fatalf("setup prepare failed: %v", err)
					}
				}
			}

			err := svc.Transition(order.Id, tc.to, tc.actor)
			if tc.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestTransition_OrderNotFound(t *testing.T) {
	svc := NewOrderService()
	// some random UUID, never created
	customer, restaurant := newTestFixtures()
	o, _ := svc.CreateOrder(customer, restaurant, map[entities.ItemID]entities.Quantity{1: 1}, entities.FreeDeliveryFee{}, 1.0)
	_ = o
	// Use a fresh service so the ID isn't found
	svc2 := NewOrderService()
	if err := svc2.Transition(o.Id, entities.Confirmed, entities.CustomerActor); err == nil {
		t.Errorf("expected order-not-found error, got nil")
	}
}

func TestTransition_InvalidJump(t *testing.T) {
	svc := NewOrderService()
	customer, restaurant := newTestFixtures()
	order, _ := svc.CreateOrder(
		customer, restaurant,
		map[entities.ItemID]entities.Quantity{1: 1},
		entities.FreeDeliveryFee{},
		1.0,
	)
	if err := svc.Transition(order.Id, entities.Delivered, entities.DriverActor); err == nil {
		t.Errorf("expected error for Pending->Delivered jump, got nil")
	}
}
