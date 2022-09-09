package web

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"seroter.com/serotershop/config"
	"seroter.com/serotershop/model"
	"seroter.com/serotershop/responses"
)

//https://cloud.google.com/spanner/docs/getting-started/go#read_data_using_the_read_api
//https://pkg.go.dev/cloud.google.com/go/spanner#section-readme

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
	// client := createSpannerClient(ctx)
	db := config.EnvSpannerURI()
	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, ProductId, CustomerId, Quantity, Status, OrderDate, FulfillmentHub, LastUpdateTime
				FROM Orders ORDER BY LastUpdateTime DESC LIMIT 20`}
	// iter := client.Single().Read(ctx, "Orders", spanner.AllKeys(), []string{"OrderId", "ProductId", "CustomerId", "Quantity", "Status", "OrderDate", "FulfillmentHub", "LastUpdateTime"})
	iter := client.Single().Query(ctx, stmt)

	defer iter.Stop()

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

	if err := updateOrder(ctx, order); err != nil {
		if spanner.ErrCode(err) == codes.AlreadyExists {
			insertOrderHistory(ctx, order)
			log.Printf("Order %v already exists: %v", order.OrderId, err)
		}
		log.Println(err)
	}
}

func AddOrderApi(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var order model.Order

	//validate the request body
	if err := c.Bind(&order); err != nil {
		return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: err.Error()})
	}

	order.OrderDate = time.Now().Format("2006-01-02")

	//use the validator library to validate required fields
	var validate = validator.New()
	if validationErr := validate.Struct(&order); validationErr != nil {
		return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: validationErr.Error()})
	}
	if err := updateOrder(ctx, order); err != nil {
		if spanner.ErrCode(err) == codes.AlreadyExists {
			insertOrderHistory(ctx, order)
			return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: err.Error()})
		}
		return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: err.Error()})
	}
	return c.JSON(http.StatusCreated, responses.OrderResponse{Status: http.StatusCreated, Message: "success", Data: fmt.Sprintf("Order %v was created.", order.OrderId)})
}

func GetSubmittedOrdersCount(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 999*time.Second)
	defer cancel()
	status := c.Param("status")
	log.Println(status)
	orders, err := ordersCountByStatus(ctx, status)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: err.Error()})
	}
	return c.JSON(http.StatusOK, orders)
}

func AddRandomOrder(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var order model.Order
	defer cancel()
	//need "name" value set on form field, not just ID
	//retrieve values
	rand.Seed(time.Now().UnixNano())
	order.OrderId = rand.Int63n(1000)
	order.ProductId = rand.Int63n(1000)
	order.CustomerId = rand.Int63n(1000)
	order.Quantity = rand.Int63n(1000)
	order.Status = "SUBMITTED"
	order.FulfillmentHub = "NYC"
	order.OrderDate = time.Now().Format("2006-01-02")
	log.Println(order.OrderId)

	var validate = validator.New()
	if validationErr := validate.Struct(&order); validationErr != nil {
		return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: validationErr.Error()})
	}
	if err := updateOrder(ctx, order); err != nil {
		if spanner.ErrCode(err) == codes.AlreadyExists {
			insertOrderHistory(ctx, order)
			return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: err.Error()})
		}
		return c.JSON(http.StatusBadRequest, responses.OrderResponse{Status: http.StatusBadRequest, Message: "error", Data: err.Error()})
	}
	return c.JSON(http.StatusCreated, responses.OrderResponse{Status: http.StatusCreated, Message: "success", Data: fmt.Sprintf("Order %v was created.", order.OrderId)})

}
