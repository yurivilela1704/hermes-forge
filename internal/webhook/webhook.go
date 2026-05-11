package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// Commit contains a single commit entry normalized across providers.
type Commit struct {
	Message string `json:"message"`
	Author  string `json:"author"`
}

// Event is a normalized webhook payload used by the rest of the app.
type Event struct {
	Provider    string    `json:"provider"`
	Repository  string    `json:"repository"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Branch      string    `json:"branch"`
	Description string    `json:"description"`
	Number      int       `json:"number"`
	URL         string    `json:"url"`
	MergedAt    time.Time `json:"merged_at"`
	Labels      string    `json:"labels"`
	GeneratedAt string    `json:"generated_at"`
	Commits     []Commit  `json:"commits"`
}

var errInvalidSignature = errors.New("invalid signature")

func validateGitLabToken(secret, token string) error {
	if strings.TrimSpace(secret) == "" {
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(secret), []byte(token)) != 1 {
		return errInvalidSignature
	}
	return nil
}

func validateGitHubSignature(secret string, payload []byte, signatureHeader string) error {
	if strings.TrimSpace(secret) == "" {
		return nil
	}
	signature := strings.TrimSpace(signatureHeader)
	if signature == "" {
		return errInvalidSignature
	}
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return errInvalidSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	receivedHex := strings.TrimPrefix(signature, prefix)
	received, err := hex.DecodeString(receivedHex)
	if err != nil {
		return errInvalidSignature
	}
	if subtle.ConstantTimeCompare(expected, received) != 1 {
		return errInvalidSignature
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func readPayload(r *http.Request) ([]byte, error) {
	const maxPayloadBytes = 2 << 20 // 2MB
	return io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes))
}
