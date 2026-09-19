package paystack

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func VerifyWebhookSignature(secretKey string, rawBody []byte, signature string) bool {
	if secretKey == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha512.New, []byte(secretKey))
	_, _ = mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

type WebhookEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type ChargeSuccessData struct {
	Reference string `json:"reference"`
	Amount    int64  `json:"amount"`
	Status    string `json:"status"`
}

func ParseChargeSuccess(raw json.RawMessage) (ChargeSuccessData, error) {
	var d ChargeSuccessData
	if err := json.Unmarshal(raw, &d); err != nil {
		return ChargeSuccessData{}, fmt.Errorf("parse charge data: %w", err)
	}
	return d, nil
}
