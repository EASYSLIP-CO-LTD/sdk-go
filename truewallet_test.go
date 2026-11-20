package easyslip

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

const walletBase = `"transactionId":"50012345678902","date":"2026-12-03T22:05:00+07:00","amount":200.25`

var walletShapes = map[string]string{
	"names":               `{` + walletBase + `,"sender":{"name":"นาย ผู้โอน ทดสอบ"},"receiver":{"name":"นาย รับเงิน ทดสอบ","phone":"08-xxx-4567"}}`,
	"phones":              `{` + walletBase + `,"sender":{"phone":"08-xxx-2345"},"receiver":{"phone":"08-xxx-9876"}}`,
	"receiver phone only": `{` + walletBase + `,"receiver":{"phone":"08-xxx-9876"}}`,
	"no parties":          `{` + walletBase + `}`,
}

func TestTrueWalletSlipShapesDecodeAndRoundTrip(t *testing.T) {
	for name, raw := range walletShapes {
		var slip TrueWalletSlip
		if err := json.Unmarshal([]byte(raw), &slip); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got, err := json.Marshal(slip)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		jsonEqual(t, got, []byte(raw))
		if slip.HasSender() != strings.Contains(raw, `"sender"`) || slip.HasReceiver() != strings.Contains(raw, `"receiver"`) {
			t.Fatalf("%s: presence wrong", name)
		}
	}
}

func TestTrueWalletOldCodeStillReadsFields(t *testing.T) {
	var old, fresh TrueWalletSlip
	_ = json.Unmarshal([]byte(walletShapes["names"]), &old)
	_ = json.Unmarshal([]byte(walletShapes["phones"]), &fresh)
	if old.Sender.Name == "" || old.Receiver.Name == "" || old.Receiver.Phone != "08-xxx-4567" {
		t.Fatal("older result lost its names")
	}
	if fresh.Sender.Name != "" || fresh.Sender.Phone != "08-xxx-2345" || fresh.Receiver.Phone != "08-xxx-9876" {
		t.Fatal("newer result fields")
	}
}

func walletClient(t *testing.T, status int, body string, seen *[]*http.Request, sent *[]string) *Client {
	t.Helper()
	c, err := NewClient("fixture-secret", WithMaxRetries(0), WithBaseURLs("https://api.test/v2", "https://api.test/v1"),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			*seen = append(*seen, r)
			data, _ := io.ReadAll(r.Body)
			*sent = append(*sent, string(data))
			return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTruewalletByPayloadV2(t *testing.T) {
	var seen []*http.Request
	var sent []string
	c := walletClient(t, 200, `{"success":true,"data":{"amountInSlip":200.25,"rawSlip":`+walletShapes["no parties"]+`}}`, &seen, &sent)
	defer c.Close()
	result, err := c.Verify.TruewalletByPayload(context.Background(), "wallet-qr", &VerifyOptions{Remark: Some("r"), CheckDuplicate: Some(true)})
	if err != nil {
		t.Fatal(err)
	}
	if seen[0].URL.Path != "/v2/verify/truewallet" {
		t.Fatal(seen[0].URL.Path)
	}
	jsonEqual(t, []byte(sent[0]), []byte(`{"payload":"wallet-qr","remark":"r","checkDuplicate":true}`))
	if result.RawSlip.HasSender() || result.RawSlip.HasReceiver() {
		t.Fatal("no parties expected")
	}
}

func TestVerifyTruemoneyByPayloadV1(t *testing.T) {
	var seen []*http.Request
	var sent []string
	c := walletClient(t, 200, `{"status":200,"data":`+walletShapes["phones"]+`}`, &seen, &sent)
	defer c.Close()
	slip, err := c.V1.VerifyTruemoneyByPayload(context.Background(), "wallet-qr", &V1VerifyOptions{CheckDuplicate: Some(false)})
	if err != nil {
		t.Fatal(err)
	}
	if seen[0].URL.Path != "/v1/verify/truewallet" {
		t.Fatal(seen[0].URL.Path)
	}
	jsonEqual(t, []byte(sent[0]), []byte(`{"payload":"wallet-qr","checkDuplicate":false}`))
	if slip.Receiver.Phone != "08-xxx-9876" {
		t.Fatal("receiver phone lost")
	}
}

func TestVerifyTruemoneyByPayloadDuplicateKeepsSlip(t *testing.T) {
	var seen []*http.Request
	var sent []string
	c := walletClient(t, 400, `{"status":400,"message":"duplicate_slip","data":`+walletShapes["phones"]+`}`, &seen, &sent)
	defer c.Close()
	_, err := c.V1.VerifyTruemoneyByPayload(context.Background(), "wallet-qr", &V1VerifyOptions{CheckDuplicate: Some(true)})
	var dup *DuplicateSlipError
	if !errors.As(err, &dup) {
		t.Fatalf("want DuplicateSlipError, got %v", err)
	}
	slip, ok := dup.Slip.(*TrueWalletSlip)
	if !ok {
		t.Fatalf("slip type %T", dup.Slip)
	}
	got, _ := json.Marshal(slip)
	jsonEqual(t, got, []byte(walletShapes["phones"]))
}

func TestTruewalletPayloadIsCheckedBeforeAnyRequest(t *testing.T) {
	var seen []*http.Request
	var sent []string
	c := walletClient(t, 200, `{}`, &seen, &sent)
	defer c.Close()
	for _, payload := range []string{"", strings.Repeat("x", 257)} {
		var bad *InputError
		if _, err := c.Verify.TruewalletByPayload(context.Background(), payload, nil); !errors.As(err, &bad) {
			t.Fatalf("v2 %d: %v", len(payload), err)
		}
		if _, err := c.V1.VerifyTruemoneyByPayload(context.Background(), payload, nil); !errors.As(err, &bad) {
			t.Fatalf("v1 %d: %v", len(payload), err)
		}
	}
	if len(seen) != 0 {
		t.Fatal("no request expected")
	}
}
