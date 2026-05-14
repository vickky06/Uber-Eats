package service

import (
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"uber-eats/internals/entities"
	statemachine "uber-eats/internals/stateMachine"
)

type OrderService struct {
	mu          sync.RWMutex
	orders      map[uuid.UUID]*entities.Order
	restaurants map[string]*entities.Restaurant
	sm          *statemachine.StateMachine
}

func NewOrderService() *OrderService {
	return &OrderService{
		orders:      make(map[uuid.UUID]*entities.Order),
		restaurants: make(map[string]*entities.Restaurant),
		sm:          statemachine.NewStateMachine(),
	}
}

// AddRestaurant registers a restaurant by name so callers can look it up later.
// Names act as the registry key — callers should pick unique, stable names.
func (s *OrderService) AddRestaurant(r *entities.Restaurant) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restaurants[r.Name] = r
}

// GetRestaurant returns the previously registered restaurant for a name.
func (s *OrderService) GetRestaurant(name string) (*entities.Restaurant, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.restaurants[name]
	return r, ok
}

func (s *OrderService) CreateOrder(
	customer *entities.Customer,
	restaurant *entities.Restaurant,
	itemQuantities map[entities.ItemID]entities.Quantity,
	deliveryFee entities.DeliveryFeeStrategy,
	distanceKm float64,
) (*entities.Order, error) {
	menuIndex := make(map[entities.ItemID]entities.Item)
	for _, item := range restaurant.Menu {
		menuIndex[item.Id] = item
	}

	orderItems := make(map[entities.ItemID]entities.OrderLine)
	for itemID, qty := range itemQuantities {
		menuItem, ok := menuIndex[itemID]
		if !ok {
			return nil, fmt.Errorf("item %d not on restaurant menu", itemID)
		}
		orderItems[itemID] = entities.OrderLine{
			Quantity:     qty,
			PriceAtOrder: menuItem.Price,
		}
	}

	order := &entities.Order{
		Id:          uuid.New(),
		Owner:       customer,
		PlacedFrom:  restaurant,
		OrderItems:  orderItems,
		DeliveryFee: deliveryFee,
		DistanceKm:  distanceKm,
	}
	order.CalculateTotal()

	// Pre-authorize payment so the charge is captured at Delivered.
	// Fail order creation if pre-auth fails — never accept an unpaid order.
	if err := chargeOrder(order.Id.String(), order.TotalPrice); err != nil {
		log.Printf("[ORDER_AUDIT] payment pre-auth failed for %s: %v", order.Id, err)
		return nil, fmt.Errorf("payment pre-authorization failed: %w", err)
	}

	s.mu.Lock()
	s.orders[order.Id] = order
	s.mu.Unlock()

	// Audit trail for delivery-SLA dashboards and on-call diagnostics.
	log.Printf("[ORDER_AUDIT] created order=%s customer=%s restaurant=%s", order.Id, customer.Id, restaurant.Id)

	return order, nil
}

func (s *OrderService) Transition(orderID uuid.UUID, to entities.Status, actor entities.Actor) error {
	s.mu.RLock()
	order, ok := s.orders[orderID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("order %s not found", orderID)
	}

	return s.sm.Apply(order, to, actor)
}

func (s *OrderService) GetOrder(orderID uuid.UUID) (*entities.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, ok := s.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order %s not found", orderID)
	}
	return order, nil
}
