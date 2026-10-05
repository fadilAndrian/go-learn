package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type fakeTxn struct {
	refNo, status, amount string // status: SNAP code 03 pending, 00 paid, 06 failed, 04 refunded
}

// RegisterFake memasang PG palsu (response sesuai spec SNAP QRIS MPM) di /fake-pg.
// State di memori; hilang saat restart. Bayar manual lewat POST /fake-pg/simulate-pay/:partnerRef?status=00|06
// yang menembak webhook (ditandatangani) ke notifyURL.
func RegisterFake(app *fiber.App, notifyURL, secret string) {
	var mu sync.Mutex
	txns := map[string]*fakeTxn{}

	fail := func(c fiber.Ctx, status int, code, msg string) error {
		return c.Status(status).JSON(Base{ResponseCode: code, ResponseMessage: msg})
	}

	g := app.Group("/fake-pg")

	g.Post("/v1.0/qr/qr-mpm-generate", func(c fiber.Ctx) error {
		var req GenerateReq
		if err := c.Bind().Body(&req); err != nil || req.PartnerReferenceNo == "" {
			return fail(c, 400, "4004700", "Invalid Mandatory Field")
		}
		mu.Lock()
		defer mu.Unlock()
		if _, dup := txns[req.PartnerReferenceNo]; dup {
			return fail(c, 409, "4094700", "Conflict")
		}
		ref := "PG-" + req.PartnerReferenceNo
		txns[req.PartnerReferenceNo] = &fakeTxn{refNo: ref, status: "03", amount: req.Amount.Value}
		return c.JSON(GenerateRes{
			Base:        Base{"2004700", "Successful"},
			ReferenceNo: ref, PartnerReferenceNo: req.PartnerReferenceNo, TerminalID: req.TerminalID,
			QRContent: "00020101021226670016COM.NOBUBANK.WWW01189360050300000879140214" + req.PartnerReferenceNo + "5204581253033605802ID5909FAKE QRIS6007JAKARTA6304ABCD",
		})
	})

	g.Post("/v1.0/qr/qr-mpm-query", func(c fiber.Ctx) error {
		var req QueryReq
		if err := c.Bind().Body(&req); err != nil {
			return fail(c, 400, "4005100", "Invalid Mandatory Field")
		}
		mu.Lock()
		defer mu.Unlock()
		t := txns[req.OriginalPartnerReferenceNo]
		if t == nil {
			return fail(c, 404, "4045101", "Transaction Not Found")
		}
		desc := map[string]string{"03": "Pending", "00": "Success", "06": "Failed", "04": "Refunded"}[t.status]
		return c.JSON(QueryRes{
			Base:                Base{"2005100", "Successful"},
			OriginalReferenceNo: t.refNo, OriginalPartnerReferenceNo: req.OriginalPartnerReferenceNo,
			LatestTransactionStatus: t.status, TransactionStatusDesc: desc,
		})
	})

	g.Post("/v1.0/qr/qr-mpm-refund", func(c fiber.Ctx) error {
		var req RefundReq
		if err := c.Bind().Body(&req); err != nil {
			return fail(c, 400, "4007800", "Invalid Mandatory Field")
		}
		mu.Lock()
		defer mu.Unlock()
		t := txns[req.OriginalPartnerReferenceNo]
		if t == nil {
			return fail(c, 404, "4047800", "Transaction Not Found")
		}
		if t.status != "00" {
			return fail(c, 403, "4037814", "Transaction Not Permitted")
		}
		t.status = "04"
		return c.JSON(RefundRes{
			Base:                Base{"2007800", "Successful"},
			OriginalReferenceNo: t.refNo, OriginalPartnerReferenceNo: req.OriginalPartnerReferenceNo,
			RefundNo: "RF-" + req.PartnerRefundNo, PartnerRefundNo: req.PartnerRefundNo,
			RefundAmount: req.RefundAmount, RefundTime: time.Now().In(wib).Format("2006-01-02T15:04:05-07:00"),
		})
	})

	g.Post("/simulate-pay/:partnerRef", func(c fiber.Ctx) error {
		status := c.Query("status", "00")
		if status != "00" && status != "06" {
			return fiber.NewError(fiber.StatusBadRequest, "status must be 00 or 06")
		}
		mu.Lock()
		t := txns[c.Params("partnerRef")]
		if t == nil {
			mu.Unlock()
			return fiber.NewError(fiber.StatusNotFound, "unknown partnerRef")
		}
		t.status = status
		n := NotifyReq{
			OriginalReferenceNo: t.refNo, OriginalPartnerReferenceNo: c.Params("partnerRef"),
			LatestTransactionStatus: status, Amount: Amount{Value: t.amount, Currency: "IDR"},
		}
		mu.Unlock()

		body, _ := json.Marshal(n)
		hr, _ := http.NewRequest(http.MethodPost, notifyURL, bytes.NewReader(body))
		hr.Header.Set("Content-Type", "application/json")
		hr.Header.Set("X-SIGNATURE", Sign(secret, body))
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(hr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadGateway, "notify failed: "+err.Error())
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		return c.JSON(fiber.Map{"notify_status": resp.StatusCode, "notify_response": json.RawMessage(out)})
	})
}
