package iyzico

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// WebhookPayload, iyzico webhook JSON gövdesidir (Direct / HPP / Subscription).
type WebhookPayload struct {
	PaymentConversationID     string `json:"paymentConversationId"`
	MerchantID                string `json:"merchantId"`
	PaymentID                 string `json:"paymentId"`
	Status                    string `json:"status"`
	IyziReferenceCode         string `json:"iyziReferenceCode"`
	IyziEventType             string `json:"iyziEventType"`
	IyziEventTime             int64  `json:"iyziEventTime"`
	IyziPaymentID             any    `json:"iyziPaymentId"`
	Token                     string `json:"token"`
	OrderReferenceCode        string `json:"orderReferenceCode"`
	CustomerReferenceCode     string `json:"customerReferenceCode"`
	SubscriptionReferenceCode string `json:"subscriptionReferenceCode"`
}

// UnmarshalJSON, paymentId / iyziPaymentId alanlarını string olarak normalize eder.
func (p *WebhookPayload) UnmarshalJSON(data []byte) error {
	type alias WebhookPayload
	var raw struct {
		alias
		PaymentID     any `json:"paymentId"`
		IyziPaymentID any `json:"iyziPaymentId"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = WebhookPayload(raw.alias)
	p.PaymentID = stringifyJSONValue(raw.PaymentID)
	if p.IyziPaymentID == nil {
		p.IyziPaymentID = raw.IyziPaymentID
	}
	return nil
}

func stringifyJSONValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(t)
	}
}

// IyziPaymentIDString, HPP imza doğrulaması için iyziPaymentId değerini döner.
func (p WebhookPayload) IyziPaymentIDString() string {
	return stringifyJSONValue(p.IyziPaymentID)
}

// VerifyWebhookSignatureV3, X-IYZ-SIGNATURE-V3 başlığını doğrular.
// https://docs.iyzico.com/en/advanced/webhook
func VerifyWebhookSignatureV3(secretKey, signature string, payload WebhookPayload) bool {
	secretKey = strings.TrimSpace(secretKey)
	signature = strings.TrimSpace(signature)
	if secretKey == "" || signature == "" {
		return false
	}
	message := webhookMessage(secretKey, payload)
	if message == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secretKey))
	_, _ = mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(signature)))
}

func webhookMessage(secretKey string, payload WebhookPayload) string {
	if strings.TrimSpace(payload.SubscriptionReferenceCode) != "" {
		eventType := payload.IyziEventType
		return payload.MerchantID + secretKey + eventType +
			payload.SubscriptionReferenceCode + payload.OrderReferenceCode + payload.CustomerReferenceCode
	}
	if strings.TrimSpace(payload.Token) != "" {
		return secretKey + payload.IyziEventType + payload.IyziPaymentIDString() +
			payload.Token + payload.PaymentConversationID + payload.Status
	}
	if payload.IyziEventType == "" || payload.PaymentID == "" ||
		payload.PaymentConversationID == "" || payload.Status == "" {
		return ""
	}
	return secretKey + payload.IyziEventType + payload.PaymentID +
		payload.PaymentConversationID + payload.Status
}
