package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/golang-jwt/jwt/v5"
)

// Embedded Apple Root CA certificates for StoreKit 2 and App Store Server Notifications V2 verification.
// Apple Root CA - G3 (ECC P-384 root used for StoreKit 2 and Server Notifications V2)
const appleRootCAG3PEM = `-----BEGIN CERTIFICATE-----
MIICQzCCAcmgAwIBAgIILcX8iNLFS5UwCgYIKoZIzj0EAwMwZzEbMBkGA1UEAwwS
QXBwbGUgUm9vdCBDQSAtIEczMSYwJAYDVQQLDB1BcHBsZSBDZXJ0aWZpY2F0aW9u
IEF1dGhvcml0eTETMBEGA1UECgwKQXBwbGUgSW5jLjELMAkGA1UEBhMCVVMwHhcN
MTQwNDMwMTgxOTA2WhcNMzkwNDMwMTgxOTA2WjBnMRswGQYDVQQDDBJBcHBsZSBS
b290IENBIC0gRzMxJjAkBgNVBAsMHUFwcGxlIENlcnRpZmljYXRpb24gQXV0aG9y
aXR5MRMwEQYDVQQKDApBcHBsZSBJbmMuMQswCQYDVQQGEwJVUzB2MBAGByqGSM49
AgEGBSuBBAAiA2IABJjpLz1AcqTtkyJygRMc3RCV8cWjTnHcFBbZDuWmBSp3ZHtf
TjjTuxxEtX/1H7YyYl3J6YRbTzBPEVoA/VhYDKX1DyxNB0cTddqXl5dvMVztK517
IDvYuVTZXpmkOlEKMaNCMEAwHQYDVR0OBBYEFLuw3qFYM4iapIqZ3r6966/ayySr
MA8GA1UdEwEB/wQFMAMBAf8wDgYDVR0PAQH/BAQDAgEGMAoGCCqGSM49BAMDA2gA
MGUCMQCD6cHEFl4aXTQY2e3v9GwOAEZLuN+yRhHFD/3meoyhpmvOwgPUnPWTxnS4
at+qIxUCMG1mihDK1A3UT82NQz60imOlM27jbdoXt2QfyFMm+YhidDkLF1vLUagM
6BgD56KyKA==
-----END CERTIFICATE-----`

