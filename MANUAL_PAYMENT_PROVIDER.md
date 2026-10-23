# Manual Payment Provider

The `manual` payment provider allows administrators to manually record cash and other physical payment methods (bank transfers, in-person payments, etc.) without integrating with external payment gateways.

## Overview

The manual provider:
- Creates payment records in a pending state when a payment is initiated
- Allows admins to mark payments as succeeded or failed after receiving them
- Tracks metadata like who recorded the payment and when
- Supports refunds on completed payments
- Stores all state in memory (suitable for local/testing use)

## Configuration

No credentials are required:

```go
ctrl, _ := payment.NewController(
    payment.Config{Provider: "manual"},
)
```

## Usage Flow

### 1. Create a Pending Payment

```go
res, err := ctrl.CreatePayment(ctx, "manual", payment.PaymentRequest{
    Reference:   "order_12345",      // your unique invoice/order ID
    Amount:      100.50,
    Currency:    "USD",
    Method:      "cash",             // or "bank_transfer", "check", etc.
    Description: "Payment for order #12345",
    Customer: payment.Customer{
        Name:  "John Doe",
        Email: "john@example.com",
    },
})
// res.Status == StatusPending
// res.CheckoutURL is empty (no redirect needed)
```

### 2. Verify Current Status

```go
result, err := ctrl.VerifyPayment(ctx, "manual", payment.VerifyRequest{
    Reference: "order_12345",
})
// result.Status is StatusPending, StatusSucceeded, StatusFailed, or StatusRefunded
```

### 3. Mark as Received (Admin Operation)

```go
p, _ := ctrl.Get("manual")
manual := p.(*payment.manualProvider)  // cast to access custom methods

err := manual.UpdatePaymentStatus(
    "order_12345",           // payment reference
    payment.StatusSucceeded, // mark as received
    "admin_user@company.com",  // who recorded this
    "Received cash payment in office", // notes
)
```

### 4. Refund if Needed

```go
refund, err := ctrl.Refund(ctx, "manual", payment.RefundRequest{
    Reference:         "order_12345",
    ProviderPaymentID: "order_12345",
    Amount:            50.00,           // 0 for full refund (not currently supported, use amount)
    Currency:          "USD",
    Reason:            "Customer requested partial refund",
})
// refund.Status == StatusRefunded
```

### 5. Cancel Pending Payment

```go
cancel, err := ctrl.CancelPayment(ctx, "manual", payment.CancelRequest{
    Reference: "order_12345",
    Reason:    "Customer withdrew, payment not received",
})
// cancel.Status == StatusFailed
// Only pending payments can be cancelled
```

## Admin API Helpers

### Get Single Payment Record

```go
record, err := manual.GetRecord("order_12345")
// record: *payment.ManualPaymentRecord
// {
//   Reference: "order_12345",
//   Amount: 100.50,
//   Currency: "USD",
//   Status: "SUCCESS",
//   Method: "cash",
//   RecordedBy: "admin_user@company.com",
//   RecordedAt: time.Time{...},
//   CompletedAt: &time.Time{...},
//   RefundedAt: nil,
//   Notes: "Received cash payment in office",
//   Metadata: {...},
// }
```

### List All Manual Payments

```go
all := manual.GetAllRecords()
for _, record := range all {
    fmt.Printf("%s: %v %s (recorded by %s)\n",
        record.Reference,
        record.Amount,
        record.Currency,
        record.RecordedBy,
    )
}
```

## Gateway Integration Example

Add to your `config.json`:

```json
{
  "payment": {
    "provider": "manual"
  }
}
```

In your server initialization:

```go
gatewayConfig := &payment.Config{
    Provider: "manual",
}
paymentProvider, err := gateway.NewPaymentProvider(gatewayConfig)
if err != nil {
    log.Fatal(err)
}
```

## REST API Integration

A typical implementation:

```go
// POST /api/payments — create a payment (returns pending status)
app.Post("/api/payments", func(req *yekonga.Request) *yekonga.Response {
    var payload struct {
        Reference string `json:"reference"`
        Amount    float64 `json:"amount"`
        Currency  string `json:"currency"`
        Method    string `json:"method"` // "cash", "bank_transfer", etc.
    }
    if err := req.BodyParser(&payload); err != nil {
        return req.BadRequest(err.Error())
    }

    res, err := paymentCtrl.CreatePayment(req.Context(), "manual", payment.PaymentRequest{
        Reference: payload.Reference,
        Amount:    payload.Amount,
        Currency:  payload.Currency,
        Method:    payload.Method,
    })
    if err != nil {
        return req.BadRequest(err.Error())
    }

    return req.JSON(res)
})

// GET /api/payments/:reference — check payment status
app.Get("/api/payments/:reference", func(req *yekonga.Request) *yekonga.Response {
    reference := req.Params("reference")
    result, err := paymentCtrl.VerifyPayment(req.Context(), "manual", payment.VerifyRequest{
        Reference: reference,
    })
    if err != nil {
        return req.BadRequest(err.Error())
    }
    return req.JSON(result)
})

// POST /api/admin/payments/:reference/mark-received — admin marks as paid
app.Post("/api/admin/payments/:reference/mark-received", func(req *yekonga.Request) *yekonga.Response {
    reference := req.Params("reference")
    var payload struct {
        Status string `json:"status"` // "SUCCESS" or "FAILED"
        Notes  string `json:"notes"`
    }
    if err := req.BodyParser(&payload); err != nil {
        return req.BadRequest(err.Error())
    }

    p, _ := paymentCtrl.Get("manual")
    manual := p.(*payment.manualProvider)

    status := payment.StatusSucceeded
    if payload.Status == "FAILED" {
        status = payment.StatusFailed
    }

    adminEmail := req.GetContext("user_email").(string) // from your auth middleware
    if err := manual.UpdatePaymentStatus(reference, status, adminEmail, payload.Notes); err != nil {
        return req.BadRequest(err.Error())
    }

    return req.JSON(map[string]string{"status": "ok"})
})

// GET /api/admin/payments — list all manual payments
app.Get("/api/admin/payments", func(req *yekonga.Request) *yekonga.Response {
    p, _ := paymentCtrl.Get("manual")
    manual := p.(*payment.manualProvider)
    records := manual.GetAllRecords()
    return req.JSON(records)
})
```

## Important Notes

- **In-Memory Storage**: All payment records live in the provider instance's memory. Restarting the service clears all records. For production, persist records to your database before acting on webhook equivalents.
- **No Webhooks**: The manual provider has no incoming webhook support; status changes are driven by admin API calls.
- **Thread-Safe**: The provider uses `sync.RWMutex` and is safe for concurrent requests.
- **Error Handling**: Verify refunds only work on succeeded payments; cancellations only on pending ones.

## Testing

See `gateway/payment/payment_providers_test.go::TestManualPaymentProvider` for comprehensive test coverage.

Run tests:

```bash
go test -v ./gateway/payment -run TestManualPaymentProvider
```
