package utils

import (
	"fmt"
	"github.com/golang-jwt/jwt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	TokenTTL        = 24 * time.Hour
	JWTSecretEnvKey = "JWT_SECRET"
)

var (
	jwtSecret  string
	jwtIssuer  string
	jwtInitErr error
	jwtOnce    sync.Once
)

func initJWTSecret() {
	secret := os.Getenv(JWTSecretEnvKey)
	if secret == "" {
		jwtInitErr = fmt.Errorf("%s not set", JWTSecretEnvKey)
		log.Printf("WARNING: %s not set, JWT operations will fail until it is configured (lazy init)", JWTSecretEnvKey)
		return
	}
	if len(secret) < 32 {
		log.Printf("WARNING: %s is shorter than 32 characters. Consider using a longer secret for better security.", JWTSecretEnvKey)
	}
	jwtSecret = secret
	// Cache issuer at startup to avoid per-request os.Getenv.
	// See: https://pkg.go.dev/sync#Once (fast path ~1-2ns after first call via atomic load)
	jwtIssuer = os.Getenv("OMEGA3_IOT")
}

func init() {
	jwtOnce.Do(initJWTSecret)
}

func ensureJWTSecret() error {
	// sync.Once guarantees exactly-once init even under concurrent goroutines
	// and provides a happens-before memory barrier so subsequent reads of
	// jwtSecret are race-free. Avoids per-request os.Getenv syscall overhead.
	// See: https://pkg.go.dev/sync#Once and https://blog.sgmansfield.com/2016/01/locking-in-crypto-rand/ (env global not thread-safe)
	jwtOnce.Do(initJWTSecret)
	return jwtInitErr
}

// GetJWTSecret returns the JWT secret (for testing purposes only)
func GetJWTSecret() string {
	if err := ensureJWTSecret(); err != nil {
		panic(fmt.Sprintf("%s not initialized: %v", JWTSecretEnvKey, err))
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
	if err := ensureJWTSecret(); err != nil {
		return "", err
	}
	// Cache time.Now() once per token to avoid two syscalls and ensure
	// ExpiresAt/IssuedAt are consistent within the same second.
	now := time.Now()
	claims := UserClaims{
		JTI:      jti,
		UserName: username,
		Role:     role,
		UUID:     userUUID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: now.Add(TokenTTL).Unix(),
			IssuedAt:  now.Unix(),
			Issuer:    jwtIssuer,
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
	if err := ensureJWTSecret(); err != nil {
		return nil, err
	}
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
