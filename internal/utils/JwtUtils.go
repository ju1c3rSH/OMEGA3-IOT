package utils

import (
	"github.com/golang-jwt/jwt"
	"os"
	"strings"
	"time"
)

var jwtSecret = os.Getenv("JWT_SECRET")

type UserClaims struct {
	UUID     string `json:"uuid"`
	UserName string `json:"id" example:"dev_001"`
	Role     int    `json:"role"`
	jwt.StandardClaims
}

func GenerateToken(username string, userUUID string, role int) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour).Unix()
	claims := UserClaims{
		UserName: username,
		Role:     role,
		UUID:     userUUID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime,
			IssuedAt:  time.Now().Unix(),
			Issuer:    os.Getenv("OMEGA3_IOT"),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
	//这里改掉Bearer是因为浏览器header会自动加上
}

// All with bearer

func ParseToken(tokenString string) (*UserClaims, error) {
	claims := &UserClaims{}
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = tokenString[7:]
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil

}

func RefreshToken(tokenString string) (string, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	return GenerateToken(claims.UserName, claims.UUID, claims.Role)
}
