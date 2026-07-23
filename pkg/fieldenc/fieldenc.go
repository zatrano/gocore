// Package fieldenc, hassas alanlar için isteğe bağlı AES-256-GCM şifreleme sağlar.
package fieldenc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const encPrefix = "enc:v1:"

// Cipher, alan düzeyi şifreleme yardımcısıdır. Anahtar boşsa düz metin geçirir.
type Cipher struct {
	aead cipher.AEAD
}

// New, base64 kodlu 32 baytlık anahtarla cipher kurar. Boş anahtar dev modu (şifresiz) demektir.
func New(keyB64 string) (*Cipher, error) {
	keyB64 = strings.TrimSpace(keyB64)
	if keyB64 == "" {
		return &Cipher{}, nil
	}
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("fieldenc: anahtar decode: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("fieldenc: anahtar 32 bayt olmalıdır (openssl rand -base64 32)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("fieldenc: aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("fieldenc: gcm: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

// Enabled, şifrelemenin aktif olup olmadığını döner.
func (c *Cipher) Enabled() bool { return c != nil && c.aead != nil }

// Encrypt, düz metni şifreler. Boş girdi veya dev modu değişmeden döner.
func (c *Cipher) Encrypt(plain string) (string, error) {
	if plain == "" || c == nil || c.aead == nil {
		return plain, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := c.aead.Seal(nonce, nonce, []byte(plain), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt, şifreli veya düz metni çözer.
func (c *Cipher) Decrypt(stored string) (string, error) {
	if stored == "" || !strings.HasPrefix(stored, encPrefix) {
		return stored, nil
	}
	if c == nil || c.aead == nil {
		return "", errors.New("fieldenc: şifreli alan var ancak anahtar yapılandırılmamış")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, encPrefix))
	if err != nil {
		return "", fmt.Errorf("fieldenc: ciphertext decode: %w", err)
	}
	nonceSize := c.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("fieldenc: geçersiz ciphertext")
	}
	plain, err := c.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("fieldenc: decrypt: %w", err)
	}
	return string(plain), nil
}
