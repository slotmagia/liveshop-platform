package secretbox

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestSealUsesRandomNonceAndBindsAssociatedData(t *testing.T) {
	box, err := New("key-1", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	plain, aad := []byte("customer-secret"), []byte("live-provider:1:agora")
	first, err := box.Seal(plain, aad)
	if err != nil {
		t.Fatal(err)
	}
	second, err := box.Seal(plain, aad)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) || bytes.Contains(first, plain) {
		t.Fatal("ciphertext leaked plaintext or reused a nonce")
	}
	opened, err := box.Open(first, aad)
	if err != nil || !bytes.Equal(opened, plain) {
		t.Fatalf("open=%q err=%v", opened, err)
	}
	if _, err := box.Open(first, []byte("live-provider:2:agora")); err == nil {
		t.Fatal("ciphertext opened under a different provider identity")
	}
}
