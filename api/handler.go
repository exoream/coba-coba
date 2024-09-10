package handler

import (
	"net/http"
	"socket/controller"
	"socket/routes"
	"socket/service"

	"github.com/labstack/echo/v4"
)

func Handler(w http.ResponseWriter, r *http.Request) {
  e := echo.New()

  // Initialize services
  userService, adminService, transactionService, webSocketService := service.NewService()

  // Initialize controller
  ctrl := controller.NewController(userService, adminService, transactionService, webSocketService)

  routes.RegisterRoutes(e, ctrl)

  // Start server (this line won't be executed on Vercel)
  e.ServeHTTP(w, r)
}