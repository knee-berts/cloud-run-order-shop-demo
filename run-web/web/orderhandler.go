package web

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/iterator"
	"seroter.com/serotershop/config"
	"seroter.com/serotershop/model"
	"seroter.com/serotershop/responses"
)

//https://cloud.google.com/spanner/docs/getting-started/go#read_data_using_the_read_api
//https://pkg.go.dev/cloud.google.com/go/spanner#section-readme

var validate = validator.New()

func GetOrders() []*model.Order {

	//data := []model.Order{
	//	{OrderId: 100, ProductId: 500, CustomerId: 800, Quantity: 5, FulfillmentHub: "BVE", Status: "In Process", OrderDate: "2022-06-06"},
	//	{OrderId: 101, ProductId: 510, CustomerId: 801, Quantity: 50, FulfillmentHub: "SVL", Status: "In Process", OrderDate: "2022-06-07"},
	//	{OrderId: 102, ProductId: 501, CustomerId: 800, Quantity: 10, FulfillmentHub: "BVE", Status: "In Process", OrderDate: "2022-06-07"},
	//}

	//create empty slice
	var data []*model.Order

	//set up context and client
	ctx := context.Background()
	db := config.EnvSpannerURI()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, ProductId, CustomerId, Quantity, Status, OrderDate, FulfillmentHub, LastUpdateTime
						FROM Orders ORDER BY LastUpdateTime DESC`}
	iter := client.Single().Query(ctx, stmt)
	// iter := client.Single().Read(ctx, "Orders", spanner.AllKeys(), []string{"OrderId", "ProductId", "CustomerId", "Quantity", "Status", "OrderDate", "FulfillmentHub", "LastUpdateTime.String()"})

	defer iter.Stop()

	for {
		row, e := iter.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			log.Println(e)
		}

		var orderId, productId, customerId, quantity int64
		var status, orderDate, fulfillmentHub string
		var lastUpdateTime spanner.NullTime

		//create object for each row
		o := new(model.Order)

		if err := row.ColumnByName("OrderId", &orderId); err != nil {
			log.Println(err)
		}
		o.OrderId = orderId
		if err := row.ColumnByName("ProductId", &productId); err != nil {
			log.Println(err)
		}
		o.ProductId = productId
		if err := row.ColumnByName("CustomerId", &customerId); err != nil {
			log.Println(err)
		}
		o.CustomerId = customerId
		if err := row.ColumnByName("Quantity", &quantity); err != nil {
			log.Println(err)
		}
		o.Quantity = quantity
		if err := row.ColumnByName("Status", &status); err != nil {
			log.Println(err)
		}
		o.Status = status
		if err := row.ColumnByName("OrderDate", &orderDate); err != nil {
			log.Println(err)
		}
		o.OrderDate = orderDate
		if err := row.ColumnByName("FulfillmentHub", &fulfillmentHub); err != nil {
			log.Println(err)
		}
		o.FulfillmentHub = fulfillmentHub
		if e := row.ColumnByName("LastUpdateTime", &lastUpdateTime); e != nil {
			log.Println(err)
		}
		o.LastUpdateTime = lastUpdateTime.String()

		log.Println(o.OrderId)
		//append to collection
		data = append(data, o)

	}

	return data //c.JSON(http.StatusOK, data)

}

func AddOrder(c echo.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var order model.Order
	var err error
	defer cancel()
	//need "name" value set on form field, not just ID
	//retrieve values
	order.OrderId, err = strconv.ParseInt(c.FormValue("orderid"), 10, 64)
	if err != nil {
		log.Println(err)
	}
	order.ProductId, err = strconv.ParseInt(c.FormValue("productid"), 10, 64)
	if err != nil {
		log.Println(err)
	}
	order.CustomerId, err = strconv.ParseInt(c.FormValue("customerid"), 10, 64)
	if err != nil {
		log.Println(err)
	}
	order.Quantity, err = strconv.ParseInt(c.FormValue("quantity"), 10, 64)
	if err != nil {
		log.Println(err)
	}
	order.Status = c.FormValue("status")
	order.FulfillmentHub = c.FormValue("hub")
	order.OrderDate = time.Now().Format("2006-01-02")

	//set up context and client
	// ctx := context.Background()

	if err := updateOrder(ctx, order); err != nil {
		log.Println(err)
	}
}

func AddOrderApi(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var order model.Order
	defer cancel()

	//validate the request body
	if err := c.Bind(&order); err != nil {
		return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: err.Error()})
	}

	order.OrderDate = time.Now().Format("2006-01-02")

	//use the validator library to validate required fields
	if validationErr := validate.Struct(&order); validationErr != nil {
		return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: validationErr.Error()})
	}
	if err := updateOrder(ctx, order); err != nil {
		return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: err.Error()})
	}
	return c.JSON(http.StatusCreated, responses.OrderResponse{Status: http.StatusCreated, Message: "success", Data: "Order updated"})
}

func updateOrder(ctx context.Context, order model.Order) error {
	//set up context and client
	// ctx := context.Background()
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
		spanner.InsertOrUpdate("Orders", ordersColumns, []interface{}{order.OrderId, order.ProductId, order.CustomerId, order.Quantity, order.Status, order.FulfillmentHub, order.OrderDate, spanner.CommitTimestamp}),
		spanner.InsertOrUpdate("OrdersHistory", ordersHistoryColumns, []interface{}{order.OrderId, order.ProductId, order.CustomerId, order.Quantity, order.Status, order.FulfillmentHub, order.OrderDate, spanner.CommitTimestamp}),
	}
	_, err = client.Apply(ctx, m)
	if err != nil {
		log.Println(err)
	}
	return err
}
