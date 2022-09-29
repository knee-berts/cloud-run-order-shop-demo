package model

import "time"

type Order struct {
	OrderId        int64
	ProductId      int64
	CustomerId     int64
	Quantity       int64
	Status         string
	OrderDate      string
	FulfillmentHub string
	LastUpdateZone string
	LastUpdateTime time.Time
}

type Orders struct {
	Orders  []*Order
	PodZone string
}

type OrdersStatus struct {
	OrderCount int64
	Status     string
}

type OrdersStatusJSON struct {
	OrderCount string
	Status     string
}
