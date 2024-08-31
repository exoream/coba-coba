package main

import (
	"socket/controller"
	"socket/routes"
	"socket/service"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	// Initialize services
	userService, adminService, transactionService, webSocketService := service.NewService()

    // Initialize controller
    ctrl := controller.NewController(userService, adminService, transactionService, webSocketService)

	routes.RegisterRoutes(e, ctrl)

    e.Logger.Fatal(e.Start(":8080"))
}