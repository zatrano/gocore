package idempotency

import (
	"encoding/json"
)

// HTTPStoredResponse, API idempotency önbelleğinde saklanan HTTP yanıtıdır.
type HTTPStoredResponse struct {
	StatusCode  int             `json:"status_code"`
	Body        json.RawMessage `json:"body"`
	ContentType string          `json:"content_type,omitempty"`
}

// Marshal, HTTPStoredResponse'u JSON bayta çevirir.
func (h HTTPStoredResponse) Marshal() ([]byte, error) {
	return json.Marshal(h)
}

// UnmarshalHTTPStoredResponse, önbellekten HTTP yanıtı okur.
func UnmarshalHTTPStoredResponse(raw []byte) (HTTPStoredResponse, error) {
	var h HTTPStoredResponse
	if err := json.Unmarshal(raw, &h); err != nil {
		return HTTPStoredResponse{}, err
	}
	return h, nil
}
