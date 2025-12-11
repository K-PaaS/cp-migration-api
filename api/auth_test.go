package api

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strconv"
	"testing"

	"kps-migration-api/config"
	"kps-migration-api/model"
)

func TestPkcs7Unpad_Valid(t *testing.T) {
	data := []byte{1, 2, 3, 4, 4, 4, 4}
	out, err := pkcs7Unpad(data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := []byte{1, 2, 3}
	if len(out) != len(expected) {
		t.Fatalf("expected len %d, got %d", len(expected), len(out))
	}
	for i := range expected {
		if out[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, out)
		}
	}
}

func TestPkcs7Unpad_Empty(t *testing.T) {
	_, err := pkcs7Unpad([]byte{})
	if err == nil {
		t.Fatalf("expected error for empty input, got nil")
	}
}

func TestPkcs7Unpad_InvalidLength(t *testing.T) {
	_, err := pkcs7Unpad([]byte{1, 2, 3, 10})
	if err == nil {
		t.Fatalf("expected error for invalid padding length, got nil")
	}
}

func TestPkcs7Unpad_IncorrectBytes(t *testing.T) {
	_, err := pkcs7Unpad([]byte{1, 2, 3, 4, 4, 4, 5})
	if err == nil {
		t.Fatalf("expected error for incorrect padding bytes, got nil")
	}
}

func TestAesDecrypt_InvalidBlockSize(t *testing.T) {
	key := make([]byte, 16)
	iv := make([]byte, aes.BlockSize)
	cipherText := []byte{1, 2, 3}

	_, err := aesDecrypt(cipherText, key, iv)
	if err == nil {
		t.Fatalf("expected error for non-multiple block size, got nil")
	}
}

func TestVaildTimestampCheck_BothBranches(t *testing.T) {
	if !vaildTimestampCheck("0", "1000") {
		t.Fatalf("expected timestamp to be valid")
	}
	if vaildTimestampCheck("0", "99999999") {
		t.Fatalf("expected timestamp to be invalid for large difference")
	}
}

func TestCurrentTimestamp_IsNumeric(t *testing.T) {
	ts := currentTimestamp()
	if _, err := strconv.Atoi(ts); err != nil {
		t.Fatalf("expected numeric timestamp, got %q (err=%v)", ts, err)
	}
}

func TestHmacEncode_Deterministic(t *testing.T) {
	if config.Env == nil {
		t.Skip("config.Env is nil; skipping because no HMAC key configured")
	}

	v1 := hmacEncode([]byte(`{"a":1}`))
	v2 := hmacEncode([]byte(`{"a":1}`))
	v3 := hmacEncode([]byte(`{"a":2}`))

	if v1 != v2 {
		t.Fatalf("expected hmacEncode to be deterministic, got %q and %q", v1, v2)
	}
	if v1 == v3 {
		t.Fatalf("expected different payloads to have different HMACs, but both were %q", v1)
	}
}

func TestDecodePayload_Base64ErrorOnData(t *testing.T) {
	payload := model.HybridPayload{
		EncryptedData: "%%%not-base64%%%",
		EncryptedKey:  "",
		IV:            "",
	}
	_, err := decodePayload[struct{}](payload)
	if err == nil || err.Error() != "base64 decode failed" {
		t.Fatalf("expected base64 decode failed error, got %v", err)
	}
}

