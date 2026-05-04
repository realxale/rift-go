package auth

import (
	"backend/pkg/config"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type JWTAuthRequest struct {
	JWT string `json:"jwt"`
}
type JAR = JWTAuthRequest
type UserData struct {
	Username string `json:"username"`
}
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}
type RegRequest struct {
	Username string `json:"username" binding:"required,min=5,max=20"`
	Password string `json:"password" binding:"required,min=5,max=20"`
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password),
		bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

var jwtSecret = []byte(config.GetEnv("JWT_SECRET", "dasdasdwefafdsaefafdsaf"))

func GenerateJWT(username string) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
func ParseJWT(jwt_token string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(jwt_token, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// func checkPermissiobs(req UserData) (bool,error) { res,err := authJWT(req) if err != nil {	return false,err } if res.Username != req.Username { return false,nil } return true,nil
// }