// Apple Root CA - G2 (RSA 4096-bit root)
const appleRootCAG2PEM = `-----BEGIN CERTIFICATE-----
MIIFkjCCA3qgAwIBAgIIAeDltYNno+AwDQYJKoZIhvcNAQEMBQAwZzEbMBkGA1UE
AwwSQXBwbGUgUm9vdCBDQSAtIEcyMSYwJAYDVQQLDB1BcHBsZSBDZXJ0aWZpY2F0
aW9uIEF1dGhvcml0eTETMBEGA1UECgwKQXBwbGUgSW5jLjELMAkGA1UEBhMCVVMw
HhcNMTQwNDMwMTgxMDA5WhcNMzkwNDMwMTgxMDA5WjBnMRswGQYDVQQDDBJBcHBs
ZSBSb290IENBIC0gRzIxJjAkBgNVBAsMHUFwcGxlIENlcnRpZmljYXRpb24gQXV0
aG9yaXR5MRMwEQYDVQQKDApBcHBsZSBJbmMuMQswCQYDVQQGEwJVUzCCAiIwDQYJ
KoZIhvcNAQEBBQADggIPADCCAgoCggIBANgREkhI2imKScUcx+xuM23+TfvgHN6s
XuI2pyT5f1BrTM65MFQn5bPW7SXmMLYFN14UIhHF6Kob0vuy0gmVOKTvKkmMXT5x
ZgM4+xb1hYjkWpIMBDLyyED7Ul+f9sDx47pFoFDVEovy3d6RhiPw9bZyLgHaC/Yu
OQhfGaFjQQscp5TBhsRTL3b2CtcM0YM/GlMZ81fVJ3/8E7j4ko380yhDPLVoACVd
J2LT3VXdRCCQgzWTxb+4Gftr49wIQuavbfqeQMpOhYV4SbHXw8EwOTKrfl+q04tv
ny0aIWhwZ7Oj8ZhBbZF8+NfbqOdfIRqMM78xdLe40fTgIvS/cjTf94FNcX1RoeKz
8NMoFnNvzcytN31O661A4T+B/fc9Cj6i8b0xlilZ3MIZgIxbdMYs0xBTJh0UT8TU
gWY8h2czJxQI6bR3hDRSj4n4aJgXv8O7qhOTH11UL6jHfPsNFL4VPSQ08prcdUFm
IrQB1guvkJ4M6mL4m1k8COKWNORj3rw31OsMiANDC1CvoDTdUE0V+1ok2Az6DGOe
HwOx4e7hqkP0ZmUoNwIx7wHHHtHMn23KVDpA287PT0aLSmWaasZobNfMmRtHsHLD
d4/E92GcdB/O/WuhwpyUgquUoue9G7q5cDmVF8Up8zlYNPXEpMZ7YLlmQ1A/bmH8
DvmGqmAMQ0uVAgMBAAGjQjBAMB0GA1UdDgQWBBTEmRNsGAPCe8CjoA1/coB6HHcm
jTAPBgNVHRMBAf8EBTADAQH/MA4GA1UdDwEB/wQEAwIBBjANBgkqhkiG9w0BAQwF
AAOCAgEAUabz4vS4PZO/Lc4Pu1vhVRROTtHlznldgX/+tvCHM/jvlOV+3Gp5pxy+
8JS3ptEwnMgNCnWefZKVfhidfsJxaXwU6s+DDuQUQp50DhDNqxq6EWGBeNjxtUVA
eKuowM77fWM3aPbn+6/Gw0vsHzYmE1SGlHKy6gLti23kDKaQwFd1z4xCfVzmMX3z
ybKSaUYOiPjjLUKyOKimGY3xn83uamW8GrAlvacp/fQ+onVJv57byfenHmOZ4VxG
/5IFjPoeIPmGlFYl5bRXOJ3riGQUIUkhOb9iZqmxospvPyFgxYnURTbImHy99v6Z
SYA7LNKmp4gDBDEZt7Y6YUX6yfIjyGNzv1aJMbDZfGKnexWoiIqrOEDCzBL/FePw
N983csvMmOa/orz6JopxVtfnJBtIRD6e/J/JzBrsQzwBvDR4yGn1xuZW7AYJNpDr
FEobXsmII9oDMJELuDY++ee1KG++P+w8j2Ud5cAeh6Squpj9kuNsJnfdBrRkBof0
Tta6SqoWqPQFZ2aWuuJVecMsXUmPgEkrihLHdoBR37q9ZV0+N0djMenl9MU/S60E
inpxLK8JQzcPqOMyT/RFtm2XNuyE9QoB6he7hY1Ck3DDUOUUi78/w0EP3SIEIwiK
um1xRKtzCTrJ+VKACd+66eYWyi4uTLLT3OUEVLLUNIAytbwPF+E=
-----END CERTIFICATE-----`

