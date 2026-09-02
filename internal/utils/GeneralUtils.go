package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
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

// generateRandomString fills charset mapping from a single bulk crypto/rand.Read
// instead of per-byte rand.Int. Bulk read amortizes the getrandom(2) syscall
// (one syscall vs 8/16 syscalls) and avoids per-iteration big.Int allocations.
// See: https://github.com/golang/go/issues/33256 (buffering crypto/rand.Read)
// See: https://go.dev/src/crypto/rand/util.go (rand.Int wastes bytes and allocates)
func generateRandomString(length int, charset string) (string, error) {
	if length <= 0 || len(charset) == 0 {
		return "", fmt.Errorf("invalid length or charset")
	}
	b := make([]byte, length)
	charsetLen := len(charset)
	// For unbiased mapping when charsetLen is not a power of two, use rejection
	// sampling: discard bytes >= maxValid to avoid modulo bias. For 64-char set
	// (RegCode) maxValid==256 so no discard; for 88-char set ~31% discard rate.
	maxValid := 256 - (256 % charsetLen)
	// Over-allocate to handle discards without extra syscall in common case.
	bufSize := length * 2
	if charsetLen == 64 {
		bufSize = length
	}
	buf := make([]byte, bufSize)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	idx := 0
	for i := 0; i < length; {
		if idx >= len(buf) {
			extra := make([]byte, length)
			if _, err := rand.Read(extra); err != nil {
				return "", fmt.Errorf("failed to read random bytes: %w", err)
			}
			buf = append(buf, extra...)
		}
		v := buf[idx]
		idx++
		if int(v) >= maxValid {
			continue
		}
		b[i] = charset[int(v)%charsetLen]
		i++
	}
	return string(b), nil
}

func GenerateRegCode() string {
	s, err := generateRandomString(RegCodeLength, RegCodeCharset)
	if err != nil {
		return "ErrorGne"
	}
	return s
}
func GenerateVerifyCode() (string, error) {
	return generateRandomString(VerifyCodeLength, VerifyCodeCharset)
}
func HashVerifyCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return fmt.Sprintf("%x", hash)
}
