package web

import (
	"html/template"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Template struct {
	Templates *template.Template
}

//implement echo interface
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

func GetHome(c echo.Context) error {

	//call orders handler
	orders := GetOrders()

	//passing in the template name (not file name)
	return c.Render(http.StatusOK, "home", orders)
}

//https://gowebexamples.com/forms/

func NewOrder(c echo.Context) error {

	//call orders handler
	AddOrder(c)

	//route back to gethome to retrieve all orders and show page
	return GetHome(c)
}
