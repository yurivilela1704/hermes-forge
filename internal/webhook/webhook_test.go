package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestValidateGitLabToken(t *testing.T) {
	t.Parallel()

	if err := validateGitLabToken("secret", "secret"); err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}
	if err := validateGitLabToken("secret", "wrong"); err == nil {
		t.Fatal("expected invalid token error")
	}
}

func TestValidateGitHubSignature(t *testing.T) {
	t.Parallel()

	secret := "super-secret"
	payload := []byte(`{"hello":"world"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if err := validateGitHubSignature(secret, payload, signature); err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}
	if err := validateGitHubSignature(secret, payload, "sha256=deadbeef"); err == nil {
		t.Fatal("expected invalid signature error")
	}
}
