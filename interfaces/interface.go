package interfaces

import (
	"socket/model"

	"github.com/gorilla/websocket"
)

type UserService interface {
	CreateUser(user model.User) error
	GetUser(id int) (model.User, error)
}

type AdminService interface {
	CreateAdmin(admin model.Admin) error
	GetAdmin(id int) (model.Admin, error)
	GetAllAdmins() ([]model.Admin, error)
}

type TransactionService interface {
	ProcessTransaction(userID int, adminID int, price float64) (model.Transaction, string, error)
	HandleMidtransNotification(notificationPayload map[string]interface{}) error
	GetTransaction(id int) (model.Transaction, string, error)
}

type WebSocketService interface {
	HandleConnection(tokenStr string, role string, conn *websocket.Conn) error
}
