package service

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"uber-eats/internals/entities"
	statemachine "uber-eats/internals/stateMachine"
)

type OrderService struct {
	mu     sync.RWMutex
	orders map[uuid.UUID]*entities.Order
	sm     *statemachine.StateMachine
}

func NewOrderService() *OrderService {
	return &OrderService{
		orders: make(map[uuid.UUID]*entities.Order),
		sm:     statemachine.NewStateMachine(),
	}
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

	s.mu.Lock()
	s.orders[order.Id] = order
	s.mu.Unlock()

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
