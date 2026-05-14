package entities

import (
	"sync"

	"github.com/google/uuid"
)

type ItemID int
type Item struct {
	Id         ItemID
	ItemName   string
	ItemDetail string
	Price      int64
}
type Quantity int

type Status int

const (
	Pending Status = iota
	Confirmed
	Preparing
	PickedUp
	Delivered
	Cancelled
	Rejected
)

type Actor int

const (
	CustomerActor Actor = iota
	RestaurantActor
	DriverActor
	SystemActor
)

type DeliveryFeeStrategy interface {
	CalculateDeliveryFee(distanceKm float64) int64
}

type FlatDeliveryFee struct {
	FeeInCents int64
}

func (f FlatDeliveryFee) CalculateDeliveryFee(distanceKm float64) int64 {
	return f.FeeInCents
}

type DistanceBasedFee struct {
	BaseFeeInCents  int64
	PerKmFeeInCents int64
}

func (d DistanceBasedFee) CalculateDeliveryFee(distanceKm float64) int64 {
	return d.BaseFeeInCents + int64(distanceKm)*d.PerKmFeeInCents
}

type FreeDeliveryFee struct{}

func (f FreeDeliveryFee) CalculateDeliveryFee(distanceKm float64) int64 {
	return 0
}

type SurgeDeliveryFee struct {
	Base       DeliveryFeeStrategy
	Multiplier float64
}

func (s SurgeDeliveryFee) CalculateDeliveryFee(distanceKm float64) int64 {
	return int64(float64(s.Base.CalculateDeliveryFee(distanceKm)) * s.Multiplier)
}

type OrderLine struct {
	Quantity     Quantity
	PriceAtOrder int64 // snapshot of item price in cents at time of order
}

type Order struct {
	OrderItems  map[ItemID]OrderLine
	TotalPrice  int64
	DeliveryFee DeliveryFeeStrategy
	DistanceKm  float64
	Id          uuid.UUID
	Owner       *Customer
	status      Status
	PlacedFrom  *Restaurant
	mu          sync.Mutex
}

func (o *Order) Lock()   { o.mu.Lock() }
func (o *Order) Unlock() { o.mu.Unlock() }

func (o *Order) GetStatus() Status { return o.status }
func (o *Order) SetStatus(s Status) { o.status = s }

func (o *Order) CalculateTotal() int64 {
	var subtotal int64
	for _, line := range o.OrderItems {
		subtotal += line.PriceAtOrder * int64(line.Quantity)
	}

	var deliveryFee int64
	if o.DeliveryFee != nil {
		deliveryFee = o.DeliveryFee.CalculateDeliveryFee(o.DistanceKm)
	}

	o.TotalPrice = subtotal + deliveryFee
	return o.TotalPrice
}
