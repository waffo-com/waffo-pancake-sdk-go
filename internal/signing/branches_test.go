package signing

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

func TestParsePrivateKey_PKCS1(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))
	if _, err := ParsePrivateKey(pemStr); err != nil {
		t.Fatalf("ParsePrivateKey PKCS#1: %v", err)
	}
}

func TestParsePrivateKey_UnknownBlock(t *testing.T) {
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte{0x01, 0x02}}))
	if _, err := ParsePrivateKey(pemStr); err == nil {
		t.Fatal("expected error for unknown block")
	}
}

func TestParsePrivateKey_NoPEMBlock(t *testing.T) {
	if _, err := ParsePrivateKey("nothing useful"); err == nil {
		t.Fatal("expected error for no PEM block")
	}
}

func TestParsePublicKey_PKCS1(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	der := x509.MarshalPKCS1PublicKey(&key.PublicKey)
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: der}))
	if _, err := ParsePublicKey(pemStr); err != nil {
		t.Fatalf("ParsePublicKey PKCS#1: %v", err)
	}
}

func TestParsePublicKey_UnknownBlock(t *testing.T) {
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "DH PARAMETERS", Bytes: []byte{0x01}}))
	if _, err := ParsePublicKey(pemStr); err == nil {
		t.Fatal("expected error for unknown block")
	}
}

func TestParsePublicKey_NoPEMBlock(t *testing.T) {
	if _, err := ParsePublicKey("nothing useful"); err == nil {
		t.Fatal("expected error for no PEM block")
	}
}

func TestNormalizePublicKey_LiteralBackslashN(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	envFormat := strings.ReplaceAll(pemStr, "\n", "\\n")
	out, err := NormalizePublicKey(envFormat)
	if err != nil {
		t.Fatalf("normalize literal \\n: %v", err)
	}
	if _, err := ParsePublicKey(out); err != nil {
		t.Fatalf("parse normalized: %v", err)
	}
}

func TestNormalizePublicKey_RawBase64(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	raw := pemStr
	for _, m := range []string{"-----BEGIN PUBLIC KEY-----", "-----END PUBLIC KEY-----", "\n", "\r"} {
		raw = strings.ReplaceAll(raw, m, "")
	}
	out, err := NormalizePublicKey(raw)
	if err != nil {
		t.Fatalf("normalize raw base64: %v", err)
	}
	if _, err := ParsePublicKey(out); err != nil {
		t.Fatalf("parse: %v", err)
	}
}

func TestNormalizePublicKey_RejectsEmpty(t *testing.T) {
	if _, err := NormalizePublicKey(""); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestNormalizePublicKey_RejectsInvalid(t *testing.T) {
	if _, err := NormalizePublicKey("not-valid!!!"); err == nil {
		t.Fatal("expected error for garbage")
	}
}

func TestVerifySignature_RejectsBadBase64(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	if err := VerifySignature("x", "not!base64!", &key.PublicKey); err == nil {
		t.Fatal("expected error for invalid base64 signature")
	}
}
