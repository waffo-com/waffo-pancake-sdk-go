package signing

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

func genPEMPrivateKey(t *testing.T, bits int) (string, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	blk := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	return string(pem.EncodeToMemory(blk)), key
}

func genPEMPublicKey(t *testing.T, pub *rsa.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}
	blk := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	return string(pem.EncodeToMemory(blk))
}

func TestNormalizePrivateKey_AcceptsStandardPEM(t *testing.T) {
	pemStr, _ := genPEMPrivateKey(t, 2048)
	out, err := NormalizePrivateKey(pemStr)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if _, err := ParsePrivateKey(out); err != nil {
		t.Fatalf("parse normalized: %v", err)
	}
}

func TestNormalizePrivateKey_AcceptsLiteralBackslashN(t *testing.T) {
	pemStr, _ := genPEMPrivateKey(t, 2048)
	envFormat := strings.ReplaceAll(pemStr, "\n", "\\n")
	out, err := NormalizePrivateKey(envFormat)
	if err != nil {
		t.Fatalf("normalize literal \\n: %v", err)
	}
	if _, err := ParsePrivateKey(out); err != nil {
		t.Fatalf("parse normalized: %v", err)
	}
}

func TestNormalizePrivateKey_AcceptsWindowsCRLF(t *testing.T) {
	pemStr, _ := genPEMPrivateKey(t, 2048)
	winFormat := strings.ReplaceAll(pemStr, "\n", "\r\n")
	out, err := NormalizePrivateKey(winFormat)
	if err != nil {
		t.Fatalf("normalize crlf: %v", err)
	}
	if _, err := ParsePrivateKey(out); err != nil {
		t.Fatalf("parse normalized: %v", err)
	}
}

func TestNormalizePrivateKey_AcceptsRawBase64(t *testing.T) {
	pemStr, _ := genPEMPrivateKey(t, 2048)
	raw := pemStr
	for _, marker := range []string{"-----BEGIN PRIVATE KEY-----", "-----END PRIVATE KEY-----", "\n", "\r"} {
		raw = strings.ReplaceAll(raw, marker, "")
	}
	out, err := NormalizePrivateKey(raw)
	if err != nil {
		t.Fatalf("normalize raw base64: %v", err)
	}
	if _, err := ParsePrivateKey(out); err != nil {
		t.Fatalf("parse normalized: %v", err)
	}
}

func TestNormalizePrivateKey_RejectsEmpty(t *testing.T) {
	if _, err := NormalizePrivateKey(""); err == nil {
		t.Fatal("expected error for empty input")
	}
	if _, err := NormalizePrivateKey("   \n  "); err == nil {
		t.Fatal("expected error for whitespace-only input")
	}
}

func TestNormalizePrivateKey_RejectsInvalid(t *testing.T) {
	if _, err := NormalizePrivateKey("not-a-pem-or-base64!!!"); err == nil {
		t.Fatal("expected error for garbage input")
	}
}

func TestNormalizePublicKey_RoundTrip(t *testing.T) {
	_, priv := genPEMPrivateKey(t, 2048)
	pubPEM := genPEMPublicKey(t, &priv.PublicKey)
	out, err := NormalizePublicKey(pubPEM)
	if err != nil {
		t.Fatalf("normalize public: %v", err)
	}
	parsed, err := ParsePublicKey(out)
	if err != nil {
		t.Fatalf("parse normalized public: %v", err)
	}
	if parsed.N.Cmp(priv.N) != 0 {
		t.Fatal("normalized public key did not round-trip")
	}
}

func TestSignAndVerify_RoundTrip(t *testing.T) {
	_, priv := genPEMPrivateKey(t, 2048)
	sig, err := SignRequest("POST", "/v1/test", "1700000000", []byte(`{"foo":"bar"}`), priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	canonical := "POST\n/v1/test\n1700000000\n" + signedBodyHashBase64([]byte(`{"foo":"bar"}`))
	if err := VerifySignature(canonical, sig, &priv.PublicKey); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifySignature_RejectsTamperedInput(t *testing.T) {
	_, priv := genPEMPrivateKey(t, 2048)
	sig, _ := SignRequest("POST", "/v1/test", "1700000000", []byte(`{"foo":"bar"}`), priv)
	canonical := "POST\n/v1/test\n1700000000\n" + signedBodyHashBase64([]byte(`{"foo":"BAZ"}`))
	if err := VerifySignature(canonical, sig, &priv.PublicKey); err == nil {
		t.Fatal("expected verification failure for tampered body")
	}
}
