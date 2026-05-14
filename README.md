# Uber Eats — Order Service (LLD)

A small order-management service modeling a food-delivery flow. Used as a benchmark target for code-review tooling.

## Domain

- **Order** moves through a state machine: `Pending → Confirmed → Preparing → PickedUp → Delivered`, with `Cancelled` / `Rejected` as terminal off-ramps.
- **Actors** (Customer, Restaurant, Driver, System) — transitions are guarded by who initiated them.
- **Delivery fee strategies**: flat, distance-based, surge, free.
- **Money in cents** (`int64`) — prices and fees never use floats.

## Layout

```
main.go                              # demo entrypoint
internals/
  entities/                          # Order, Customer, Restaurant, fee strategies
  service/order_service.go           # CreateOrder, Transition, GetOrder
  stateMachine/                      # transition table + Apply
```

## Run

```bash
go run .
go test ./...
```
