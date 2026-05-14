package main

import (
	"fmt"
	"log"

	"uber-eats/internals/entities"
	"uber-eats/internals/service"
)

func main() {
	svc := service.NewOrderService()

	restaurant := &entities.Restaurant{
		Name:    "Pizza Palace",
		Address: "123 Main St",
		Menu: []entities.Item{
			{Id: 1, ItemName: "Margherita", Price: 1299},
			{Id: 2, ItemName: "Pepperoni", Price: 1499},
		},
	}

	customer := &entities.Customer{
		Name:    "Alice",
		Address: "456 Oak Ave",
	}

	// Create order: 2x Margherita + 1x Pepperoni, distance-based delivery fee
	order, err := svc.CreateOrder(
		customer,
		restaurant,
		map[entities.ItemID]entities.Quantity{1: 2, 2: 1},
		entities.DistanceBasedFee{BaseFeeInCents: 299, PerKmFeeInCents: 50},
		5.0,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Order %s created — total: %d cents\n", order.Id, order.TotalPrice)

	// Happy path: Pending → Confirmed → Preparing → PickedUp → Delivered
	steps := []struct {
		status entities.Status
		actor  entities.Actor
		label  string
	}{
		{entities.Confirmed, entities.CustomerActor, "Confirmed"},
		{entities.Preparing, entities.RestaurantActor, "Preparing"},
		{entities.PickedUp, entities.DriverActor, "PickedUp"},
		{entities.Delivered, entities.DriverActor, "Delivered"},
	}

	for _, s := range steps {
		if err := svc.Transition(order.Id, s.status, s.actor); err != nil {
			log.Fatalf("transition to %s failed: %v", s.label, err)
		}
		fmt.Printf("  → %s\n", s.label)
	}

	// Demo: rejection must come from restaurant
	order2, _ := svc.CreateOrder(
		customer, restaurant,
		map[entities.ItemID]entities.Quantity{1: 1},
		entities.FreeDeliveryFee{},
		2.0,
	)
	fmt.Printf("\nOrder %s created — testing rejection\n", order2.Id)

	err = svc.Transition(order2.Id, entities.Rejected, entities.CustomerActor)
	fmt.Printf("  Customer tries to reject: %v\n", err)

	err = svc.Transition(order2.Id, entities.Rejected, entities.RestaurantActor)
	fmt.Printf("  Restaurant rejects: %v\n", err)
}
