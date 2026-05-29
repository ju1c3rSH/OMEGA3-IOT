package utils

import (
	"fmt"
	"github.com/golang-jwt/jwt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	TokenTTL         = 24 * time.Hour
	JWTSecretEnvKey  = "JWT_SECRET"
)

var jwtSecret string

func init() {
	jwtSecret = os.Getenv(JWTSecretEnvKey)
	if jwtSecret == "" {
		log.Fatalf("FATAL: %s environment variable is not set. JWT signing requires a secret.", JWTSecretEnvKey)
	}
	if len(jwtSecret) < 32 {
		log.Printf("WARNING: %s is shorter than 32 characters. Consider using a longer secret for better security.", JWTSecretEnvKey)
	}
}

// GetJWTSecret returns the JWT secret (for testing purposes only)
func GetJWTSecret() string {
	if jwtSecret == "" {
		panic(fmt.Sprintf("%s not initialized", JWTSecretEnvKey))
	}
	return jwtSecret
}

type UserClaims struct {
	JTI      string `json:"jti"`
	UUID     string `json:"uuid"`
	UserName string `json:"username" example:"dev_001"`
	Role     int    `json:"role"`
	jwt.StandardClaims
}

func GenerateToken(username string, userUUID string, role int, jti string) (string, error) {
	expirationTime := time.Now().Add(TokenTTL).Unix()
	claims := UserClaims{
		JTI:      jti,
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
	newJTI := GenerateUUID().String()
	return GenerateToken(claims.UserName, claims.UUID, claims.Role, newJTI)
}