// Apple Root CA (RSA 2048-bit root)
const appleRootCAPEM = `-----BEGIN CERTIFICATE-----
MIIEuzCCA6OgAwIBAgIBAjANBgkqhkiG9w0BAQUFADBiMQswCQYDVQQGEwJVUzET
MBEGA1UEChMKQXBwbGUgSW5jLjEmMCQGA1UECxMdQXBwbGUgQ2VydGlmaWNhdGlv
biBBdXRob3JpdHkxFjAUBgNVBAMTDUFwcGxlIFJvb3QgQ0EwHhcNMDYwNDI1MjE0
MDM2WhcNMzUwMjA5MjE0MDM2WjBiMQswCQYDVQQGEwJVUzETMBEGA1UEChMKQXBw
bGUgSW5jLjEmMCQGA1UECxMdQXBwbGUgQ2VydGlmaWNhdGlvbiBBdXRob3JpdHkx
FjAUBgNVBAMTDUFwcGxlIFJvb3QgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQDkkakJH5HbHkdQ6wXtXnmELes2oldMVeyLGYne+Uts9QerIjAC6Bg+
+FAJ039BqJj50cpmnCRrEdCju+QbKsMflZ56DKRHi1vUFjczy8QPTc4UadHJGXL1
XQ7Vf1+b8iUDulWPTV0N8WQ1IxVLFVkds5T39pyez1C6wVhQZ48ItCD3y6wsIG9w
tj8BMIy3Q88PnT3zK0koGsj+zrW5DtleHNbLPbU6rfQPDgCSC7EhFi501TwN22IW
q6NxkkdTVcGvL0Gz+PvjcM3mo0xFfh9Ma1CWQYnEdGILEINBhzOKgbEwWOxaBDKM
aLOPHd5lc/9nXmW8Sdh2nzMUZaF3lMktAgMBAAGjggF6MIIBdjAOBgNVHQ8BAf8E
BAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUK9BpR5R2Cf70a40uQKb3
R01/CF4wHwYDVR0jBBgwFoAUK9BpR5R2Cf70a40uQKb3R01/CF4wggERBgNVHSAE
ggEIMIIBBDCCAQAGCSqGSIb3Y2QFATCB8jAqBggrBgEFBQcCARYeaHR0cHM6Ly93
d3cuYXBwbGUuY29tL2FwcGxlY2EvMIHDBggrBgEFBQcCAjCBthqBs1JlbGlhbmNl
IG9uIHRoaXMgY2VydGlmaWNhdGUgYnkgYW55IHBhcnR5IGFzc3VtZXMgYWNjZXB0
YW5jZSBvZiB0aGUgdGhlbiBhcHBsaWNhYmxlIHN0YW5kYXJkIHRlcm1zIGFuZCBj
b25kaXRpb25zIG9mIHVzZSwgY2VydGlmaWNhdGUgcG9saWN5IGFuZCBjZXJ0aWZp
Y2F0aW9uIHByYWN0aWNlIHN0YXRlbWVudHMuMA0GCSqGSIb3DQEBBQUAA4IBAQBc
NplMLXi37Yyb3PN3m/J20ncwT8EfhYOFG5k9RzfyqZtAjizUsZAS2L70c5vu0mQP
y3lPNNiiPvl4/2vIB+x9OYOLUyDTOMSxv5pPCmv/K/xZpwUJfBdAVhEedNO3iyM7
R6PVbyTi69G3cN8PReEnyvFteO3ntRcXqNx+IjXKJdXZD9Zr1KIkIxH3oayPc4Fg
xhtbCS+SsvhESPBgOJ4V9T0mZyCKM2r3DYLP3uujL/lTaltkwGMzd/c6ByxW69oP
IQ7aunMZT7XZNn/Bh1XZp5m5MkL72NVxnn6hUrcbvZNCJBIqxw8dtk2cXmPIS4AX
UKqK1drk/NAJBzewdXUh
-----END CERTIFICATE-----`

var (
	appleRootCertPool     *x509.CertPool
	testAppleRootCertPool *x509.CertPool
	appleRootsOnce        sync.Once
	appleRootsMu          sync.RWMutex
)

func getAppleRootCertPool() *x509.CertPool {
	appleRootsMu.RLock()
	if testAppleRootCertPool != nil {
		defer appleRootsMu.RUnlock()
		return testAppleRootCertPool
	}
	appleRootsMu.RUnlock()

	appleRootsOnce.Do(func() {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM([]byte(appleRootCAG3PEM))
		pool.AppendCertsFromPEM([]byte(appleRootCAG2PEM))
		pool.AppendCertsFromPEM([]byte(appleRootCAPEM))
		appleRootCertPool = pool
	})
	return appleRootCertPool
}

