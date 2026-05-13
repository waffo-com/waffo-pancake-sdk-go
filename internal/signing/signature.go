package signing

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
)

// SignRequest builds the canonical request string
//
//	METHOD\nPATH\nTIMESTAMP\nSHA256_BASE64(BODY)
//
// and returns the base64-encoded RSA-SHA256 (PKCS#1 v1.5) signature.
func SignRequest(method, path, timestamp string, body []byte, key *rsa.PrivateKey) (string, error) {
	bodyHash := sha256.Sum256(body)
	bodyB64 := base64.StdEncoding.EncodeToString(bodyHash[:])
	canonical := method + "\n" + path + "\n" + timestamp + "\n" + bodyB64
	canonicalHash := sha256.Sum256([]byte(canonical))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, canonicalHash[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifySignature verifies a base64-encoded RSA-SHA256 signature over the
// given input string.
func VerifySignature(input, signatureB64 string, key *rsa.PublicKey) error {
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(input))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], sig)
}
