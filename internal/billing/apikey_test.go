package billing

import (
	"strings"
	"testing"
)

const testSecret = "test-signing-secret-32bytes-long!"

func TestGenerateAndValidate(t *testing.T) {
	customerID := "cus_abc123"
	key := GenerateAPIKey(customerID, testSecret, true)

	if !strings.HasPrefix(key, prefixLive) {
		t.Fatalf("expected prefix %s, got %s", prefixLive, key[:8])
	}

	got, err := ValidateAPIKey(key, testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != customerID {
		t.Fatalf("expected %s, got %s", customerID, got)
	}
}

func TestGenerateTestKey(t *testing.T) {
	key := GenerateAPIKey("cus_test", testSecret, false)
	if !strings.HasPrefix(key, prefixTest) {
		t.Fatalf("expected test prefix, got %s", key)
	}
	got, err := ValidateAPIKey(key, testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "cus_test" {
		t.Fatalf("expected cus_test, got %s", got)
	}
}

func TestValidateWrongSecret(t *testing.T) {
	key := GenerateAPIKey("cus_abc123", testSecret, true)
	_, err := ValidateAPIKey(key, "wrong-secret-xxxxxxxxxxxxxxxxxxxxx")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestValidateTamperedKey(t *testing.T) {
	key := GenerateAPIKey("cus_abc123", testSecret, true)
	// Tamper with the last char
	tampered := key[:len(key)-1] + "0"
	_, err := ValidateAPIKey(tampered, testSecret)
	if err == nil {
		t.Fatal("expected error for tampered key")
	}
}

func TestValidateInvalidFormat(t *testing.T) {
	tests := []string{
		"",
		"rk_live_",
		"rk_live_x",
		"invalid_key",
		"rk_live__" + strings.Repeat("a", 64), // empty customer ID
	}
	for _, key := range tests {
		_, err := ValidateAPIKey(key, testSecret)
		if err == nil {
			t.Errorf("expected error for key %q", key)
		}
	}
}

func TestIsBillingKey(t *testing.T) {
	if !IsBillingKey("rk_live_cus_abc_xxxx") {
		t.Error("expected true for rk_live_ prefix")
	}
	if !IsBillingKey("rk_test_cus_abc_xxxx") {
		t.Error("expected true for rk_test_ prefix")
	}
	if IsBillingKey("sk_live_xxxx") {
		t.Error("expected false for sk_live_ prefix")
	}
	if IsBillingKey("regular-api-key") {
		t.Error("expected false for regular key")
	}
}

func TestCustomerIDWithUnderscores(t *testing.T) {
	// Stripe customer IDs contain underscores like cus_abc_123
	customerID := "cus_NffrFeUfNV2Hib"
	key := GenerateAPIKey(customerID, testSecret, true)
	got, err := ValidateAPIKey(key, testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != customerID {
		t.Fatalf("expected %s, got %s", customerID, got)
	}
}