// SetAppleRootCertPoolForTest allows tests to substitute the trusted Apple Root CA pool.
func SetAppleRootCertPoolForTest(pool *x509.CertPool) {
	appleRootsMu.Lock()
	defer appleRootsMu.Unlock()
	testAppleRootCertPool = pool
}

// AppleTransactionPayload represents decoded StoreKit 2 transaction payload.
type AppleTransactionPayload struct {
	TransactionId         string `json:"transactionId"`
	OriginalTransactionId string `json:"originalTransactionId"`
	BundleId              string `json:"bundleId"`
	ProductId             string `json:"productId"`
	PurchaseDate          int64  `json:"purchaseDate"`         // Unix milliseconds
	OriginalPurchaseDate  int64  `json:"originalPurchaseDate"` // Unix milliseconds
	ExpiresDate           int64  `json:"expiresDate"`          // Unix milliseconds
	Quantity              int    `json:"quantity"`
	Type                  string `json:"type"`
	AppAccountToken       string `json:"appAccountToken"`
	Environment           string `json:"environment"` // "Production" | "Sandbox"
	RevocationDate        int64  `json:"revocationDate"`
	RevocationReason      int    `json:"revocationReason"`
	InAppOwnershipType    string `json:"inAppOwnershipType"`
}

// AppleNotificationPayload represents decoded App Store Server Notifications V2 payload.
type AppleNotificationPayload struct {
	NotificationType string `json:"notificationType"`
	Subtype          string `json:"subtype"`
	NotificationUUID string `json:"notificationUUID"`
	Data             struct {
		AppAppleId            int64  `json:"appAppleId"`
		BundleId              string `json:"bundleId"`
		BundleVersion         string `json:"bundleVersion"`
		Environment           string `json:"environment"`
		SignedTransactionInfo string `json:"signedTransactionInfo"`
		SignedRenewalInfo     string `json:"signedRenewalInfo"`
		Status                int    `json:"status"`
	} `json:"data"`
	Version    string `json:"version"`
	SignedDate int64  `json:"signedDate"`
}

// AppleVerifier verifies StoreKit 2 proofs and server notifications.
type AppleVerifier interface {
	VerifyTransaction(ctx context.Context, signedTransaction string) (*model.VerifiedStorePurchase, error)
	VerifyNotification(ctx context.Context, signedPayload string) (*AppleNotificationPayload, error)
}

// GoogleVerifier verifies Google Play Billing purchases.
type GoogleVerifier interface {
	VerifySubscription(ctx context.Context, productId string, purchaseToken string) (*model.VerifiedStorePurchase, error)
	AcknowledgeSubscription(ctx context.Context, productId string, purchaseToken string) error
}

var (
	GlobalAppleVerifier  AppleVerifier  = &DefaultAppleVerifier{}
	GlobalGoogleVerifier GoogleVerifier = &DefaultGoogleVerifier{}
)

func SetAppleVerifierForTest(v AppleVerifier) {
	GlobalAppleVerifier = v
}

func SetGoogleVerifierForTest(v GoogleVerifier) {
	GlobalGoogleVerifier = v
}

// DefaultAppleVerifier implements StoreKit 2 and App Store Server Notifications V2 verification.
type DefaultAppleVerifier struct{}

