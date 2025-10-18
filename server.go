package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	// Создаем новый экземпляр Echo
	e := echo.New()

	// Определяем маршрут для главной страницы
	// e.GET("/", func(c echo.Context) error {
	// 	return c.String(http.StatusOK, "Hello world")
	// })
	e.GET("/user/:name", func(c echo.Context) error {
		name := c.Param("name") // Получаем параметр name из URL
		return c.String(http.StatusOK, "Привет, "+name+"!")
	})

	// Запускаем сервер на порту 8080
	e.Start(":8080")

}
