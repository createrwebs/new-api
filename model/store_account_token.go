package model

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ToraAccountNamespaceUUID is the canonical, fixed RFC 4122 namespace UUID
// used for deriving deterministic, privacy-preserving store account tokens.
var ToraAccountNamespaceUUID = uuid.MustParse("e0f4d3b2-7a89-4c1e-9f3a-2b5d8e7c1042")

// NormalizeStoreEnvironment maps raw store environment names to canonical "production" or "sandbox".
func NormalizeStoreEnvironment(env string) string {
	clean := strings.ToLower(strings.TrimSpace(env))
	switch clean {
	case "sandbox", "test", "xcode":
		return "sandbox"
	default:
		return "production"
	}
}

// DeriveStoreAccountToken computes a deterministic, privacy-safe RFC 4122 UUIDv5
// from the user ID and environment.
//
// This token:
// 1. Fully satisfies Apple StoreKit 2 appAccountToken UUID string format requirements.
// 2. Fits within Google Play Billing obfuscatedAccountId 64-character constraints.
// 3. Leaks NO cleartext user ID, username, email, or PII to Apple or Google.
// 4. Enforces environment isolation between sandbox and production.
func DeriveStoreAccountToken(userId int, env string) string {
	normEnv := NormalizeStoreEnvironment(env)
	name := fmt.Sprintf("tora:%s:user:%d", normEnv, userId)
	return uuid.NewSHA1(ToraAccountNamespaceUUID, []byte(name)).String()
}

// ValidateStoreAccountToken verifies whether a provided store token matches
// the expected token for the given user ID and environment.
func ValidateStoreAccountToken(token string, userId int, env string) bool {
	if token == "" || userId <= 0 {
		return false
	}
	expected := DeriveStoreAccountToken(userId, env)
	return strings.EqualFold(strings.TrimSpace(token), expected)
}