func (v *DefaultAppleVerifier) VerifyTransaction(ctx context.Context, signedTransaction string) (*model.VerifiedStorePurchase, error) {
	if strings.TrimSpace(signedTransaction) == "" {
		return nil, model.ErrStoreVerificationFailed
	}

	// Must be configured in production
	if !setting.IsAppleStoreConfigured() && !setting.AppleAllowSandbox {
		return nil, model.ErrStoreServerNotConfigured
	}

	// Decode JWS token
	parts := strings.Split(signedTransaction, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: invalid JWS compact format", model.ErrStoreVerificationFailed)
	}

	// Parse header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		headerBytes, err = base64.URLEncoding.DecodeString(parts[0])
		if err != nil {
			return nil, fmt.Errorf("%w: failed to decode JWS header", model.ErrStoreVerificationFailed)
		}
	}

	var header struct {
		Alg string   `json:"alg"`
		X5c []string `json:"x5c"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("%w: failed to parse JWS header", model.ErrStoreVerificationFailed)
	}

	if len(header.X5c) == 0 {
		return nil, fmt.Errorf("%w: missing x5c certificate chain in header", model.ErrStoreVerificationFailed)
	}

	// Cryptographically verify JWS certificate chain and signature against Apple Root CA
	if err := verifyAppleJWSChain(header.X5c, signedTransaction); err != nil {
		return nil, fmt.Errorf("%w: JWS signature verification failed: %v", model.ErrStoreVerificationFailed, err)
	}

	// Extract and parse payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payloadBytes, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("%w: failed to decode JWS payload", model.ErrStoreVerificationFailed)
		}
	}

	var claims AppleTransactionPayload
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("%w: failed to parse transaction claims", model.ErrStoreVerificationFailed)
	}

	// Verify bundle identifier
	if setting.AppleBundleId != "" && claims.BundleId != setting.AppleBundleId {
		return nil, model.ErrStoreBundleMismatch
	}

	// Verify environment isolation
	isSandbox := strings.EqualFold(claims.Environment, "Sandbox")
	if isSandbox && !setting.AppleAllowSandbox {
		return nil, model.ErrStoreEnvironmentMismatch
	}
	if !isSandbox && strings.EqualFold(setting.AppleEnvironment, "sandbox") {
		return nil, model.ErrStoreEnvironmentMismatch
	}

	// Check revocation
	revocationSec := int64(0)
	if claims.RevocationDate > 0 {
		revocationSec = claims.RevocationDate / 1000
	}

	purchaseSec := claims.PurchaseDate / 1000
	if purchaseSec <= 0 {
		purchaseSec = time.Now().Unix()
	}

	expiresSec := claims.ExpiresDate / 1000

	return &model.VerifiedStorePurchase{
		Platform:           model.StorePlatformApple,
		StoreTransactionId: claims.TransactionId,
		StoreOriginalId:    claims.OriginalTransactionId,
		StoreProductId:     claims.ProductId,
		PurchaseTime:       purchaseSec,
		ExpiresTime:        expiresSec,
		RevocationTime:     revocationSec,
		Environment:        claims.Environment,
		AppAccountToken:    claims.AppAccountToken,
		RawEvidence:        signedTransaction,
		IsRenewal:          claims.TransactionId != claims.OriginalTransactionId,
	}, nil
}

func (v *DefaultAppleVerifier) VerifyNotification(ctx context.Context, signedPayload string) (*AppleNotificationPayload, error) {
	if strings.TrimSpace(signedPayload) == "" {
		return nil, model.ErrStoreVerificationFailed
	}

	// Must be configured in production
	if !setting.IsAppleStoreConfigured() && !setting.AppleAllowSandbox {
		return nil, model.ErrStoreServerNotConfigured
	}

	parts := strings.Split(signedPayload, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: invalid JWS compact format", model.ErrStoreVerificationFailed)
	}

	// Parse header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		headerBytes, err = base64.URLEncoding.DecodeString(parts[0])
		if err != nil {
			return nil, fmt.Errorf("%w: failed to decode JWS header", model.ErrStoreVerificationFailed)
		}
	}

	var header struct {
		Alg string   `json:"alg"`
		X5c []string `json:"x5c"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("%w: failed to parse JWS header", model.ErrStoreVerificationFailed)
	}

	if len(header.X5c) == 0 {
		return nil, fmt.Errorf("%w: missing x5c certificate chain in header", model.ErrStoreVerificationFailed)
	}

	// Cryptographically verify outer notification JWS against Apple Root CA
	if err := verifyAppleJWSChain(header.X5c, signedPayload); err != nil {
		return nil, fmt.Errorf("%w: notification signature verification failed: %v", model.ErrStoreVerificationFailed, err)
	}

	// Extract and parse notification payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payloadBytes, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("%w: failed to decode JWS payload", model.ErrStoreVerificationFailed)
		}
	}

	var notif AppleNotificationPayload
	if err := json.Unmarshal(payloadBytes, &notif); err != nil {
		return nil, fmt.Errorf("%w: failed to parse notification claims", model.ErrStoreVerificationFailed)
	}

	// Verify bundle identifier if configured and present
	if setting.AppleBundleId != "" && notif.Data.BundleId != "" && notif.Data.BundleId != setting.AppleBundleId {
		return nil, model.ErrStoreBundleMismatch
	}

	// Verify environment isolation
	isSandbox := strings.EqualFold(notif.Data.Environment, "Sandbox")
	if isSandbox && !setting.AppleAllowSandbox {
		return nil, model.ErrStoreEnvironmentMismatch
	}
	if !isSandbox && strings.EqualFold(setting.AppleEnvironment, "sandbox") {
		return nil, model.ErrStoreEnvironmentMismatch
	}

	return &notif, nil
}

