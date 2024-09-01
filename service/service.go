package service

import (
	"errors"
	"fmt"
	"socket/interfaces"
	"socket/model"
	"strconv"
	"sync"
	"time"

	"socket/helper"

	"github.com/gorilla/websocket"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
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
func (s *service) ProcessTransaction(userID int, adminID int, price float64) (model.Transaction, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, transaction := range s.transactions {
		if transaction.UserID == userID || transaction.AdminID == adminID {
			if transaction.Status == "success" && time.Now().Before(transaction.ExpiresAt) {
				return model.Transaction{}, "", errors.New("user or admin already has an active session")
			}
		}
	}

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
		Price:     price,
		Status:    "pending",
		ExpiresAt: expiration,
	}
	s.transactions[transactionID] = transaction

	// token, err := helper.CreateJWT(userID, adminID, transactionID)
    // if err != nil {
    //     return model.Transaction{}, "", err
    // }

	// Setup Midtrans Snap Client
	snapClient := snap.Client{}
	snapClient.New("SB-Mid-server-YCb-jBlX8BE6NZWIsQvW7hTA", midtrans.Sandbox)

	chargeReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  fmt.Sprintf("%d", transactionID),
			GrossAmt: int64(price),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			Email: fmt.Sprintf("user%d@example.com", userID),
		},
		EnabledPayments: snap.AllSnapPaymentType,
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
	}

	// Request to Midtrans Snap API
	snapResponse, err := snapClient.CreateTransaction(chargeReq)
	if err != nil {
		return model.Transaction{}, "", err
	}

	if snapResponse.RedirectURL != "" {
		transaction.Status = "pending_payment"
		s.transactions[transactionID] = transaction
		return transaction, snapResponse.RedirectURL, nil
	} else {
		return model.Transaction{}, "", errors.New("failed to create payment")
	}
}

func (s *service) HandleMidtransNotification(notificationPayload map[string]interface{}) error {
    orderID, ok := notificationPayload["order_id"].(string)
    if !ok || orderID == "" {
        return errors.New("missing or invalid order_id")
    }

    // Extract transactionStatus safely
    transactionStatus, ok := notificationPayload["transaction_status"].(string)
    if !ok || transactionStatus == "" {
        return errors.New("missing or invalid transaction_status")
    }

    // Extract fraudStatus safely
    fraudStatus, ok := notificationPayload["fraud_status"].(string)
    if !ok || fraudStatus == "" {
        return errors.New("missing or invalid fraud_status")
    }

    // Parse orderID to get transaction ID
    transactionID, err := strconv.Atoi(orderID)
    if err != nil {
        return err
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    // Get the transaction from the map
    transaction, exists := s.transactions[transactionID]
    if !exists {
        return errors.New("transaction not found")
    }

    // Update transaction status based on Midtrans notification
    switch transactionStatus {
    case "capture":
        if fraudStatus == "accept" {
            transaction.Status = "success"
        } else {
            transaction.Status = "fraud"
        }
    case "settlement":
        transaction.Status = "success" // Payment is complete
    case "deny", "cancel", "expire":
        transaction.Status = "failed"
    case "pending":
        transaction.Status = "pending_payment"
    default:
        return errors.New("unknown transaction status")
    }

    // Save the updated transaction
    s.transactions[transactionID] = transaction
    return nil
}

// GetTransaction implements interfaces.TransactionService.
func (s *service) GetTransaction(id int) (model.Transaction, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	transaction, exists := s.transactions[id]
	if !exists {
		return model.Transaction{}, "", errors.New("transaction not found")
	}

	var token string
	if transaction.Status == "success" {
		// Create JWT token only if transaction is successful
		var err error
		token, err = helper.CreateJWT(transaction.UserID, transaction.AdminID, id)
		if err != nil {
			return model.Transaction{}, "", err
		}
	}
	return transaction, token, nil
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

	// Check if the user or admin is already in an active session
	if role == "admin" {
		if _, exists := s.adminSessions[adminID]; exists {
			conn.Close()
			return errors.New("admin already in an active session")
		}
	} else {
		if _, exists := s.userSessions[userID]; exists {
			conn.Close()
			return errors.New("user already in an active session")
		}
	}

	
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