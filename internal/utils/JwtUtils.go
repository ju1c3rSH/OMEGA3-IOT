package utils

import (
	"github.com/golang-jwt/jwt"
	"os"
	"strings"
	"time"
)

var jwtSecret = os.Getenv("JWT_SECRET")

// JWT认证说明：当前使用无状态JWT方案。若需支持token撤销/黑名单，可引入Redis存储。
// 参考：https://www.zhihu.com/question/12853133755/answer/2014974048233365651
type UserClaims struct {
	UUID     string `json:"uuid"`
	UserName string `json:"username" example:"dev_001"`
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
