package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestPKI creates a mock Root CA and leaf certificate signed by the Root CA,
// and configures the store verifier test pool.
func setupTestPKI(t *testing.T) (*ecdsa.PrivateKey, []byte, []byte, *x509.CertPool) {
	t.Helper()

	// 1. Root CA
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1001),
		Subject: pkix.Name{
			CommonName:   "Apple Root CA - G3 Test",
			Organization: []string{"Apple Inc."},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	rootPool := x509.NewCertPool()
	rootPool.AddCert(caCert)

	// Set test root pool
	SetAppleRootCertPoolForTest(rootPool)
	t.Cleanup(func() {
		SetAppleRootCertPoolForTest(nil)
	})

	// 2. Leaf Certificate
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1002),
		Subject: pkix.Name{
			CommonName:   "StoreKit 2 Test Leaf",
			Organization: []string{"Apple Inc."},
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	return leafKey, leafDER, caDER, rootPool
}

func createSignedJWS(t *testing.T, claims any, leafKey *ecdsa.PrivateKey, leafCertDER []byte, caCertDER []byte) string {
	t.Helper()

	header := map[string]any{
		"alg": "ES256",
		"x5c": []string{
			base64.StdEncoding.EncodeToString(leafCertDER),
			base64.StdEncoding.EncodeToString(caCertDER),
		},
	}
	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)

	payloadJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	signingInput := fmt.Sprintf("%s.%s",
		base64.RawURLEncoding.EncodeToString(headerJSON),
		base64.RawURLEncoding.EncodeToString(payloadJSON),
	)

	sig, err := jwt.SigningMethodES256.Sign(signingInput, leafKey)
	require.NoError(t, err)

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestDefaultAppleVerifier_UnconfiguredFailsSafely(t *testing.T) {
	prevBundle, prevKey, prevAllow := setting.AppleBundleId, setting.AppleKeyId, setting.AppleAllowSandbox
	setting.AppleBundleId = ""
	setting.AppleKeyId = ""
	setting.ApplePrivateKey = ""
	setting.AppleAllowSandbox = false
	t.Cleanup(func() {
		setting.AppleBundleId, setting.AppleKeyId, setting.AppleAllowSandbox = prevBundle, prevKey, prevAllow
	})

	verifier := &DefaultAppleVerifier{}
	_, err := verifier.VerifyTransaction(context.Background(), "a.b.c")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreServerNotConfigured)

	_, err = verifier.VerifyNotification(context.Background(), "a.b.c")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreServerNotConfigured)
}

func TestDefaultAppleVerifier_EmptyTokenFails(t *testing.T) {
	verifier := &DefaultAppleVerifier{}
	_, err := verifier.VerifyTransaction(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreVerificationFailed)

	_, err = verifier.VerifyNotification(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreVerificationFailed)
}

func TestDefaultAppleVerifier_MalformedJWSFails(t *testing.T) {
	prevAllow := setting.AppleAllowSandbox
	setting.AppleAllowSandbox = true
	t.Cleanup(func() { setting.AppleAllowSandbox = prevAllow })

	verifier := &DefaultAppleVerifier{}
	_, err := verifier.VerifyTransaction(context.Background(), "not-a-valid-jwt")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreVerificationFailed)

	_, err = verifier.VerifyNotification(context.Background(), "not-a-valid-jwt")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreVerificationFailed)
}

func TestDefaultAppleVerifier_MissingX5cRejected(t *testing.T) {
	prevAllow := setting.AppleAllowSandbox
	setting.AppleAllowSandbox = true
	t.Cleanup(func() { setting.AppleAllowSandbox = prevAllow })

	headerJSON, _ := json.Marshal(map[string]any{"alg": "ES256"})
	payloadJSON, _ := json.Marshal(map[string]any{"transactionId": "tx_no_x5c"})
	token := fmt.Sprintf("%s.%s.fakeSig",
		base64.RawURLEncoding.EncodeToString(headerJSON),
		base64.RawURLEncoding.EncodeToString(payloadJSON),
	)

	verifier := &DefaultAppleVerifier{}
	_, err := verifier.VerifyTransaction(context.Background(), token)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreVerificationFailed)
	assert.Contains(t, err.Error(), "missing x5c")
}

