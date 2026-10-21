package compiler

import (
	"encoding/base64"
	"strings"
)

// ValidSSCiphers defines Shadowsocks ciphers supported across Mihomo and Sing-box.
var ValidSSCiphers = map[string]bool{
	"aes-128-gcm": true, "aes-192-gcm": true, "aes-256-gcm": true,
	"aes-128-cfb": true, "aes-192-cfb": true, "aes-256-cfb": true,
	"aes-128-ctr": true, "aes-192-ctr": true, "aes-256-ctr": true,
	"rc4-md5": true, "chacha20-ietf": true, "chacha20-ietf-poly1305": true,
	"xchacha20-ietf-poly1305": true,
	"2022-blake3-aes-128-gcm": true, "2022-blake3-aes-256-gcm": true,
	"2022-blake3-chacha20-poly1305": true,
	"none": true, "dummy": true,
}

// cleanSingleSSKey decodes and validates a single base64 key against expectedLen.
func cleanSingleSSKey(rawKey string, expectedLen int) (string, bool) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return "", false
	}
	// Try standard decoding first
	decoded, err := base64.StdEncoding.DecodeString(rawKey)
	if err == nil && len(decoded) == expectedLen {
		return base64.StdEncoding.EncodeToString(decoded), true
	}

	// Try normalizing URL-safe base64 and missing padding
	padded := strings.ReplaceAll(rawKey, "-", "+")
	padded = strings.ReplaceAll(padded, "_", "/")
	pad := len(padded) % 4
	if pad > 0 {
		padded += strings.Repeat("=", 4-pad)
	}

	decoded, err = base64.StdEncoding.DecodeString(padded)
	if err == nil && len(decoded) == expectedLen {
		return base64.StdEncoding.EncodeToString(decoded), true
	}

	return "", false
}

// CleanSS2022Password validates and normalizes Shadowsocks-2022 keys.
// Supports both single-key and multi-user format (<server_key>:<user_key>).
// If URL-safe base64 is detected, it converts it to standard base64 and verifies exact byte length.
func CleanSS2022Password(cipher, rawPwd string) (string, bool) {
	cipher = strings.ToLower(cipher)
	if !strings.HasPrefix(cipher, "2022-") {
		return rawPwd, true
	}

	expectedLen := 0
	switch cipher {
	case "2022-blake3-aes-128-gcm":
		expectedLen = 16
	case "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305":
		expectedLen = 32
	default:
		return "", false
	}

	// Multi-user format (<server_key>:<user_key>)
	if strings.Contains(rawPwd, ":") {
		parts := strings.Split(rawPwd, ":")
		if len(parts) != 2 {
			return "", false
		}
		k1, ok1 := cleanSingleSSKey(parts[0], expectedLen)
		k2, ok2 := cleanSingleSSKey(parts[1], expectedLen)
		if !ok1 || !ok2 {
			return "", false
		}
		return k1 + ":" + k2, true
	}

	return cleanSingleSSKey(rawPwd, expectedLen)
}
