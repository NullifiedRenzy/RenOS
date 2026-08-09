package enc

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := DeriveKey("test-passphrase-32-chars-long!!!")
	plaintext := []byte("super-secret-password-12345")

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if encrypted == "" {
		t.Fatal("encrypted output is empty")
	}
	if encrypted == string(plaintext) {
		t.Fatal("encrypted output is same as plaintext")
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted text mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDifferentEachTime(t *testing.T) {
	key := DeriveKey("test-key")
	plaintext := []byte("same plaintext")

	e1, _ := Encrypt(key, plaintext)
	e2, _ := Encrypt(key, plaintext)
	if e1 == e2 {
		t.Fatal("two encryptions of same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := DeriveKey("key-one")
	key2 := DeriveKey("key-two")
	plaintext := []byte("secret data")

	encrypted, err := Encrypt(key1, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(key2, encrypted)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}

func TestDecryptInvalidData(t *testing.T) {
	key := DeriveKey("test-key")
	_, err := Decrypt(key, "not-valid-base64!!!")
	if err == nil {
		t.Fatal("Decrypt with invalid data should fail")
	}
}

func TestDecryptEmptyKey(t *testing.T) {
	_, err := Decrypt(nil, "somedata")
	if err != ErrEmptyKey {
		t.Fatal("Decrypt with empty key should return ErrEmptyKey")
	}
}

func TestEncryptEmptyKey(t *testing.T) {
	_, err := Encrypt(nil, []byte("data"))
	if err != ErrEmptyKey {
		t.Fatal("Encrypt with empty key should return ErrEmptyKey")
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	key := DeriveKey("test-key")
	encrypted, err := Encrypt(key, []byte{})
	if err != nil {
		t.Fatalf("Encrypt empty plaintext failed: %v", err)
	}
	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt empty plaintext failed: %v", err)
	}
	if len(decrypted) != 0 {
		t.Fatalf("expected empty plaintext, got %d bytes", len(decrypted))
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	k1 := DeriveKey("same-passphrase")
	k2 := DeriveKey("same-passphrase")
	if !bytes.Equal(k1, k2) {
		t.Fatal("DeriveKey should be deterministic for same input")
	}
}

func TestDeriveKeyLength(t *testing.T) {
	key := DeriveKey("any passphrase")
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestLargePayload(t *testing.T) {
	key := DeriveKey("test-key")
	plaintext := make([]byte, 1024*1024) // 1MB
	rand.Read(plaintext)

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt large payload failed: %v", err)
	}
	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt large payload failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("large payload round-trip mismatch")
	}
}
