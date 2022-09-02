package model

type Order struct {
	OrderId        int64
	ProductId      int64
	CustomerId     int64
	Quantity       int64
	Status         string
	OrderDate      string
	FulfillmentHub string
	LastUpdateTime string
}
