// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package collab

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"unicode"
)

// Key is a shared script's secret: whoever has it can read and edit the
// script. It travels in a link's fragment (#k=...), which browsers never
// send to a server, or as text.
type Key [32]byte

// NewKey is a new random key.
func NewKey() (Key, error) {
	var k Key
	_, err := rand.Read(k[:])
	return k, err
}

// String is the key as text, URL-safe.
func (k Key) String() string { return base64.RawURLEncoding.EncodeToString(k[:]) }

// keyLen is a key's length as text.
var keyLen = base64.RawURLEncoding.EncodedLen(len(Key{}))

// ParseKey reads a key from its text, or from a link with #k=. What a chat
// or mail does to a link is ignored: text around it (a full stop, a
// bracket, quotes), white space or a line break inside it.
func ParseKey(s string) (Key, error) {
	var k Key
	if i := strings.LastIndex(s, "k="); i >= 0 {
		s = s[i+2:]
	}
	var b strings.Builder
	for _, r := range s {
		if b.Len() == keyLen {
			break
		}
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case unicode.IsSpace(r): // wrapped onto two lines
		case b.Len() > 0: // what follows it
			return k, errors.New("not a shared script's key")
		}
	}
	raw, err := base64.RawURLEncoding.DecodeString(b.String())
	if err != nil || len(raw) != len(k) {
		return k, errors.New("not a shared script's key")
	}
	copy(k[:], raw)
	return k, nil
}

func (k Key) derive(label string) []byte {
	m := hmac.New(sha256.New, k[:])
	m.Write([]byte(label))
	return m.Sum(nil)
}

// Room is where the script's peers meet (a tracker's info hash): derived
// from the key, it tells nothing about it or the script.
func (k Key) Room() [20]byte {
	var r [20]byte
	copy(r[:], k.derive("accolade room v1"))
	return r
}

func (k Key) aead() (cipher.AEAD, error) {
	block, err := aes.NewCipher(k.derive("accolade seal v1"))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// sealLabel binds every message to its use.
var sealLabel = []byte("accolade collab v1")

// Seal encrypts and authenticates a message for the script's peers.
func (k Key) Seal(msg []byte) ([]byte, error) {
	a, err := k.aead()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return a.Seal(nonce, nonce, msg, sealLabel), nil
}

// Open decrypts a sealed message; it fails for anything not sealed with
// the key.
func (k Key) Open(sealed []byte) ([]byte, error) {
	a, err := k.aead()
	if err != nil {
		return nil, err
	}
	if len(sealed) < a.NonceSize() {
		return nil, errors.New("not a message of this script")
	}
	msg, err := a.Open(nil, sealed[:a.NonceSize()], sealed[a.NonceSize():], sealLabel)
	if err != nil {
		return nil, errors.New("not a message of this script")
	}
	return msg, nil
}
