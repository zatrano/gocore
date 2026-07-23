package iyzico

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func signDirect(secret, eventType, paymentID, conversationID, status string) string {
	msg := secret + eventType + paymentID + conversationID + status
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignatureV3_Direct(t *testing.T) {
	secret := "sandbox-secret"
	payload := WebhookPayload{
		IyziEventType:         "THREE_DS_AUTH",
		PaymentID:             "987654",
		PaymentConversationID: "conv-abc",
		Status:                "SUCCESS",
	}
	sig := signDirect(secret, payload.IyziEventType, payload.PaymentID, payload.PaymentConversationID, payload.Status)
	if !VerifyWebhookSignatureV3(secret, sig, payload) {
		t.Fatal("direct format signature should verify")
	}
	if VerifyWebhookSignatureV3(secret, "bad", payload) {
		t.Fatal("invalid signature should fail")
	}
}

func TestVerifyWebhookSignatureV3_HPP(t *testing.T) {
	secret := "sandbox-secret"
	payload := WebhookPayload{
		IyziEventType:         "CHECKOUT_FORM_AUTH",
		IyziPaymentID:         jsonNumber("555"),
		Token:                 "token-1",
		PaymentConversationID: "conv-hpp",
		Status:                "SUCCESS",
	}
	msg := secret + payload.IyziEventType + payload.IyziPaymentIDString() +
		payload.Token + payload.PaymentConversationID + payload.Status
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))
	if !VerifyWebhookSignatureV3(secret, sig, payload) {
		t.Fatal("HPP format signature should verify")
	}
}

func jsonNumber(s string) any { return s }
