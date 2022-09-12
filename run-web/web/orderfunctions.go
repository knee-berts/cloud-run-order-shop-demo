package web

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/spanner"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/iterator"
	"seroter.com/serotershop/config"
	"seroter.com/serotershop/model"
)

type SpannerContext struct {
	echo.Context
}

// Create Middleware that sets Spanner Connection
func SetSpannerConnection(next echo.HandlerFunc) echo.HandlerFunc {

	// return ctx, client
	return func(c echo.Context) error {
		ctx := context.Background()
		db := config.EnvSpannerURI()
		client, err := spanner.NewClient(ctx, db)
		if err != nil {
			log.Fatal(err)
		}
		defer client.Close()
		c.Set("spanner_client", *client)
		c.Set("spanner_context", ctx)
		return next(c)
	}
}

// Helper function to retrieve spanner client and context
func getSpannerConnection(c echo.Context) (context.Context, spanner.Client) {
	return c.Get("spanner_context").(context.Context),
		c.Get("spanner_client").(spanner.Client)
}

func updateOrder(c echo.Context, order model.Order) error {
	ctx, client := getSpannerConnection(c)

	ordersColumns := []string{"OrderId", "ProductId", "CustomerId", "Quantity", "Status", "FulfillmentHub", "OrderDate", "LastUpdateTime"}
	ordersHistoryColumns := []string{"OrderId", "ProductId", "CustomerId", "Quantity", "Status", "FulfillmentHub", "OrderDate", "TimeStamp"}
	m := []*spanner.Mutation{
		spanner.Insert("Orders", ordersColumns, []interface{}{order.OrderId, order.ProductId, order.CustomerId, order.Quantity, order.Status, order.FulfillmentHub, order.OrderDate, spanner.CommitTimestamp}),
		spanner.InsertOrUpdate("OrdersHistory", ordersHistoryColumns, []interface{}{order.OrderId, order.ProductId, order.CustomerId, order.Quantity, order.Status, order.FulfillmentHub, order.OrderDate, spanner.CommitTimestamp}),
	}
	_, err := client.Apply(ctx, m)
	if err != nil {
		log.Println(err)
	}
	return err
}

func insertOrderHistory(c echo.Context, order model.Order) error {
	ctx, client := getSpannerConnection(c)

	ordersHistoryColumns := []string{"OrderId", "ProductId", "CustomerId", "Quantity", "Status", "FulfillmentHub", "OrderDate", "TimeStamp"}
	m := []*spanner.Mutation{
		spanner.InsertOrUpdate("OrdersHistory", ordersHistoryColumns, []interface{}{order.OrderId, order.ProductId, order.CustomerId, order.Quantity, "DUPLICATE", order.FulfillmentHub, order.OrderDate, spanner.CommitTimestamp}),
	}
	_, err := client.Apply(ctx, m)
	if err != nil {
		return err
	}
	return nil
}

func listOrdersByStatus(c echo.Context, status string) ([]*model.Order, error) {
	ctx, client := getSpannerConnection(c)

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

func ordersCountByStatus(c echo.Context, status string) (*model.OrdersStatus, error) {
	ctx, client := getSpannerConnection(c)

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
