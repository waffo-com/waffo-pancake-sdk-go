package signing

import (
	"crypto/sha256"
	"encoding/base64"
)

// signedBodyHashBase64 mirrors the SHA256-base64 body digest used inside
// SignRequest so tests can reconstruct the canonical signature input.
func signedBodyHashBase64(body []byte) string {
	h := sha256.Sum256(body)
	return base64.StdEncoding.EncodeToString(h[:])
}
