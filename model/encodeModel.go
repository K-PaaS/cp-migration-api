package model

type PayloadModelWithoutHMAC[T any] struct {
	Data      T      `json:"data"`
	Timestamp string `json:"timestamp"`
}

type PayloadModelWithHMAC[T any] struct {
	Data      T      `json:"data"`
	Timestamp string `json:"timestamp"`
	Hmac_data string `json:"hmac_data"`
}
type HybridPayload struct {
	EncryptedKey string `json:"key"`
	//IV            []byte `json:"iv"`
	IV            string `json:"iv"`
	EncryptedData string `json:"data"`
}
