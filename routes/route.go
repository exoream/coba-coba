package routes

import (
	"socket/controller"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo, ctrl *controller.Controller) {
	// Routes for User
	e.POST("/users", ctrl.CreateUser)
    e.GET("/users/:id", ctrl.GetUser)

    // Routes for Admin
    e.POST("/admins", ctrl.CreateAdmin)
	e.GET("/admins", ctrl.GetAllAdmins)
    e.GET("/admins/:id", ctrl.GetAdmin)

	// Routes for Transaction
	e.POST("/transactions", ctrl.ProcessTransaction)
	e.GET("/transactions/:id", ctrl.GetTransaction)

	// WebSocket route
	e.GET("/chat", ctrl.HandleWebSocket)
	e.GET("/tes", ctrl.SimpleWebSocketHandler)
}