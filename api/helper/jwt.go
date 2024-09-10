package helper

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var secretKey = []byte("secret")

func CreateJWT(userID, adminID, transactionID int) (string, error) {
    // Define token claims
    claims := jwt.MapClaims{
        "user_id":       userID,
        "admin_id":      adminID,
        "transaction_id": transactionID,
        "exp":           time.Now().Add(time.Hour * 1).Unix(),
    }

    // Create token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(secretKey)
}

func VerifyJWT(tokenStr string) (*jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return secretKey, nil
    })
    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return &claims, nil
    } else {
        return nil, errors.New("invalid token")
    }
}
