package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const (
	MJImageAccessQueryParameter = "access"
	MJImageAccessPurpose        = "mj-image"
	DefaultMJImageAccessTTL     = 30 * time.Minute
	maxMJTaskIDLength           = 191
)

var ErrMJImageAccessInvalid = errors.New("midjourney image access is invalid")

func mjImageAccessMessage(purpose, mjID string, ownerUserId int, expiresAt int64) []byte {
	return []byte(fmt.Sprintf("%s\x00%s\x00%d\x00%d", purpose, mjID, ownerUserId, expiresAt))
}

// IssueMJImageAccessWithPurpose issues a capability token with a custom purpose string (used for testing purpose isolation).
func IssueMJImageAccessWithPurpose(purpose, mjID string, ownerUserId int, ttl time.Duration) (string, error) {
	purpose = strings.TrimSpace(purpose)
	mjID = strings.TrimSpace(mjID)
	if purpose == "" || mjID == "" || len(mjID) > maxMJTaskIDLength || ownerUserId <= 0 || common.CryptoSecret == "" {
		return "", ErrMJImageAccessInvalid
	}
	if ttl <= 0 {
		ttl = DefaultMJImageAccessTTL
	}
	expiresAt := time.Now().Add(ttl).Unix()

	mac := hmac.New(sha256.New, []byte(common.CryptoSecret))
	_, _ = mac.Write(mjImageAccessMessage(purpose, mjID, ownerUserId, expiresAt))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%d.%d.%s", purpose, expiresAt, ownerUserId, sig), nil
}

// IssueMJImageAccess creates a capability token bound to purpose ("mj-image"), resource (mjID), owner (ownerUserId), and expiration time.
func IssueMJImageAccess(mjID string, ownerUserId int, ttl time.Duration) (string, error) {
	return IssueMJImageAccessWithPurpose(MJImageAccessPurpose, mjID, ownerUserId, ttl)
}

// ParseAndValidateMJImageAccess parses the token, validates its format, purpose ("mj-image"), expiration, and HMAC signature against mjID.
// It returns the bound ownerUserId and expiresAt timestamp if cryptographically authentic and not expired.
func ParseAndValidateMJImageAccess(token, mjID string) (ownerUserId int, expiresAt int64, err error) {
	mjID = strings.TrimSpace(mjID)
	if mjID == "" || len(mjID) > maxMJTaskIDLength || common.CryptoSecret == "" {
		return 0, 0, ErrMJImageAccessInvalid
	}
	parts := strings.Split(token, ".")
	if len(parts) != 4 {
		return 0, 0, ErrMJImageAccessInvalid
	}
	if parts[0] != MJImageAccessPurpose {
		return 0, 0, ErrMJImageAccessInvalid
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return 0, 0, ErrMJImageAccessInvalid
	}
	ownerId, err := strconv.Atoi(parts[2])
	if err != nil || ownerId <= 0 {
		return 0, 0, ErrMJImageAccessInvalid
	}
	actualSig, err := base64.RawURLEncoding.Strict().DecodeString(parts[3])
	if err != nil || len(actualSig) != sha256.Size {
		return 0, 0, ErrMJImageAccessInvalid
	}

	mac := hmac.New(sha256.New, []byte(common.CryptoSecret))
	_, _ = mac.Write(mjImageAccessMessage(parts[0], mjID, ownerId, exp))
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(actualSig, expectedSig) {
		return 0, 0, ErrMJImageAccessInvalid
	}
	return ownerId, exp, nil
}

// VerifyMJImageAccess verifies the capability token's signature, purpose, expiration, and binds the caller identity to ownerUserId (or admin bypass).
func VerifyMJImageAccess(token, mjID string, callerUserId int, callerRole int) bool {
	ownerUserId, _, err := ParseAndValidateMJImageAccess(token, mjID)
	if err != nil {
		return false
	}
	// Admins and root users can access across users
	if callerRole >= common.RoleAdminUser {
		return true
	}
	// Normal users must be authenticated and their caller identity must match ownerUserId bound in the token
	if callerUserId <= 0 || callerUserId != ownerUserId {
		return false
	}
	return true
}

// BuildMJImageURL returns the proxy URL for the midjourney task image with a capability token bound to mjID and ownerUserId.
func BuildMJImageURL(mjID string, ownerUserId int, ttl time.Duration) string {
	mjID = strings.TrimSpace(mjID)
	if mjID == "" {
		return ""
	}
	baseAddress := strings.TrimRight(system_setting.ServerAddress, "/")
	if ownerUserId <= 0 {
		return baseAddress + "/mj/image/" + url.PathEscape(mjID)
	}
	token, err := IssueMJImageAccess(mjID, ownerUserId, ttl)
	if err != nil {
		return baseAddress + "/mj/image/" + url.PathEscape(mjID)
	}
	return fmt.Sprintf("%s/mj/image/%s?access=%s", baseAddress, url.PathEscape(mjID), url.QueryEscape(token))
}