func TestDecodePayload_RsaDecodeError(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("data"))
	payload := model.HybridPayload{
		EncryptedData: encoded,
		EncryptedKey:  encoded,
		IV:            encoded,
	}

	oldRsa := rsaDecodeFunc
	oldAes := aesDecryptFunc
	oldNow := currentTimestampFunc
	oldHmac := hmacEncodeFunc
	oldValid := vaildTimestampCheckFunc
	defer func() {
		rsaDecodeFunc = oldRsa
		aesDecryptFunc = oldAes
		currentTimestampFunc = oldNow
		hmacEncodeFunc = oldHmac
		vaildTimestampCheckFunc = oldValid
	}()

	rsaDecodeFunc = func(b []byte) ([]byte, error) {
		return nil, errors.New("rsa fail")
	}
	aesDecryptFunc = func(cipherText, key, iv []byte) ([]byte, error) {
		return nil, errors.New("should not reach aesDecrypt")
	}
	currentTimestampFunc = func() string { return "0" }
	hmacEncodeFunc = func(b []byte) string { return "" }
	vaildTimestampCheckFunc = func(o, c string) bool { return true }

	_, err := decodePayload[struct{}](payload)
	if err == nil || err.Error() != "rsa fail" {
		t.Fatalf("expected rsa fail error, got %v", err)
	}
}

func TestDecodePayload_HmacMismatch(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("data"))
	payload := model.HybridPayload{
		EncryptedData: encoded,
		EncryptedKey:  encoded,
		IV:            encoded,
	}

	oldRsa := rsaDecodeFunc
	oldAes := aesDecryptFunc
	oldNow := currentTimestampFunc
	oldHmac := hmacEncodeFunc
	oldValid := vaildTimestampCheckFunc
	defer func() {
		rsaDecodeFunc = oldRsa
		aesDecryptFunc = oldAes
		currentTimestampFunc = oldNow
		hmacEncodeFunc = oldHmac
		vaildTimestampCheckFunc = oldValid
	}()

	rsaDecodeFunc = func(b []byte) ([]byte, error) { return []byte("key"), nil }

	type inner struct {
		Value string `json:"value"`
	}
	type payloadWithHMAC struct {
		Data      inner  `json:"data"`
		Timestamp string `json:"timestamp"`
		Hmac_data string `json:"hmac_data"`
	}

	decodedJSON, _ := json.Marshal(payloadWithHMAC{
		Data:      inner{Value: "x"},
		Timestamp: "0",
		Hmac_data: "bad-hmac",
	})

	aesDecryptFunc = func(cipherText, key, iv []byte) ([]byte, error) {
		return decodedJSON, nil
	}
	currentTimestampFunc = func() string { return "0" }
	hmacEncodeFunc = hmacEncode
	vaildTimestampCheckFunc = func(o, c string) bool { return true }

	_, err := decodePayload[inner](payload)
	if err == nil || err.Error() != "integrity verification failed" {
		t.Fatalf("expected integrity verification failed, got %v", err)
	}
}

func TestDecodePayload_Success(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("data"))
	payload := model.HybridPayload{
		EncryptedData: encoded,
		EncryptedKey:  encoded,
		IV:            encoded,
	}

	oldRsa := rsaDecodeFunc
	oldAes := aesDecryptFunc
	oldNow := currentTimestampFunc
	oldHmac := hmacEncodeFunc
	oldValid := vaildTimestampCheckFunc
	defer func() {
		rsaDecodeFunc = oldRsa
		aesDecryptFunc = oldAes
		currentTimestampFunc = oldNow
		hmacEncodeFunc = oldHmac
		vaildTimestampCheckFunc = oldValid
	}()

	rsaDecodeFunc = func(b []byte) ([]byte, error) { return []byte("key"), nil }

	type inner struct {
		Value string `json:"value"`
	}
	type payloadWithHMAC struct {
		Data      inner  `json:"data"`
		Timestamp string `json:"timestamp"`
		Hmac_data string `json:"hmac_data"`
	}

	currentTimestampFunc = func() string { return "1000" }
	vaildTimestampCheckFunc = func(o, c string) bool { return true }
	hmacEncodeFunc = func(b []byte) string { return "good-hmac" }

	decodedJSON, _ := json.Marshal(payloadWithHMAC{
		Data:      inner{Value: "ok"},
		Timestamp: "0",
		Hmac_data: "good-hmac",
	})

	aesDecryptFunc = func(cipherText, key, iv []byte) ([]byte, error) {
		return decodedJSON, nil
	}

	out, err := decodePayload[inner](payload)
	if err != nil {
		t.Fatalf("expected success, got error %v", err)
	}
	if out.Value != "ok" {
		t.Fatalf("expected value ok, got %+v", out)
	}
}