func TestDefaultAppleVerifier_SelfSignedUntrustedCertRejected(t *testing.T) {
	prevAllow := setting.AppleAllowSandbox
	setting.AppleAllowSandbox = true
	t.Cleanup(func() { setting.AppleAllowSandbox = prevAllow })

	// Create an attacker self-signed certificate NOT trusted by Apple Root CA
	attackerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	attackerTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(9999),
		Subject: pkix.Name{
			CommonName: "Attacker Fake Apple CA",
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
	}

	attackerDER, err := x509.CreateCertificate(rand.Reader, attackerTemplate, attackerTemplate, &attackerKey.PublicKey, attackerKey)
	require.NoError(t, err)

	token := createSignedJWS(t, AppleTransactionPayload{
		TransactionId:         "tx_fake_attack",
		OriginalTransactionId: "orig_fake_attack",
		BundleId:              "com.saascover.tora",
		ProductId:             "com.saascover.tora.pro",
		Environment:           "Sandbox",
	}, attackerKey, attackerDER, attackerDER)

	// Since attacker cert does not chain to Apple Root CA (or test pool), it MUST be rejected
	verifier := &DefaultAppleVerifier{}
	_, err = verifier.VerifyTransaction(context.Background(), token)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreVerificationFailed)
	assert.Contains(t, err.Error(), "certificate chain verification failed")
}

func TestDefaultAppleVerifier_TamperedSignatureRejected(t *testing.T) {
	prevAllow := setting.AppleAllowSandbox
	setting.AppleAllowSandbox = true
	t.Cleanup(func() { setting.AppleAllowSandbox = prevAllow })

	leafKey, leafDER, caDER, _ := setupTestPKI(t)

	validJWS := createSignedJWS(t, AppleTransactionPayload{
		TransactionId:         "tx_valid",
		OriginalTransactionId: "orig_valid",
		BundleId:              "com.saascover.tora",
		ProductId:             "com.saascover.tora.pro",
		Environment:           "Sandbox",
	}, leafKey, leafDER, caDER)

	// Tamper with the signature portion
	tamperedJWS := validJWS[:len(validJWS)-4] + "AAAA"

	verifier := &DefaultAppleVerifier{}
	_, err := verifier.VerifyTransaction(context.Background(), tamperedJWS)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreVerificationFailed)
	assert.Contains(t, err.Error(), "signature verification failed")
}

func TestDefaultAppleVerifier_BundleMismatchRejected(t *testing.T) {
	prevBundle, prevAllow := setting.AppleBundleId, setting.AppleAllowSandbox
	setting.AppleBundleId = "com.saascover.tora"
	setting.AppleAllowSandbox = true
	t.Cleanup(func() {
		setting.AppleBundleId, setting.AppleAllowSandbox = prevBundle, prevAllow
	})

	leafKey, leafDER, caDER, _ := setupTestPKI(t)

	jWS := createSignedJWS(t, AppleTransactionPayload{
		TransactionId:         "tx_1",
		OriginalTransactionId: "orig_1",
		BundleId:              "com.wrong.bundle",
		ProductId:             "com.saascover.tora.pro",
		PurchaseDate:          time.Now().UnixMilli(),
		ExpiresDate:           time.Now().Add(30 * 24 * time.Hour).UnixMilli(),
		Environment:           "Sandbox",
	}, leafKey, leafDER, caDER)

	verifier := &DefaultAppleVerifier{}
	_, err := verifier.VerifyTransaction(context.Background(), jWS)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreBundleMismatch)
}

func TestDefaultAppleVerifier_SandboxRejectedInProduction(t *testing.T) {
	prevBundle, prevAllow, prevEnv := setting.AppleBundleId, setting.AppleAllowSandbox, setting.AppleEnvironment
	setting.AppleBundleId = "com.saascover.tora"
	setting.AppleAllowSandbox = false
	setting.AppleEnvironment = "production"
	setting.AppleKeyId = "test-key"
	t.Cleanup(func() {
		setting.AppleBundleId, setting.AppleAllowSandbox, setting.AppleEnvironment = prevBundle, prevAllow, prevEnv
	})

	leafKey, leafDER, caDER, _ := setupTestPKI(t)

	jWS := createSignedJWS(t, AppleTransactionPayload{
		TransactionId:         "tx_sandbox",
		OriginalTransactionId: "orig_sandbox",
		BundleId:              "com.saascover.tora",
		ProductId:             "com.saascover.tora.pro",
		PurchaseDate:          time.Now().UnixMilli(),
		ExpiresDate:           time.Now().Add(30 * 24 * time.Hour).UnixMilli(),
		Environment:           "Sandbox", // Sandbox proof
	}, leafKey, leafDER, caDER)

	verifier := &DefaultAppleVerifier{}
	_, err := verifier.VerifyTransaction(context.Background(), jWS)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreEnvironmentMismatch)
}

