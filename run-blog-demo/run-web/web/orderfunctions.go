package web

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"seroter.com/serotershop/config"
	"seroter.com/serotershop/model"
)

// func createSpannerClient(ctx context.Context) *spanner.Client {
// 	db := config.EnvSpannerURI()
// 	client, err := spanner.NewClient(ctx, db)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	defer client.Close()
// 	return client
// }

func updateOrder(ctx context.Context, order model.Order) error {
	//set up context and client
	db := config.EnvSpannerURI()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()

	//do database write
	ordersColumns := []string{"OrderId", "ProductId", "CustomerId", "Quantity", "Status", "FulfillmentHub", "OrderDate", "LastUpdateTime"}
	ordersHistoryColumns := []string{"OrderId", "ProductId", "CustomerId", "Quantity", "Status", "FulfillmentHub", "OrderDate", "TimeStamp"}
	m := []*spanner.Mutation{
		spanner.Insert("Orders", ordersColumns, []interface{}{order.OrderId, order.ProductId, order.CustomerId, order.Quantity, order.Status, order.FulfillmentHub, order.OrderDate, spanner.CommitTimestamp}),
		spanner.InsertOrUpdate("OrdersHistory", ordersHistoryColumns, []interface{}{order.OrderId, order.ProductId, order.CustomerId, order.Quantity, order.Status, order.FulfillmentHub, order.OrderDate, spanner.CommitTimestamp}),
	}
	_, err = client.Apply(ctx, m)
	if err != nil {
		log.Println(err)
	}
	return err
}

func insertOrderHistory(ctx context.Context, order model.Order) error {
	db := config.EnvSpannerURI()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()

	ordersHistoryColumns := []string{"OrderId", "ProductId", "CustomerId", "Quantity", "Status", "FulfillmentHub", "OrderDate", "TimeStamp"}
	m := []*spanner.Mutation{
		spanner.InsertOrUpdate("OrdersHistory", ordersHistoryColumns, []interface{}{order.OrderId, order.ProductId, order.CustomerId, order.Quantity, "DUPLICATE", order.FulfillmentHub, order.OrderDate, spanner.CommitTimestamp}),
	}
	_, err = client.Apply(ctx, m)
	if err != nil {
		return err
	}
	return nil
}

func listOrdersByStatus(ctx context.Context, status string) ([]*model.Order, error) {
	db := config.EnvSpannerURI()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()
	iter := client.Single().Query(ctx, spanner.NewStatement(fmt.Sprintf("SELECT OrderId FROM Orders WHERE Status = '%s'", status)))
	defer iter.Stop()

	var data []*model.Order
	for {
		row, e := iter.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			log.Println(e)
		}

		//create object for each row
		o := new(model.Order)

		//load row into struct that maps to same shape
		rerr := row.ToStruct(o)
		if rerr != nil {
			log.Println(rerr)
		}

		//append to collection
		data = append(data, o)

	}

	return data, nil
}

func ordersCountByStatus(ctx context.Context, status string) (*model.OrdersStatus, error) {
	db := config.EnvSpannerURI()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	log.Printf("SELECT OrderId FROM Orders WHERE Status = '%v'", status)
	stmt := spanner.NewStatement(fmt.Sprintf("SELECT OrderId FROM Orders WHERE Status = '%v'", status))
	iter := ro.Query(ctx, stmt)
	defer iter.Stop()
	var count int64 = 0
	for {
		row, e := iter.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			log.Println(e)
		}
		count++
		var orderId int64
		if err := row.Columns(&orderId); err != nil {
			log.Println(err)
		}
		log.Println(orderId)
	}

	var data model.OrdersStatus

	data.OrderCount = count
	data.Status = status
	return &data, nil
}
