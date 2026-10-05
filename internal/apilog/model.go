package apilog

import (
	"encoding/json"
	"time"
)

const (
	InboundCreate  = "inbound_create"
	OutboundCreate = "outbound_create"
	InboundCheck   = "inbound_check"
	OutboundCheck  = "outbound_check"
	Callback       = "callback"
)

type Log struct {
	ID            int64           `json:"id"`
	TransactionID int64           `json:"transaction_id"`
	Type          string          `json:"type"`
	Method        string          `json:"method"`
	URL           string          `json:"url"`
	RequestBody   json.RawMessage `json:"request_body"`
	ResponseBody  json.RawMessage `json:"response_body"`
	StatusCode    int             `json:"status_code"`
	DurationMs    int             `json:"duration_ms"`
	CreatedAt     time.Time       `json:"created_at"`
}
