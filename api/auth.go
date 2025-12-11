package api

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"kps-migration-api/config"
	"kps-migration-api/model"
	"strconv"
	"time"
)

// 1. 현재 timestamp function 호출 (param x, return timestamp)
func currentTimestamp() string {
	timestamp := strconv.FormatInt(time.Now().UTC().UnixMilli(), 10)
	return timestamp //        1744352110
}

// 3. 객체를 json string화 하여 hmac encode function 호출 (param string, return string)
func hmacEncode(json []byte) string {
	hmac256 := hmac.New(sha256.New, []byte(config.Env.HmacKey))
	hmac256.Write([]byte(json))
	signature := hex.EncodeToString(hmac256.Sum(nil))

	return signature
}

func rsaDecode(json []byte) ([]byte, error) {
	spkiBlock, _ := pem.Decode([]byte(config.Env.PrivateKey))
	var spkiKey *rsa.PrivateKey
	priInterface, err := x509.ParsePKCS8PrivateKey(spkiBlock.Bytes)
	if err != nil {
		return nil, err
	}
	spkiKey = priInterface.(*rsa.PrivateKey)
	decryptSignature, err := spkiKey.Decrypt(nil, json, &rsa.OAEPOptions{Hash: crypto.SHA256})
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("don't decode rsa Data")
	}
	return decryptSignature, nil
}

func vaildTimestampCheck(originTimestamp string, currentTimestamp string) bool {
	current, err := strconv.Atoi(currentTimestamp)
	if err != nil {
		panic(err)
	}
	original, err := strconv.Atoi(originTimestamp)
	if err != nil {
		panic(err)
	}
	if (current - original) > 5*60*1000 {
		return false
	} else {
		return true
	}
}

// aes decode
func aesDecrypt(cipherText, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(cipherText)%aes.BlockSize != 0 {
		return nil, errors.New("cipherText is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(cipherText))
	mode.CryptBlocks(plaintext, cipherText)

	return pkcs7Unpad(plaintext)
}

// PKCS7 패딩 제거
func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("unpad: input is empty")
	}
	paddingLen := int(data[length-1])
	if paddingLen > length || paddingLen == 0 {
		return nil, errors.New("unpad: invalid padding")
	}
	for _, b := range data[length-paddingLen:] {
		if int(b) != paddingLen {
			return nil, errors.New("unpad: incorrect padding bytes")
		}
	}
	return data[:length-paddingLen], nil
}

var (
	rsaDecodeFunc           = rsaDecode
	aesDecryptFunc          = aesDecrypt
	currentTimestampFunc    = currentTimestamp
	hmacEncodeFunc          = hmacEncode
	vaildTimestampCheckFunc = vaildTimestampCheck
)

// 복호화 구현
// 1. base64 decode function 호출
// 2. 키를 rsa decode funcion 호출
// 2. 객체를 aes decode funcion 호출
// 3. 객체를 hmac encode function 호출 하여 넘어온 hmac_data와 비교 분석
// 4. 객체의 timestamp를 확인하여 timestamp vaild function 호출
// 5. 이후 기능 수행(struct에 mapping이 불가능 하다면 잘못된 정보가 넘어왔다 생각하고 error 발생.)
func decodePayload[T any](payload model.HybridPayload) (T, error) {
	currentTimestamp := currentTimestampFunc()

	//1. base64 decode
	encodedAesData, err := base64.StdEncoding.DecodeString(payload.EncryptedData)
	if err != nil {
		var t T
		return t, errors.New("base64 decode failed")
	}
	encodedAesKey, err := base64.StdEncoding.DecodeString(payload.EncryptedKey)
	if err != nil {
		var t T
		return t, errors.New("base64 decode failed")
	}
	//iv := payload.IV
	iv, err := base64.StdEncoding.DecodeString(payload.IV)
	if err != nil {
		var t T
		return t, errors.New("base64 decode failed")
	}

	// 2. 키를 rsa decode funcion 호출
	decodedAesKey, err := rsaDecodeFunc(encodedAesKey)
	if err != nil {
		var t T
		return t, err
	}

	// 3. 객체를 aes decode funcion 호출
	decodedData, err := aesDecryptFunc(encodedAesData, decodedAesKey, []byte(iv))
	if err != nil {
		var t T
		return t, err
	}

	var PayloadModelWithHMAC model.PayloadModelWithHMAC[T]

	json.Unmarshal(decodedData, &PayloadModelWithHMAC)

	hmac_data := PayloadModelWithHMAC.Hmac_data

	var payloadModelWithoutHMAC model.PayloadModelWithoutHMAC[T]
	json.Unmarshal(decodedData, &payloadModelWithoutHMAC)
	byt, err := json.Marshal(payloadModelWithoutHMAC)
	if err != nil {
		var t T
		return t, errors.New("json parsing failed")
	}
	// 4. 객체를 hmac encode function 호출 하여 넘어온 hmac_data와 비교 분석
	if hmacEncodeFunc(byt) != hmac_data {
		var t T
		return t, errors.New("integrity verification failed")
		//틀리므로 에러 발생 후 리턴
	}
	// 5. 객체의 timestamp를 확인하여 timestamp vaild function 호출
	if !vaildTimestampCheckFunc(payloadModelWithoutHMAC.Timestamp, currentTimestamp) {
		// 유효하지 않으므로 기한이 지난 요청이라 판단하고 에러 발생 후 리턴
		var t T
		return t, errors.New("your request has expired")
	}

	return PayloadModelWithHMAC.Data, nil
}
