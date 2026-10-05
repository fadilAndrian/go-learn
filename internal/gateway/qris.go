// Package gateway = client outbound ke payment gateway, QRIS MPM versi BI SNAP (service code 47).
package gateway

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrGateway = errors.New("payment gateway error")

var wib = time.FixedZone("WIB", 7*3600)

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Base struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}

type GenerateReq struct {
	PartnerReferenceNo string `json:"partnerReferenceNo"`
	MerchantID         string `json:"merchantId"`
	TerminalID         string `json:"terminalId"`
	Amount             Amount `json:"amount"`
}

type GenerateRes struct {
	Base
	ReferenceNo        string `json:"referenceNo"`
	PartnerReferenceNo string `json:"partnerReferenceNo"`
	QRContent          string `json:"qrContent"`
	TerminalID         string `json:"terminalId"`
}

type QueryReq struct {
	OriginalReferenceNo        string `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string `json:"originalPartnerReferenceNo"`
	ServiceCode                string `json:"serviceCode"`
	MerchantID                 string `json:"merchantId"`
}

type QueryRes struct {
	Base
	OriginalReferenceNo        string `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string `json:"originalPartnerReferenceNo"`
	LatestTransactionStatus    string `json:"latestTransactionStatus"` // 00 paid, 03 pending, 05/06 failed
	TransactionStatusDesc      string `json:"transactionStatusDesc"`
}

type RefundReq struct {
	MerchantID                 string `json:"merchantId"`
	OriginalReferenceNo        string `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string `json:"originalPartnerReferenceNo"`
	PartnerRefundNo            string `json:"partnerRefundNo"`
	RefundAmount               Amount `json:"refundAmount"`
	Reason                     string `json:"reason,omitempty"`
}

type RefundRes struct {
	Base
	OriginalReferenceNo        string `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string `json:"originalPartnerReferenceNo"`
	RefundNo                   string `json:"refundNo"`
	PartnerRefundNo            string `json:"partnerRefundNo"`
	RefundAmount               Amount `json:"refundAmount"`
	RefundTime                 string `json:"refundTime"`
}

// NotifyReq = body webhook payment notify dari PG ke kita.
type NotifyReq struct {
	OriginalReferenceNo        string `json:"originalReferenceNo"`
	OriginalPartnerReferenceNo string `json:"originalPartnerReferenceNo"`
	LatestTransactionStatus    string `json:"latestTransactionStatus"`
	TransactionStatusDesc      string `json:"transactionStatusDesc"`
	Amount                     Amount `json:"amount"`
}

// Call = rekaman satu request ke PG, untuk dicatat ke api log.
type Call struct {
	URL    string
	Req    json.RawMessage
	Res    json.RawMessage // nil kalau response bukan JSON
	Status int
	Dur    time.Duration
}

type Client struct {
	baseURL, partnerID string
	http               *http.Client
}

func NewClient(baseURL, partnerID string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), partnerID: partnerID, http: &http.Client{Timeout: 10 * time.Second}}
}

// post kirim JSON, sukses kalau responseCode berawalan "200" (konvensi SNAP: 200xxxx).
// ponytail: tanpa B2B access token & X-SIGNATURE asimetris; tambahkan saat konek PG sungguhan.
func (c *Client) post(path string, req, out any) (Call, error) {
	body, _ := json.Marshal(req)
	call := Call{URL: c.baseURL + path, Req: body}

	hr, err := http.NewRequest(http.MethodPost, call.URL, bytes.NewReader(body))
	if err != nil {
		return call, fmt.Errorf("%w: %v", ErrGateway, err)
	}
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("X-TIMESTAMP", time.Now().In(wib).Format("2006-01-02T15:04:05-07:00"))
	hr.Header.Set("X-PARTNER-ID", c.partnerID)
	hr.Header.Set("X-EXTERNAL-ID", strconv.FormatInt(time.Now().UnixNano(), 10))
	hr.Header.Set("CHANNEL-ID", "95221")

	start := time.Now()
	resp, err := c.http.Do(hr)
	call.Dur = time.Since(start)
	if err != nil {
		return call, fmt.Errorf("%w: %v", ErrGateway, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	call.Status = resp.StatusCode
	if json.Valid(raw) {
		call.Res = raw
	}

	var base Base
	if err := json.Unmarshal(raw, &base); err != nil {
		return call, fmt.Errorf("%w: invalid response (http %d)", ErrGateway, resp.StatusCode)
	}
	if !strings.HasPrefix(base.ResponseCode, "200") {
		return call, fmt.Errorf("%w: %s %s", ErrGateway, base.ResponseCode, base.ResponseMessage)
	}
	return call, json.Unmarshal(raw, out)
}

// ponytail: terminalId tetap, jadikan config per merchant kalau PG mewajibkan.
const terminalID = "TERM01"

func (c *Client) Generate(merchantID, partnerRef, amount string) (GenerateRes, Call, error) {
	var res GenerateRes
	call, err := c.post("/v1.0/qr/qr-mpm-generate", GenerateReq{
		PartnerReferenceNo: partnerRef, MerchantID: merchantID, TerminalID: terminalID,
		Amount: Amount{Value: amount, Currency: "IDR"},
	}, &res)
	return res, call, err
}

func (c *Client) Query(merchantID, providerRef, partnerRef string) (QueryRes, Call, error) {
	var res QueryRes
	call, err := c.post("/v1.0/qr/qr-mpm-query", QueryReq{
		OriginalReferenceNo: providerRef, OriginalPartnerReferenceNo: partnerRef,
		ServiceCode: "47", MerchantID: merchantID,
	}, &res)
	return res, call, err
}

func (c *Client) Refund(merchantID, providerRef, partnerRef, refundNo, amount, reason string) (RefundRes, Call, error) {
	var res RefundRes
	call, err := c.post("/v1.0/qr/qr-mpm-refund", RefundReq{
		MerchantID: merchantID, OriginalReferenceNo: providerRef, OriginalPartnerReferenceNo: partnerRef,
		PartnerRefundNo: refundNo, RefundAmount: Amount{Value: amount, Currency: "IDR"}, Reason: reason,
	}, &res)
	return res, call, err
}

// Sign = hex HMAC-SHA256 body webhook. ponytail: SNAP asli pakai signature asimetris.
func Sign(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

func ValidSignature(secret string, body []byte, sig string) bool {
	return secret != "" && hmac.Equal([]byte(Sign(secret, body)), []byte(sig))
}
