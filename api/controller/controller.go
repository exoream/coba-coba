package controller

import (
	"net/http"
	"socket/api/interfaces"
	"socket/api/model"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type Controller struct {
    UserService  interfaces.UserService
    AdminService interfaces.AdminService
	TransactionService interfaces.TransactionService
	WebSocketService interfaces.WebSocketService
}

func NewController(userService interfaces.UserService, adminService interfaces.AdminService, transactionService interfaces.TransactionService, wsService interfaces.WebSocketService) *Controller {
    return &Controller{
        UserService:  userService,
        AdminService: adminService,
		TransactionService: transactionService,
		WebSocketService:   wsService,
    }
}

func (c *Controller) CreateUser(ctx echo.Context) error {
    var user model.User
    if err := ctx.Bind(&user); err != nil {
        return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
    }

    if err := c.UserService.CreateUser(user); err != nil {
        return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }

    return ctx.JSON(http.StatusCreated, user)
}

func (c *Controller) GetUser(ctx echo.Context) error {
    idStr := ctx.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
    }

    user, err := c.UserService.GetUser(id)
    if err != nil {
        return ctx.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
    }

    return ctx.JSON(http.StatusOK, user)
}

// Admin Handlers
func (c *Controller) CreateAdmin(ctx echo.Context) error {
    var admin model.Admin
    if err := ctx.Bind(&admin); err != nil {
        return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
    }

    if err := c.AdminService.CreateAdmin(admin); err != nil {
        return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }

    return ctx.JSON(http.StatusCreated, admin)
}

func (c *Controller) GetAdmin(ctx echo.Context) error {
    idStr := ctx.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
    }

    admin, err := c.AdminService.GetAdmin(id)
    if err != nil {
        return ctx.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
    }

    return ctx.JSON(http.StatusOK, admin)
}

func (c *Controller) GetAllAdmins(ctx echo.Context) error {
	admins, err := c.AdminService.GetAllAdmins()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, admins)
}

func (c *Controller) ProcessTransaction(ctx echo.Context) error {
	var transaction model.Transaction
	if err := ctx.Bind(&transaction); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	transaction, token, err := c.TransactionService.ProcessTransaction(transaction.UserID, transaction.AdminID, transaction.Price)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusCreated, map[string]interface{}{
		"transaction": transaction,
		"token":       token,
	})
}

func (c *Controller) MidtransNotification(ctx echo.Context) error {
    var notificationPayload map[string]interface{}

    if err := ctx.Bind(&notificationPayload); err != nil {
        return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
    }

    // Handle notification
    err := c.TransactionService.HandleMidtransNotification(notificationPayload)
    if err != nil {
        return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }

    return ctx.JSON(http.StatusOK, map[string]string{"status": "success"})
}

func (c *Controller) GetTransaction(ctx echo.Context) error {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	transaction, token, err := c.TransactionService.GetTransaction(id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	response := map[string]interface{}{
		"transaction": transaction,
	}

	if token != "" {
		response["token"] = token
	}

	return ctx.JSON(http.StatusOK, response)
}

func (c *Controller) HandleWebSocket(ctx echo.Context) error {
	token := ctx.QueryParam("token")
    role := ctx.QueryParam("role")

    if token == "" || role == "" {
        return ctx.String(http.StatusUnauthorized, "Missing authorization token or role")
    }

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	
    conn, err := upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
    if err != nil {
        return ctx.String(http.StatusInternalServerError, "Failed to upgrade to WebSocket")
    }

    err = c.WebSocketService.HandleConnection(token, role, conn)
    if err != nil {
        conn.Close()
        return ctx.String(http.StatusInternalServerError, err.Error())
    }

    return nil
}


func (c *Controller) SimpleWebSocketHandler(ctx echo.Context) error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "Failed to upgrade to WebSocket")
	}
	defer conn.Close()

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return nil
		}
		if err := conn.WriteMessage(msgType, msg); err != nil {
			return err
		}
	}
}