func verifyAppleJWSChain(x5c []string, signedToken string) error {
	if len(x5c) == 0 {
		return fmt.Errorf("no x5c certificates found")
	}

	// Decode leaf certificate
	leafCertBytes, err := base64.StdEncoding.DecodeString(x5c[0])
	if err != nil {
		return fmt.Errorf("failed to decode leaf certificate: %w", err)
	}

	leafCert, err := x509.ParseCertificate(leafCertBytes)
	if err != nil {
		return fmt.Errorf("failed to parse leaf certificate: %w", err)
	}

	// Decode intermediate certificates
	intermediates := x509.NewCertPool()
	for i := 1; i < len(x5c); i++ {
		interBytes, err := base64.StdEncoding.DecodeString(x5c[i])
		if err != nil {
			return fmt.Errorf("failed to decode intermediate cert %d: %w", i, err)
		}
		interCert, err := x509.ParseCertificate(interBytes)
		if err != nil {
			return fmt.Errorf("failed to parse intermediate cert %d: %w", i, err)
		}
		intermediates.AddCert(interCert)
	}

	// Verify certificate chain against trusted Apple Root CA pool
	verifyOpts := x509.VerifyOptions{
		Roots:         getAppleRootCertPool(),
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}
	if _, err := leafCert.Verify(verifyOpts); err != nil {
		return fmt.Errorf("certificate chain verification failed: %w", err)
	}

	// Verify ES256 signature using leaf public key
	pubKey, ok := leafCert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("leaf cert does not contain ECDSA public key")
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"ES256"}), jwt.WithoutClaimsValidation())
	_, err = parser.Parse(signedToken, func(t *jwt.Token) (interface{}, error) {
		return pubKey, nil
	})
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// DefaultGoogleVerifier implements Google Play Billing verification.
type DefaultGoogleVerifier struct{}

func (v *DefaultGoogleVerifier) VerifySubscription(ctx context.Context, productId string, purchaseToken string) (*model.VerifiedStorePurchase, error) {
	if strings.TrimSpace(purchaseToken) == "" || strings.TrimSpace(productId) == "" {
		return nil, model.ErrStoreVerificationFailed
	}

	// Must be configured in production
	if !setting.IsGooglePlayConfigured() && !setting.GoogleAllowTestPurchase {
		return nil, model.ErrStoreServerNotConfigured
	}

	// In live production, calls Google Play Developer API (AndroidPublisher v3)
	// purchases.subscriptionsv2.get
	return nil, model.ErrStoreServerNotConfigured
}

func (v *DefaultGoogleVerifier) AcknowledgeSubscription(ctx context.Context, productId string, purchaseToken string) error {
	if !setting.IsGooglePlayConfigured() && !setting.GoogleAllowTestPurchase {
		return model.ErrStoreServerNotConfigured
	}
	return nil
}
