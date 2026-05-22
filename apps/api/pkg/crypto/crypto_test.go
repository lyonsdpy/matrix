package crypto

import (
	"testing"
)

func TestEncryptTryDecryptRoundTrip(t *testing.T) {
	plain := "my_neo4j_password"
	enc, err := Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	dec := TryDecrypt(enc)
	if dec != plain {
		t.Fatalf("want %q, got %q", plain, dec)
	}
}

func TestTryDecryptPlainPassthrough(t *testing.T) {
	cases := []string{"plaintext", "matrix", "my_password_123", ""}
	for _, plain := range cases {
		if got := TryDecrypt(plain); got != plain {
			t.Fatalf("TryDecrypt(%q): want passthrough %q, got %q", plain, plain, got)
		}
	}
}

func TestEncryptProducesUniqueNonce(t *testing.T) {
	e1, _ := Encrypt("same")
	e2, _ := Encrypt("same")
	if e1 == e2 {
		t.Fatal("two encryptions of the same value should differ (random nonce)")
	}
}

func TestTryDecryptWrongKeyReturnsOriginal(t *testing.T) {
	// 伪造一段合法 base64 但内容随机（非本密钥加密），应原样返回
	garbage := "dGhpcyBpcyBub3QgYSB2YWxpZCBjaXBoZXJ0ZXh0IGF0IGFsbA=="
	if got := TryDecrypt(garbage); got != garbage {
		t.Fatalf("want original %q back, got %q", garbage, got)
	}
}