func TestDefaultAppleVerifier_ValidSandboxWhenAllowed(t *testing.T) {
	prevBundle, prevAllow, prevEnv := setting.AppleBundleId, setting.AppleAllowSandbox, setting.AppleEnvironment
	setting.AppleBundleId = "com.saascover.tora"
	setting.AppleAllowSandbox = true
	setting.AppleEnvironment = "sandbox"
	t.Cleanup(func() {
		setting.AppleBundleId, setting.AppleAllowSandbox, setting.AppleEnvironment = prevBundle, prevAllow, prevEnv
	})

	leafKey, leafDER, caDER, _ := setupTestPKI(t)

	nowMs := time.Now().UnixMilli()
	expMs := nowMs + 30*86400*1000

	jWS := createSignedJWS(t, AppleTransactionPayload{
		TransactionId:         "tx_valid_sandbox",
		OriginalTransactionId: "orig_valid_sandbox",
		BundleId:              "com.saascover.tora",
		ProductId:             "com.saascover.tora.pro",
		PurchaseDate:          nowMs,
		ExpiresDate:           expMs,
		Environment:           "Sandbox",
		AppAccountToken:       "uuid-1234",
	}, leafKey, leafDER, caDER)

	verifier := &DefaultAppleVerifier{}
	verified, err := verifier.VerifyTransaction(context.Background(), jWS)
	require.NoError(t, err)
	assert.Equal(t, model.StorePlatformApple, verified.Platform)
	assert.Equal(t, "tx_valid_sandbox", verified.StoreTransactionId)
	assert.Equal(t, "orig_valid_sandbox", verified.StoreOriginalId)
	assert.Equal(t, "com.saascover.tora.pro", verified.StoreProductId)
	assert.Equal(t, nowMs/1000, verified.PurchaseTime)
	assert.Equal(t, expMs/1000, verified.ExpiresTime)
	assert.Equal(t, "uuid-1234", verified.AppAccountToken)
}

func TestDefaultAppleVerifier_NotificationVerifiedSuccessfully(t *testing.T) {
	prevBundle, prevAllow, prevEnv := setting.AppleBundleId, setting.AppleAllowSandbox, setting.AppleEnvironment
	setting.AppleBundleId = "com.saascover.tora"
	setting.AppleAllowSandbox = true
	setting.AppleEnvironment = "sandbox"
	t.Cleanup(func() {
		setting.AppleBundleId, setting.AppleAllowSandbox, setting.AppleEnvironment = prevBundle, prevAllow, prevEnv
	})

	leafKey, leafDER, caDER, _ := setupTestPKI(t)

	notifPayload := AppleNotificationPayload{
		NotificationType: "DID_RENEW",
		Subtype:          "BILLING_RECOVERY",
		NotificationUUID: "notif-uuid-12345",
	}
	notifPayload.Data.BundleId = "com.saascover.tora"
	notifPayload.Data.Environment = "Sandbox"
	notifPayload.Data.SignedTransactionInfo = "mock-inner-jws"

	jws := createSignedJWS(t, notifPayload, leafKey, leafDER, caDER)

	verifier := &DefaultAppleVerifier{}
	notif, err := verifier.VerifyNotification(context.Background(), jws)
	require.NoError(t, err)
	assert.Equal(t, "DID_RENEW", notif.NotificationType)
	assert.Equal(t, "BILLING_RECOVERY", notif.Subtype)
	assert.Equal(t, "com.saascover.tora", notif.Data.BundleId)
	assert.Equal(t, "mock-inner-jws", notif.Data.SignedTransactionInfo)
}

func TestDefaultGoogleVerifier_UnconfiguredFailsSafely(t *testing.T) {
	prevPkg, prevSA, prevTest := setting.GooglePackageName, setting.GoogleServiceAccountJSON, setting.GoogleAllowTestPurchase
	setting.GooglePackageName = ""
	setting.GoogleServiceAccountJSON = ""
	setting.GoogleAllowTestPurchase = false
	t.Cleanup(func() {
		setting.GooglePackageName, setting.GoogleServiceAccountJSON, setting.GoogleAllowTestPurchase = prevPkg, prevSA, prevTest
	})

	verifier := &DefaultGoogleVerifier{}
	_, err := verifier.VerifySubscription(context.Background(), "product_1", "token_1")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreServerNotConfigured)

	err = verifier.AcknowledgeSubscription(context.Background(), "product_1", "token_1")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreServerNotConfigured)
}

func TestDefaultGoogleVerifier_EmptyParams(t *testing.T) {
	verifier := &DefaultGoogleVerifier{}
	_, err := verifier.VerifySubscription(context.Background(), "", "token_1")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreVerificationFailed)

	_, err = verifier.VerifySubscription(context.Background(), "prod_1", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrStoreVerificationFailed)
}
