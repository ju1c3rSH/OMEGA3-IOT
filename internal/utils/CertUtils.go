package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CertConfig 定义证书配置
type CertConfig struct {
	CertFile string
	KeyFile  string
}

// DefaultCertDir 默认证书存储目录
const DefaultCertDir = "./certs"

// EnsureCertificates 确保 TLS 证书可用
// - 如果 certFile/keyFile 配置了路径，读取现有证书
// - 如果留空，自动生成自签名证书并保存到 DefaultCertDir
func EnsureCertificates(certFile, keyFile string) (string, string, error) {
	if certFile != "" && keyFile != "" {
		// 验证文件存在
		if err := verifyCertFiles(certFile, keyFile); err != nil {
			return "", "", fmt.Errorf("certificate files invalid: %w", err)
		}
		log.Printf("[TLS] Using existing certificates: cert=%s, key=%s", certFile, keyFile)
		return certFile, keyFile, nil
	}

	// 自动生成自签名证书
	return ensureSelfSignedCert()
}

// verifyCertFiles 验证证书文件是否存在且可读
func verifyCertFiles(certFile, keyFile string) error {
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		return fmt.Errorf("certificate file not found: %s", certFile)
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return fmt.Errorf("key file not found: %s", keyFile)
	}
	return nil
}

// ensureSelfSignedCert 确保自签名证书存在，不存在则生成
func ensureSelfSignedCert() (string, string, error) {
	certPath := filepath.Join(DefaultCertDir, "server.crt")
	keyPath := filepath.Join(DefaultCertDir, "server.key")

	// 检查是否已存在
	if err := verifyCertFiles(certPath, keyPath); err == nil {
		log.Printf("[TLS] Reusing existing self-signed certificates: cert=%s, key=%s", certPath, keyPath)
		return certPath, keyPath, nil
	}

	// 生成新证书
	log.Println("[TLS] Generating new self-signed certificates...")
	certPEM, keyPEM, err := generateSelfSignedCert()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate self-signed certificate: %w", err)
	}

	// 创建目录
	if err := os.MkdirAll(DefaultCertDir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create certs directory: %w", err)
	}

	// 写入证书文件
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write certificate file: %w", err)
	}

	// 写入私钥文件（权限更严格）
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return "", "", fmt.Errorf("failed to write key file: %w", err)
	}

	log.Printf("[TLS] Self-signed certificates generated: cert=%s, key=%s", certPath, keyPath)
	return certPath, keyPath, nil
}

// generateSelfSignedCert 生成自签名 ECDSA 证书
func generateSelfSignedCert() (certPEM, keyPEM []byte, err error) {
	// 生成 ECDSA P-256 私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// 生成随机序列号
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// 证书模板
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"OMEGA3-IOT"},
			CommonName:   "OMEGA3-IOT Self-Signed",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1年有效期
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
		DNSNames: []string{"localhost"},
	}

	// 自签名：CA 证书 = 自身
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	// 创建证书
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// 编码为 PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// 编码私钥为 PEM
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})

	return certPEM, keyPEM, nil
}
