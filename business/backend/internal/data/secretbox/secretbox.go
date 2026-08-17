// Package secretbox encrypts provider credentials before they reach MySQL.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

type Box struct {
	keyID string
	aead  cipher.AEAD
}

func New(keyID, encodedKey string) (*Box, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 32 || keyID == "" {
		return nil, errors.New("secretbox: valid key id and 32-byte key are required")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{keyID: keyID, aead: aead}, nil
}

func (b *Box) KeyID() string { return b.keyID }

func (b *Box) Seal(plain, associated []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return b.aead.Seal(nonce, nonce, plain, associated), nil
}

func (b *Box) Open(sealed, associated []byte) ([]byte, error) {
	if len(sealed) < b.aead.NonceSize() {
		return nil, errors.New("secretbox: ciphertext is invalid")
	}
	nonce := sealed[:b.aead.NonceSize()]
	return b.aead.Open(nil, nonce, sealed[b.aead.NonceSize():], associated)
}
