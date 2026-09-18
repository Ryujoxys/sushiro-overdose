package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ryujoxys/sushiro-overdose/internal/core"
)

// CertDirPath 返回本代理 CA 证书的存放目录（~/.sushiro-proxy）。
// CA 证书与私钥都放这里，需要用户把 ca.crt 装进系统/浏览器信任库后，MITM 签发的叶子证书才会被客户端接受。
func CertDirPath() string {
	return filepath.Join(core.DataHomeDir(), ".sushiro-proxy")
}

// LoadOrGenerateCA 优先复用已存在且仍在有效期内的本地 CA；否则新生成一对 CA 证书/私钥并落盘。
// MITM 要求每次启动都用同一套 CA，否则之前签发的叶子证书会因换 CA 而全部失效。
// 复用判定链：能读到文件 → 能组 tls.Certificate → 证书未过期 → 私钥可解析；任一环失败都落到「重新生成」。
// 新 CA 有效期 10 年（NotAfter=+10y），NotBefore 回退 1 小时避免本机时钟略快导致「证书尚未生效」。
func LoadOrGenerateCA() (tls.Certificate, *rsa.PrivateKey, error) {
	dir := CertDirPath()
	certPath := filepath.Join(dir, "ca.crt")
	keyPath := filepath.Join(dir, "ca.key")

	// Try to load existing
	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err == nil {
			block, _ := pem.Decode(certPEM)
			if block != nil {
				parsed, err := x509.ParseCertificate(block.Bytes)
				// 仅当 CA 证书仍在有效期内（Now < NotAfter）才复用，过期则重新生成。
				if err == nil && time.Now().Before(parsed.NotAfter) {
					keyBlock, _ := pem.Decode(keyPEM)
					if keyBlock != nil {
						key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
						if err == nil {
							return cert, key, nil
						}
					}
				}
			}
		}
	}

	// Generate new CA
	// 目录权限 0o700：里面是能签任意域名的 CA 私钥，必须只属主可读写。
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("create cert dir: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("generate RSA key: %w", err)
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: "Sushiro Proxy CA"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		// CA 只允许用来签证书（CertSign/CRLSign），限定用途缩小私钥泄漏后的影响面。
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// 自签 CA：template 同时作签发者与被签发者。
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("create CA cert: %w", err)
	}

	certPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	// 证书可读（644，便于安装到信任库），私钥严格 600。
	if err := os.WriteFile(certPath, certPEMBytes, 0o644); err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("write CA cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEMBytes, 0o600); err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("write CA key: %w", err)
	}

	cert, err := tls.X509KeyPair(certPEMBytes, keyPEMBytes)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("load generated cert: %w", err)
	}

	fmt.Println("CA证书已生成:", certPath)
	return cert, key, nil
}

// loadLocalCACertificate 从磁盘读取并解析本代理 CA 证书（不含私钥），供指纹计算等只读用途。
func loadLocalCACertificate() (*x509.Certificate, error) {
	certPath := filepath.Join(CertDirPath(), "ca.crt")
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("decode CA certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// LocalCACertSHA1Thumbprint 返回本代理 CA 证书的 SHA1 指纹（大写十六进制）。
// 用途：安装到系统信任库后，用它核对「装进去的证书确实是我们生成的那个」，防中间人替换。
func LocalCACertSHA1Thumbprint() (string, error) {
	cert, err := loadLocalCACertificate()
	if err != nil {
		return "", err
	}
	return certSHA1Thumbprint(cert), nil
}

// certSHA1Thumbprint 计算证书 DER 原文的 SHA1 指纹。指纹匹配即代表证书字节完全一致。
func certSHA1Thumbprint(cert *x509.Certificate) string {
	sum := sha1.Sum(cert.Raw)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// generateHostCert 用本地 CA 即时签发一张针对 host 的叶子证书，供 MITM 握手使用。
// 与 CA 不同，叶子证书有效期仅 24 小时（抓包是临时会话，无需长期有效），且每次 CONNECT 都重新签发。
// NotBefore 回退 1 小时容忍时钟偏差；KeyUsage 限定为数字签名+密钥加密、扩展用途限定 ServerAuth。
// host 为 IP 时填 IPAddresses，否则填 DNSNames，保证客户端做 SAN 校验时能匹配。
// 返回的 tls.Certificate 携带叶子+CA 两条证书，让客户端可向上验证到（已信任的）CA。
func generateHostCert(caTLSCert tls.Certificate, caKey *rsa.PrivateKey, host string) (tls.Certificate, error) {
	caCert, err := x509.ParseCertificate(caTLSCert.Certificate[0])
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse CA cert: %w", err)
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	// 去端口后判断是 IP 还是域名，分别填 IPAddresses / DNSNames，二者只能按真实类型填，否则 SAN 校验失败。
	h := host
	if ip, _, err := net.SplitHostPort(host); err == nil {
		h = ip
	}
	if ip := net.ParseIP(h); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{h}
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate leaf key: %w", err)
	}

	// 用 CA 私钥签发叶子证书：被签者是叶子公钥，签发者是 CA。
	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create leaf cert: %w", err)
	}

	return tls.Certificate{
		// 叶子在前、CA 在后，构成证书链，TLS 握手时一并下发。
		Certificate: [][]byte{certDER, caCert.Raw},
		PrivateKey:  leafKey,
	}, nil
}