func TestCurrentTimestamp_Numeric(t *testing.T) {
	ts := currentTimestamp()
	if ts == "" {
		t.Fatal("expected non-empty timestamp")
	}
	if _, err := strconv.ParseInt(ts, 10, 64); err != nil {
		t.Fatalf("expected numeric timestamp, got %q (%v)", ts, err)
	}
}

func TestAESDecrypt_InvalidKey(t *testing.T) {
	cipherText := []byte("1234567890abcdef")
	iv := []byte("1234567890abcdef")
	key := []byte("short")
	if _, err := aesDecrypt(cipherText, key, iv); err == nil {
		t.Fatalf("expected error for invalid key size, got nil")
	}
}

func TestAESDecrypt_InvalidBlockSize(t *testing.T) {
	key := make([]byte, 32) // AES-256
	iv := make([]byte, aes.BlockSize)
	cipherText := make([]byte, aes.BlockSize+1)
	_, err := aesDecrypt(cipherText, key, iv)
	if err == nil || err.Error() != "cipherText is not a multiple of the block size" {
		t.Fatalf("expected block size error, got %v", err)
	}
}

func TestRSADecode_Success(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey error: %v", err)
	}
	pemPriv := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	config.Env.PrivateKey = string(pemPriv)

	plaintext := []byte("secret data")
	cipherText, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &priv.PublicKey, plaintext, nil)
	if err != nil {
		t.Fatalf("EncryptOAEP error: %v", err)
	}

	decoded, err := rsaDecode(cipherText)
	if err != nil {
		t.Fatalf("rsaDecode error: %v", err)
	}
	if string(decoded) != string(plaintext) {
		t.Fatalf("expected %q, got %q", string(plaintext), string(decoded))
	}
}

func TestRSADecode_InvalidCipher(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey error: %v", err)
	}
	pemPriv := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	config.Env.PrivateKey = string(pemPriv)

	if _, err := rsaDecode([]byte("not-a-valid-ciphertext")); err == nil {
		t.Fatalf("expected error from rsaDecode with invalid ciphertext, got nil")
	}
}

func TestDecodePayload_Expired(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("data"))
	payload := model.HybridPayload{
		EncryptedData: encoded,
		EncryptedKey:  encoded,
		IV:            encoded,
	}

	oldRsa := rsaDecodeFunc
	oldAes := aesDecryptFunc
	oldNow := currentTimestampFunc
	oldHmac := hmacEncodeFunc
	oldValid := vaildTimestampCheckFunc
	defer func() {
		rsaDecodeFunc = oldRsa
		aesDecryptFunc = oldAes
		currentTimestampFunc = oldNow
		hmacEncodeFunc = oldHmac
		vaildTimestampCheckFunc = oldValid
	}()

	rsaDecodeFunc = func(b []byte) ([]byte, error) { return []byte("key"), nil }

	type inner struct {
		Value string `json:"value"`
	}
	type payloadWithHMAC struct {
		Data      inner  `json:"data"`
		Timestamp string `json:"timestamp"`
		Hmac_data string `json:"hmac_data"`
	}

	decodedJSON, _ := json.Marshal(payloadWithHMAC{
		Data:      inner{Value: "ok"},
		Timestamp: "0",
		Hmac_data: "good-hmac",
	})

	aesDecryptFunc = func(cipherText, key, iv []byte) ([]byte, error) {
		return decodedJSON, nil
	}
	currentTimestampFunc = func() string { return "1000" }
	hmacEncodeFunc = func(b []byte) string { return "good-hmac" }
	vaildTimestampCheckFunc = func(o, c string) bool { return false }

	_, err := decodePayload[inner](payload)
	if err == nil || err.Error() != "your request has expired" {
		t.Fatalf("expected your request has expired error, got %v", err)
	}
}
