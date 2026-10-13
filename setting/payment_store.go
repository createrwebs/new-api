package setting

import (
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var (
	// Apple App Store configuration
	AppleBundleId     = ""
	AppleKeyId        = ""
	AppleIssuerId     = ""
	ApplePrivateKey   = ""
	AppleEnvironment  = "production" // "production" or "sandbox"
	AppleAllowSandbox = false

	// Google Play configuration
	GooglePackageName             = ""
	GoogleServiceAccountJSON      = ""
	GoogleAllowTestPurchase       = false
	GooglePubSubVerificationToken = ""

	// Account Binding Hardening policy
	// If true, initial binding of new store purchases requires a cryptographically matching account token.
	StoreRequireAccountToken = true
)

func getEnvFirstNonEmpty(keys ...string) string {
	for _, k := range keys {
		if val := strings.TrimSpace(os.Getenv(k)); val != "" {
			return val
		}
	}
	return ""
}

func getEnvFirstBool(defaultVal bool, keys ...string) bool {
	for _, k := range keys {
		if val := strings.TrimSpace(os.Getenv(k)); val != "" {
			return common.GetEnvOrDefaultBool(k, defaultVal)
		}
	}
	return defaultVal
}

func init() {
	// Support both APPLE_IAP_* and APPLE_* (and APPLE_STOREKIT_*) conventions
	AppleBundleId = getEnvFirstNonEmpty("APPLE_IAP_BUNDLE_ID", "APPLE_BUNDLE_ID")
	AppleKeyId = getEnvFirstNonEmpty("APPLE_IAP_KEY_ID", "APPLE_KEY_ID", "APPLE_STOREKIT_KEY_ID")
	AppleIssuerId = getEnvFirstNonEmpty("APPLE_IAP_ISSUER_ID", "APPLE_ISSUER_ID", "APPLE_STOREKIT_ISSUER_ID")
	ApplePrivateKey = getEnvFirstNonEmpty("APPLE_IAP_PRIVATE_KEY", "APPLE_PRIVATE_KEY", "APPLE_STOREKIT_PRIVATE_KEY")

	// Read private key from file if path is specified and key content is empty
	if ApplePrivateKey == "" {
		if keyPath := getEnvFirstNonEmpty("APPLE_IAP_PRIVATE_KEY_PATH", "APPLE_PRIVATE_KEY_PATH", "APPLE_STOREKIT_PRIVATE_KEY_PATH"); keyPath != "" {
			if content, err := os.ReadFile(keyPath); err == nil {
				ApplePrivateKey = strings.TrimSpace(string(content))
			}
		}
	}

	AppleEnvironment = strings.ToLower(getEnvFirstNonEmpty("APPLE_IAP_ENVIRONMENT", "APPLE_ENVIRONMENT"))
	if AppleEnvironment == "" {
		AppleEnvironment = "production"
	}
	AppleAllowSandbox = getEnvFirstBool(false, "APPLE_IAP_ALLOW_SANDBOX", "APPLE_ALLOW_SANDBOX")

	// Support both GOOGLE_IAP_* and GOOGLE_* conventions
	GooglePackageName = getEnvFirstNonEmpty("GOOGLE_IAP_PACKAGE_NAME", "GOOGLE_PACKAGE_NAME")
	GoogleServiceAccountJSON = getEnvFirstNonEmpty("GOOGLE_IAP_SERVICE_ACCOUNT_JSON", "GOOGLE_SERVICE_ACCOUNT_JSON")

	// Read service account JSON from file if path is specified and json content is empty
	if GoogleServiceAccountJSON == "" {
		if saPath := getEnvFirstNonEmpty("GOOGLE_IAP_SERVICE_ACCOUNT_PATH", "GOOGLE_SERVICE_ACCOUNT_PATH"); saPath != "" {
			if content, err := os.ReadFile(saPath); err == nil {
				GoogleServiceAccountJSON = strings.TrimSpace(string(content))
			}
		}
	}

	GoogleAllowTestPurchase = getEnvFirstBool(false, "GOOGLE_IAP_ALLOW_TEST_PURCHASE", "GOOGLE_ALLOW_TEST_PURCHASE")
	GooglePubSubVerificationToken = getEnvFirstNonEmpty("GOOGLE_IAP_PUBSUB_VERIFICATION_TOKEN", "GOOGLE_PUBSUB_VERIFICATION_TOKEN")

	// Account binding policy (defaults to true for strict cryptographic binding)
	StoreRequireAccountToken = getEnvFirstBool(true, "STORE_REQUIRE_ACCOUNT_TOKEN", "STORE_IAP_REQUIRE_ACCOUNT_TOKEN")
}

func IsAppleStoreConfigured() bool {
	return AppleBundleId != "" && (AppleKeyId != "" || ApplePrivateKey != "")
}

func IsGooglePlayConfigured() bool {
	return GooglePackageName != "" && GoogleServiceAccountJSON != ""
}
