package main

import (
	"fmt"

	"html/template"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"seroter.com/serotershop/config"
	"seroter.com/serotershop/web"
)

//https://gowebexamples.com/templates/

func main() {

	fmt.Println("started up")

	e := echo.New()
	e.Use(middleware.Logger())

	e.GET("/", web.GetHome)
	e.POST("/", web.NewOrder)
	e.POST("/addOrder", web.AddOrderApi)
	e.PUT("/addRandomOrder", web.AddRandomOrder)
	e.GET("/orderStatusCount/:status", web.GetSubmittedOrdersCount)

	t := &web.Template{
		Templates: template.Must(template.ParseGlob("web/index.html")),
	}

	e.Renderer = t

	// e.Logger.Fatal(e.Start(fmt.Sprintf(":8080")))
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", config.EnvAppPort())))
}
