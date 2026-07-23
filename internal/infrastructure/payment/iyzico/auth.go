package iyzico

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

func authorizationHeader(apiKey, secretKey, uriPath string, body []byte) (auth, rnd string) {
	rnd = strconv.FormatInt(time.Now().UnixNano(), 10)
	payload := rnd + uriPath + string(body)
	mac := hmac.New(sha256.New, []byte(secretKey))
	_, _ = mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	plain := fmt.Sprintf("apiKey:%s&randomKey:%s&signature:%s", apiKey, rnd, signature)
	return "IYZWSv2 " + base64.StdEncoding.EncodeToString([]byte(plain)), rnd
}
