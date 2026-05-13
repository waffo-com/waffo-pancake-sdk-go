// Package signing implements RSA-SHA256 request signing and PEM key
// normalization. It is internal because callers should not bypass the SDK's
// signed HTTP client to invoke the Waffo Pancake API directly.
package signing

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var base64Re = regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`)

// NormalizePrivateKey accepts any of the common PEM private-key formats and
// returns a well-formed PEM string parseable by crypto/x509. It handles
// literal "\n" sequences from environment variables, Windows "\r\n" line
// endings, leading/trailing whitespace, single-line base64 content, and raw
// base64 without PEM headers (assumed PKCS#8).
func NormalizePrivateKey(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("private key is empty; provide an RSA private key in PEM format")
	}

	pemStr := strings.ReplaceAll(raw, "\\n", "\n")
	pemStr = strings.ReplaceAll(pemStr, "\r\n", "\n")
	pemStr = strings.TrimSpace(pemStr)

	const (
		pkcs8Header = "-----BEGIN PRIVATE KEY-----"
		pkcs8Footer = "-----END PRIVATE KEY-----"
		pkcs1Header = "-----BEGIN RSA PRIVATE KEY-----"
		pkcs1Footer = "-----END RSA PRIVATE KEY-----"
	)

	hasPKCS1 := strings.Contains(pemStr, pkcs1Header)
	hasPKCS8 := strings.Contains(pemStr, pkcs8Header)

	var rebuilt string
	if hasPKCS1 || hasPKCS8 {
		stripped := pemStr
		stripped = strings.ReplaceAll(stripped, pkcs8Header, "")
		stripped = strings.ReplaceAll(stripped, pkcs8Footer, "")
		stripped = strings.ReplaceAll(stripped, pkcs1Header, "")
		stripped = strings.ReplaceAll(stripped, pkcs1Footer, "")
		b64 := stripWhitespace(stripped)
		if b64 == "" {
			return "", errors.New("private key contains PEM headers but no key data")
		}
		header, footer := pkcs8Header, pkcs8Footer
		if hasPKCS1 {
			header, footer = pkcs1Header, pkcs1Footer
		}
		rebuilt = header + "\n" + wrap64(b64) + "\n" + footer
	} else {
		b64 := stripWhitespace(pemStr)
		if !base64Re.MatchString(b64) {
			return "", errors.New("private key is not valid PEM or base64; expected an RSA private key in PEM format or raw base64")
		}
		rebuilt = pkcs8Header + "\n" + wrap64(b64) + "\n" + pkcs8Footer
	}

	if _, err := ParsePrivateKey(rebuilt); err != nil {
		return "", fmt.Errorf("private key could not be parsed: %w", err)
	}
	return rebuilt, nil
}

// NormalizePublicKey is the public-key counterpart of NormalizePrivateKey.
// SPKI ("BEGIN PUBLIC KEY") and PKCS#1 ("BEGIN RSA PUBLIC KEY") inputs are
// both accepted; raw base64 is treated as SPKI.
func NormalizePublicKey(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("public key is empty; provide an RSA public key in PEM format")
	}

	pemStr := strings.ReplaceAll(raw, "\\n", "\n")
	pemStr = strings.ReplaceAll(pemStr, "\r\n", "\n")
	pemStr = strings.TrimSpace(pemStr)

	const (
		spkiHeader  = "-----BEGIN PUBLIC KEY-----"
		spkiFooter  = "-----END PUBLIC KEY-----"
		pkcs1Header = "-----BEGIN RSA PUBLIC KEY-----"
		pkcs1Footer = "-----END RSA PUBLIC KEY-----"
	)

	hasPKCS1 := strings.Contains(pemStr, pkcs1Header)
	hasSPKI := strings.Contains(pemStr, spkiHeader)

	var rebuilt string
	if hasPKCS1 || hasSPKI {
		stripped := pemStr
		stripped = strings.ReplaceAll(stripped, spkiHeader, "")
		stripped = strings.ReplaceAll(stripped, spkiFooter, "")
		stripped = strings.ReplaceAll(stripped, pkcs1Header, "")
		stripped = strings.ReplaceAll(stripped, pkcs1Footer, "")
		b64 := stripWhitespace(stripped)
		if b64 == "" {
			return "", errors.New("public key contains PEM headers but no key data")
		}
		header, footer := spkiHeader, spkiFooter
		if hasPKCS1 {
			header, footer = pkcs1Header, pkcs1Footer
		}
		rebuilt = header + "\n" + wrap64(b64) + "\n" + footer
	} else {
		b64 := stripWhitespace(pemStr)
		if !base64Re.MatchString(b64) {
			return "", errors.New("public key is not valid PEM or base64; expected an RSA public key in PEM format or raw base64")
		}
		rebuilt = spkiHeader + "\n" + wrap64(b64) + "\n" + spkiFooter
	}

	if _, err := ParsePublicKey(rebuilt); err != nil {
		return "", fmt.Errorf("public key could not be parsed: %w", err)
	}
	return rebuilt, nil
}

// ParsePrivateKey decodes a PEM-encoded RSA private key. PKCS#1 and PKCS#8
// inputs are both accepted; non-RSA keys are rejected.
func ParsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("no PEM block found in private key")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("PKCS#8 private key is not an RSA key")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type for private key: %q", block.Type)
	}
}

// ParsePublicKey decodes a PEM-encoded RSA public key. SPKI ("PUBLIC KEY")
// and PKCS#1 ("RSA PUBLIC KEY") inputs are both accepted.
func ParsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("no PEM block found in public key")
	}
	switch block.Type {
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("SPKI public key is not an RSA key")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type for public key: %q", block.Type)
	}
}

func stripWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func wrap64(s string) string {
	if len(s) <= 64 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + len(s)/64)
	for i := 0; i < len(s); i += 64 {
		end := i + 64
		if end > len(s) {
			end = len(s)
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(s[i:end])
	}
	return b.String()
}
