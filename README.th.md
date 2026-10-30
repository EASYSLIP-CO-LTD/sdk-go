# EasySlip SDK สำหรับ Go

[![CI](https://github.com/EASYSLIP-CO-LTD/sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/EASYSLIP-CO-LTD/sdk-go/actions/workflows/ci.yml) [![MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

เรียก EasySlip API จาก Go เพื่อตรวจสลิป จัดการบัญชีรับเงิน และสร้าง QR สำหรับรับชำระเงิน

## ความสามารถ

รองรับการตรวจสลิป v2 งานแบบ asynchronous บัญชีรับเงิน ข้อมูลบัญชีผู้ใช้ และ API v1 เดิม ทุกคำขอรับ context ได้ ตรวจชนิด error ผ่าน `errors.As` และ `errors.Is` และมีฟังก์ชันตรวจลายเซ็น webhook จาก body ต้นฉบับ

## สิ่งที่ต้องใช้

Go 1.26 หรือ 1.27 ใช้เฉพาะ standard library ไม่มี dependency เพิ่มเติม

## ติดตั้ง

```sh
go get github.com/EASYSLIP-CO-LTD/sdk-go
```

## เริ่มใช้งาน

ตั้ง `EASYSLIP_API_KEY` หรือให้ `EASYSLIP_API_KEY_FILE` ชี้ไปยังไฟล์คีย์ที่เก็บเป็นความลับ เตรียม `payload.txt`, `slip.png` และ `wallet-slip.png` แล้วเปลี่ยน URL ตัวอย่างเป็น URL รูปสลิปที่เข้าถึงได้จากอินเทอร์เน็ต การตรวจแต่ละครั้งเป็นคนละคำขอและอาจใช้โควต้า

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

เมธอดรับรูปใช้ `Image`, `[]byte`, string ที่เป็น path ในเครื่อง หรือ `io.Reader` ได้ อย่าส่ง string จากผู้ใช้เข้าเมธอดไฟล์โดยตรง เมธอด payload รับข้อมูล QR ของสลิป ไม่ใช่เลขอ้างอิงธุรกรรม ส่วนเมธอด URL ให้ API ดาวน์โหลดรูป `GenerateQR` ใช้สร้าง QR รับเงิน และเมธอด v1 อยู่ใน `client.V1`

## ตั้งค่า

| ตัวเลือก | ค่าเริ่มต้น | การใช้งาน |
| --- | --- | --- |
| `NewClient(apiKey)` | อ่าน EASYSLIP_API_KEY ก่อน EASYSLIP_API_KEY_FILE | คีย์ที่ส่งเข้า constructor มีลำดับสูงสุด |
| `WithTimeout` | 30 วินาที | จำกัดเวลาของแต่ละคำขอ HTTP |
| `WithMaxRetries` | 2 | กำหนดจำนวนครั้งที่ลองใหม่ได้ |
| `WithBaseURLs` | อ่าน EASYSLIP_V2_BASE_URL / EASYSLIP_V1_BASE_URL หรือใช้ https://api.easyslip.com/v2 และ /v1 | เปลี่ยน URL ของ v2 และ v1 แยกกัน |
| `WithHTTPClient` | http.Client ที่ SDK สร้าง | ส่ง HTTP client ของคุณเอง โดยคุณเป็นผู้จัดการอายุการใช้งาน |

```go
client, err := easyslip.NewClient("", easyslip.WithTimeout(15*time.Second), easyslip.WithMaxRetries(1), easyslip.WithHTTPClient(&http.Client{}))
if err != nil { log.Fatal(err) }
defer client.Close()
```

ส่ง context ที่มี deadline เพื่อจำกัดเวลารวม `Jobs.WaitForJob` ตรวจสถานะทุกสองวินาทีและรอรวมไม่เกินสิบนาทีโดยปริยาย ปรับได้ด้วย `WaitOptions` เมื่อใช้งานเสร็จให้ปิด client เพื่อคืน connection ที่ SDK เป็นเจ้าของ

## จัดการข้อผิดพลาด

ข้อผิดพลาดของ SDK ใช้ interface `EasySlipError` ตรวจชนิดด้วย package `errors` และตรวจการยกเลิกผ่าน package `context`

| ชนิด error | HTTP status | เกิดเมื่อใด | วิธีรับมือ |
| --- | --- | --- | --- |
| `AuthenticationError` | 401 | ไม่มีคีย์หรือคีย์ไม่ถูกต้อง | ตรวจคีย์ของแอปพลิเคชัน |
| `PermissionDeniedError / QuotaExceededError` | 403 | สิทธิ์บัญชีหรือโควต้าไม่เพียงพอ | ตรวจสิทธิ์และโควต้าคงเหลือ |
| `InvalidRequestError / InvalidImageError / InvalidUrlError` | 400, 413, 415, 422 | API ไม่รับข้อมูลที่ส่งไป | แก้ข้อมูลก่อนส่งใหม่ |
| `DuplicateSlipError` | 400 | พบสลิปที่เคยตรวจแล้ว | ใช้ผลตรวจครั้งก่อน |
| `NotFoundError / SlipNotFoundError / SlipPendingError` | 404 | ไม่พบข้อมูลหรือสลิปยังไม่พร้อมให้ตรวจ | ตรวจรหัสข้อมูลหรือรอแล้วตรวจสลิปอีกครั้ง |
| `ConflictError` | 409 | คำขอขัดกับสถานะปัจจุบัน | อ่านสถานะล่าสุดก่อนแก้ไข |
| `RateLimitError` | 429 | จำนวนคำขอเกินที่ API รองรับ | รอตามค่า Retry-After |
| `ServerError / ServiceUnavailableError` | 5xx | API ประมวลผลคำขอไม่สำเร็จ | ตรวจสถานะบริการก่อนส่งคำขอที่คิดโควต้าซ้ำ |
| `NetworkError / RequestTimeoutError` | ไม่มี | เชื่อมต่อไม่สำเร็จหรือเกินเวลารอ | ตรวจการเชื่อมต่อและผลเดิมก่อนส่งคำขอที่คิดโควต้าซ้ำ |
| `InputError` | ไม่มี | ข้อมูลไม่ผ่านการตรวจใน SDK | แก้ข้อมูลที่ส่งเข้า SDK |
| `WebhookSignatureError` | ไม่มี | ลายเซ็นไม่ตรงกับ body | ปฏิเสธ webhook นี้ |
| `JobWaitTimeoutError` | ไม่มี | รอผลรวมเกินเวลาที่กำหนด | ตรวจ LastJob ก่อนเริ่มรอใหม่ |
| `BatchSubmitError` | แล้วแต่คำขอ | ส่ง batch ได้บางส่วนแล้วหยุด | ตรวจรายการที่ส่งสำเร็จก่อนทำต่อ |

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

คำขอที่อ่านข้อมูลอย่างปลอดภัยจะลองใหม่เมื่อเชื่อมต่อไม่สำเร็จหรือพบ 5xx บางกรณี รวมถึงอาจลองใหม่เมื่อพบ 429 การตรวจสลิปและส่งงานที่อาจคิดโควต้าจะไม่ลองใหม่หลังส่งคำขอแล้วเกิด read timeout หรือ 5xx ระยะรอมี jitter และใช้ค่า Retry-After ได้สูงสุด 60 วินาที

error และ response header ที่เปิดเผยจะปิดบังคีย์ รวมถึงรูปแบบ percent encoding และ JSON escape สาเหตุจาก transport เก็บเฉพาะข้อมูลที่ล้างแล้ว `SensitiveRawBody()` คืน body ต้นฉบับโดยตั้งใจ จึงไม่ควรนำไปลง log

## ตรวจลายเซ็น webhook

อ่าน body ต้นฉบับลง `rawBody` โดยไม่แปลงข้อมูล แล้วตรวจลายเซ็นก่อนประมวลผล ตัวอย่างภายใน HTTP handler:

```go
event, err := easyslip.ConstructWebhookEvent(rawBody, request.Header.Get(easyslip.WebhookSignatureHeader), os.Getenv("EASYSLIP_WEBHOOK_SECRET"))
if err != nil {
    http.Error(w, "Invalid signature", http.StatusBadRequest)
    return
}
_ = event
```

## ตัวอย่างและเอกสาร API

ดู [ตัวอย่างที่รันได้](examples/README.md) สำหรับทุก operation และ [คู่มือ Go SDK](https://document.easyslip.com/th/sdk/go) สำหรับฟิลด์ API

## ติดต่อและแจ้งปัญหา

ปัญหาบัญชีหรือโควต้าติดต่อ [EasySlip](https://easyslip.com/) แจ้งบั๊ก SDK ผ่าน [GitHub Issues](https://github.com/EASYSLIP-CO-LTD/sdk-go/issues) โดยไม่ใส่คีย์หรือข้อมูลส่วนบุคคล ช่องทางแจ้งช่องโหว่อยู่ใน [SECURITY.md](SECURITY.md)

## สัญญาอนุญาต

[MIT](LICENSE) ลิขสิทธิ์ 2026 EasySlip Co., Ltd.
