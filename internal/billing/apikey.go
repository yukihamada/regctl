package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	prefixLive = "rk_live_"
	prefixTest = "rk_test_"
	hmacHexLen = 64 // SHA-256 produces 32 bytes = 64 hex chars
)

// GenerateAPIKey creates an API key in the format rk_live_<customer_id>_<hmac_hex>.
func GenerateAPIKey(customerID, signingSecret string, live bool) string {
	prefix := prefixTest
	if live {
		prefix = prefixLive
	}
	payload := prefix + customerID
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s%s_%s", prefix, customerID, sig)
}

// ValidateAPIKey verifies the HMAC signature and extracts the customer ID.
func ValidateAPIKey(apiKey, signingSecret string) (string, error) {
	if len(apiKey) < len(prefixLive)+hmacHexLen+2 { // prefix + at least 1 char id + _ + hmac
		return "", errors.New("invalid API key format")
	}

	var prefix string
	switch {
	case strings.HasPrefix(apiKey, prefixLive):
		prefix = prefixLive
	case strings.HasPrefix(apiKey, prefixTest):
		prefix = prefixTest
	default:
		return "", errors.New("invalid API key prefix")
	}

	// The key is: prefix + customerID + "_" + hmac(64 chars)
	// Customer ID may contain underscores, so take the last 64 chars as HMAC.
	body := apiKey[len(prefix):]
	if len(body) < hmacHexLen+2 { // at least 1 char id + _ + hmac
		return "", errors.New("invalid API key format")
	}

	providedSig := body[len(body)-hmacHexLen:]
	customerID := body[:len(body)-hmacHexLen-1] // strip trailing "_" and hmac

	if customerID == "" {
		return "", errors.New("invalid API key: empty customer ID")
	}

	// Recompute HMAC
	payload := prefix + customerID
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return "", errors.New("invalid API key signature")
	}

	return customerID, nil
}

// IsBillingKey returns true if the key has a billing API key prefix.
func IsBillingKey(key string) bool {
	return strings.HasPrefix(key, prefixLive) || strings.HasPrefix(key, prefixTest)
}
