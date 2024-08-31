package service

import (
	"errors"
	"socket/interfaces"
	"socket/model"
	"sync"
	"time"

	"socket/helper"

	"github.com/gorilla/websocket"
)


var sessionsLock = sync.Mutex{}

type service struct {
	users                map[int]model.User
	admins               map[int]model.Admin
	transactions         map[int]model.Transaction
	userSessions       map[int]*websocket.Conn
	adminSessions      map[int]*websocket.Conn
	mu                   sync.Mutex
	transactionCounter   int
}

func NewService() (interfaces.UserService, interfaces.AdminService, interfaces.TransactionService, interfaces.WebSocketService) {
	s := &service{
		users:                make(map[int]model.User),
		admins:               make(map[int]model.Admin),
		transactions:         make(map[int]model.Transaction),
		userSessions:       make(map[int]*websocket.Conn),
		adminSessions:      make(map[int]*websocket.Conn),
	}

	return s, s, s, s
}

// CreateUser implements interfaces.UserService.
func (s *service) CreateUser(user model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; exists {
		return errors.New("user already exists")
	}
	s.users[user.ID] = user
	return nil
}

// GetUser implements interfaces.UserService.
func (s *service) GetUser(id int) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return model.User{}, errors.New("user not found")
	}
	return user, nil
}

// CreateAdmin implements interfaces.AdminService.
func (s *service) CreateAdmin(admin model.Admin) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.admins[admin.ID]; exists {
		return errors.New("admin already exists")
	}

	s.admins[admin.ID] = admin

	return nil
}

// GetAdmin implements interfaces.AdminService.
func (s *service) GetAdmin(id int) (model.Admin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	admin, exists := s.admins[id]
	if !exists {
		return model.Admin{}, errors.New("admin not found")
	}
	return admin, nil
}

func (s *service) GetAllAdmins() ([]model.Admin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	admins := make([]model.Admin, 0, len(s.admins))
	for _, admin := range s.admins {
		admins = append(admins, admin)
	}
	return admins, nil
}

// ProcessTransaction implements interfaces.TransactionService.
func (s *service) ProcessTransaction(userID int, adminID int) (model.Transaction, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify user and admin
	if _, exists := s.users[userID]; !exists {
		return model.Transaction{}, "", errors.New("user not found")
	}

	if _, exists := s.admins[adminID]; !exists {
		return model.Transaction{}, "", errors.New("admin not found")
	}

	s.transactionCounter++
	transactionID := s.transactionCounter

	expiration := time.Now().Add(10 * time.Minute)

	transaction := model.Transaction{
		ID:        transactionID,
		UserID:    userID,
		AdminID:   adminID,
		ExpiresAt: expiration,
	}
	s.transactions[transactionID] = transaction

	token, err := helper.CreateJWT(userID, adminID, transactionID)
    if err != nil {
        return model.Transaction{}, "", err
    }


	return transaction, token, nil
}

// GetTransaction implements interfaces.TransactionService.
func (s *service) GetTransaction(id int) (model.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	transaction, exists := s.transactions[id]
	if !exists {
		return model.Transaction{}, errors.New("transaction not found")
	}
	return transaction, nil
}

// HandleConnection implements interfaces.WebSocketService.
func (s *service) HandleConnection(tokenStr string, role string, conn *websocket.Conn) error {
	s.mu.Lock()
    defer s.mu.Unlock()

    claims, err := helper.VerifyJWT(tokenStr)
    if err != nil {
        conn.Close()
        return errors.New("invalid or expired token")
    }

    userID := int((*claims)["user_id"].(float64))
    adminID := int((*claims)["admin_id"].(float64))
	transactionID := int((*claims)["transaction_id"].(float64))

    transaction, exists := s.transactions[transactionID]
    if !exists || time.Now().After(transaction.ExpiresAt) {
        conn.Close()
        return errors.New("transaction expired or not found")
    }

    if role == "admin" {
        s.adminSessions[adminID] = conn
    } else {
        s.userSessions[userID] = conn
    }
	
    go s.handleMessages(userID, adminID, role, conn)
    return nil
}

func (s *service) handleMessages(userID, adminID int, role string, conn *websocket.Conn) {
	expiration := time.Now().Add(10 * time.Minute)
	for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            conn.Close()
            break
        }

		if time.Now().After(expiration) {
            conn.Close()
            break
        }

        if role == "user" {
            sessionsLock.Lock()
            if adminConn, ok := s.adminSessions[adminID]; ok {
                adminConn.WriteMessage(websocket.TextMessage, msg)
            }
            sessionsLock.Unlock()
        } else {
            sessionsLock.Lock()
            if userConn, ok := s.userSessions[userID]; ok {
                userConn.WriteMessage(websocket.TextMessage, msg)
            }
            sessionsLock.Unlock()
        }
    }

	s.mu.Lock()
    if role == "admin" {
        delete(s.adminSessions, adminID)
    } else {
        delete(s.userSessions, userID)
    }
    s.mu.Unlock()
}