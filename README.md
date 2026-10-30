# EasySlip SDK for Go

[![CI](https://github.com/EASYSLIP-CO-LTD/sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/EASYSLIP-CO-LTD/sdk-go/actions/workflows/ci.yml) [![MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Use the EasySlip API from Go to verify slips, manage receiving accounts and generate payment QR codes.

## Features

The client supports v2 verification and asynchronous jobs, bank accounts, account information and the legacy v1 API. Requests accept contexts, errors work with `errors.As` and `errors.Is`, and webhook helpers check signatures over the original body.

## Requirements

Go 1.26 or 1.27. The SDK uses only the standard library.

## Installation

```sh
go get github.com/EASYSLIP-CO-LTD/sdk-go
```

## Quick start

Set `EASYSLIP_API_KEY` or point `EASYSLIP_API_KEY_FILE` at a private key file. Prepare `payload.txt`, `slip.png` and `wallet-slip.png`. Replace the sample URL with a publicly reachable slip image URL. Each verification is a separate request and can consume quota.

```go
package main

import (
    "context"
    "encoding/base64"
    "log"
    "os"
    "strings"

    "github.com/EASYSLIP-CO-LTD/sdk-go"
)

func main() {
    client, err := easyslip.NewClient("")
    if err != nil { log.Fatal(err) }
    defer client.Close()
    ctx := context.Background()
    payload, err := os.ReadFile("payload.txt")
    if err != nil { log.Fatal(err) }
    image, err := os.ReadFile("slip.png")
    if err != nil { log.Fatal(err) }
    result, err := client.Verify.BankByPayload(ctx, strings.TrimSpace(string(payload)), nil)
    if err != nil { log.Fatal(err) }
    log.Print(result.Meta.RequestID)
    if _, err = client.Verify.BankByImage(ctx, image, nil); err != nil { log.Fatal(err) }
    if _, err = client.Verify.BankByURL(ctx, "https://example.com/slip.png", nil); err != nil { log.Fatal(err) }
    if _, err = client.Verify.BankByBase64(ctx, base64.StdEncoding.EncodeToString(image), nil); err != nil { log.Fatal(err) }
    if _, err = client.Verify.TruewalletByImage(ctx, "wallet-slip.png", nil); err != nil { log.Fatal(err) }
    if _, err = client.GetInfo(ctx); err != nil { log.Fatal(err) }
    if _, err = client.V1.GenerateQR(ctx, easyslip.QRRequest{Type: "PROMPTPAY", Msisdn: easyslip.Some("0812345678"), Amount: easyslip.Some(100.0)}); err != nil { log.Fatal(err) }
    if _, err = client.V1.VerifyBankByPayload(ctx, strings.TrimSpace(string(payload)), nil); err != nil { log.Fatal(err) }
}
```

Image methods accept `Image`, `[]byte`, a trusted local path string, or `io.Reader`. Do not pass user-supplied strings as paths. Payload methods expect the slip QR payload, not the transaction reference. URL methods ask the API to fetch an image. `GenerateQR` creates a payment QR code. Legacy methods are under `client.V1`.

## Configuration

| Option | Default | Purpose |
| --- | --- | --- |
| `NewClient(apiKey)` | EASYSLIP_API_KEY, then EASYSLIP_API_KEY_FILE | An explicit key takes precedence. |
| `WithTimeout` | 30 seconds | Bound each HTTP request. |
| `WithMaxRetries` | 2 | Set the retry budget. |
| `WithBaseURLs` | EASYSLIP_V2_BASE_URL / EASYSLIP_V1_BASE_URL, otherwise https://api.easyslip.com/v2 /v1 | Override v2 and v1 independently. |
| `WithHTTPClient` | SDK-owned http.Client | Supply a custom client; you retain ownership. |

```go
client, err := easyslip.NewClient("", easyslip.WithTimeout(15*time.Second), easyslip.WithMaxRetries(1), easyslip.WithHTTPClient(&http.Client{}))
if err != nil { log.Fatal(err) }
defer client.Close()
```

Pass a context with a deadline to bound the whole operation. `Jobs.WaitForJob` defaults to a two-second polling interval and a ten-minute overall timeout; `WaitOptions` overrides both. Close the client after use to release owned idle connections.

## Error handling

All SDK errors implement `EasySlipError`. Import `errors` for type checks and `context` for cancellation checks.

| Error class | HTTP status | When it happens | What to do |
| --- | --- | --- | --- |
| `AuthenticationError` | 401 | The key is missing or invalid. | Check the application key. |
| `PermissionDeniedError / QuotaExceededError` | 403 | The account or quota does not allow this call. | Check account access and quota. |
| `InvalidRequestError / InvalidImageError / InvalidUrlError` | 400, 413, 415, 422 | The API rejects the input. | Correct the input before retrying. |
| `DuplicateSlipError` | 400 | Duplicate detection finds an earlier slip. | Use the previous verification result. |
| `NotFoundError / SlipNotFoundError / SlipPendingError` | 404 | The resource is absent or the slip is not yet available. | Check the identifier or wait for the slip. |
| `ConflictError` | 409 | The operation conflicts with existing state. | Read the current state before changing it. |
| `RateLimitError` | 429 | The API is limiting requests. | Respect Retry-After. |
| `ServerError / ServiceUnavailableError` | 5xx | The API could not complete the request. | Check service status before repeating billed calls. |
| `NetworkError / RequestTimeoutError` | None | The connection fails or times out. | Check connectivity; avoid repeating billed calls blindly. |
| `InputError` | None | Local validation rejects the request. | Correct the input. |
| `WebhookSignatureError` | None | The signature does not authenticate the body. | Reject the webhook. |
| `JobWaitTimeoutError` | None | The overall polling deadline expires. | Inspect LastJob before polling again. |
| `BatchSubmitError` | Varies | A batch stops after partial progress. | Inspect completed results before resuming. |

```go
var auth *easyslip.AuthenticationError
var limited *easyslip.RateLimitError
var api *easyslip.ApiError
switch {
case errors.As(err, &auth):
    log.Print("Check the application API key")
case errors.As(err, &limited):
    log.Print("Wait before making another request")
case errors.As(err, &api):
    log.Printf("status=%d code=%v request_id=%v", api.Status, api.Code, api.RequestID)
case errors.Is(err, context.DeadlineExceeded):
    log.Print("The operation exceeded its deadline")
case err != nil:
    log.Printf("SDK request failed: %v", err)
}
```

Safe calls retry connection failures and selected 5xx responses. HTTP 429 can also be retried. Billed verification and job submissions do not retry read timeouts or 5xx after sending. Backoff includes jitter and respects Retry-After up to 60 seconds.

Public errors and response headers redact the configured key, including percent and JSON-escaped forms. Transport causes retain sanitized diagnostics only. `SensitiveRawBody()` deliberately returns the original body; keep it out of logs.

## Webhook signatures

Read the untouched request body into `rawBody`, then authenticate it before processing. In an HTTP handler, use:

```go
event, err := easyslip.ConstructWebhookEvent(rawBody, request.Header.Get(easyslip.WebhookSignatureHeader), os.Getenv("EASYSLIP_WEBHOOK_SECRET"))
if err != nil {
    http.Error(w, "Invalid signature", http.StatusBadRequest)
    return
}
_ = event
```

## Examples and API documentation

See [runnable examples](examples/README.md) for every operation and the [Go SDK reference](https://document.easyslip.com/en/sdk/go) for API fields.

## Support

For account or quota questions, contact [EasySlip](https://easyslip.com/). Report SDK bugs in [GitHub Issues](https://github.com/EASYSLIP-CO-LTD/sdk-go/issues), without keys or personal data. Report security issues privately through [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE), copyright 2026 EasySlip Co., Ltd.
