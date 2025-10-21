package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"strings"
)

const (
	RegCodeLength     = 8
	VerifyCodeLength  = 16
	RegCodeCharset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#"
	VerifyCodeCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:,.<>?"
)

func GenerateUUID() uuid.UUID { return uuid.New() }

func ConvertHyphenIntoDash(str string) string {
	return strings.ReplaceAll(str, "-", "_")
}

func ConvertDashIntoHyphen(str string) string {
	return strings.ReplaceAll(str, "_", "-")
}

func GenerateRegCode() string {

	b := make([]byte, RegCodeLength)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(RegCodeCharset))))
		if err != nil {
			return "ErrorGne"
		}
		b[i] = RegCodeCharset[idx.Int64()]
	}
	return string(b)
}
func GenerateVerifyCode() (string, error) {
	charsetLen := big.NewInt(int64(len(VerifyCodeCharset)))
	b := make([]byte, VerifyCodeLength)

	for i := range b {
		// 生成一个 0 到 charsetLen-1 之间的随机数
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		b[i] = VerifyCodeCharset[idx.Int64()]
	}
	//TODO 加salt
	return string(b), nil
}
func HashVerifyCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return fmt.Sprintf("%x", hash)
}
