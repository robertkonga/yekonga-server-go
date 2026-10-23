package payment

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Manual payment provider handles cash and other physical payment methods.
// Admin manually records when a payment is received; the provider tracks state
// in memory (manual.records) and returns it on verification.
//
// Credentials: none required (config.Credentials is unused).

func init() {
	RegisterFactory(ProviderManual, func(cfg Config) (Provider, error) {
		return &manualProvider{
			cfg:     cfg,
			records: make(map[string]*ManualPaymentRecord),
		}, nil
	})
}

// ManualPaymentRecord stores the state of a manually-recorded payment.
type ManualPaymentRecord struct {
	Reference     string
	Amount        float64
	Currency      string
	Status        Status
	Method        string
	Description   string
	RecordedBy    string
	RecordedAt    time.Time
	CompletedAt   *time.Time
	RefundedAt    *time.Time
	FailureReason string
	Notes         string
	Metadata      map[string]string
}

type manualProvider struct {
	cfg     Config
	mu      sync.RWMutex
	records map[string]*ManualPaymentRecord
}

func (m *manualProvider) Name() string { return ProviderManual }

// CreatePayment creates a pending manual payment record and returns no checkout URL.
// The admin later calls VerifyPayment with status=SUCCESS to mark it complete.
func (m *manualProvider) CreatePayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.records[req.Reference]; exists {
		return nil, fmt.Errorf("%w: manual payment %q already exists", ErrInvalidRequest, req.Reference)
	}

	record := &ManualPaymentRecord{
		Reference:   req.Reference,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Status:      StatusPending,
		Method:      firstNonEmpty(req.Method, req.Channel, "cash"),
		Description: req.Description,
		RecordedAt:  time.Now(),
		Metadata:    req.Metadata,
	}

	m.records[req.Reference] = record

	return &PaymentResponse{
		Provider:    m.Name(),
		Reference:   req.Reference,
		Status:      StatusPending,
		CheckoutURL: "", // no URL for manual payment
		Raw: map[string]any{
			"recorded_at": record.RecordedAt,
			"method":      record.Method,
		},
	}, nil
}

// VerifyPayment returns the current state of a manual payment record.
// To update status, use UpdatePaymentStatus on the provider directly.
func (m *manualProvider) VerifyPayment(ctx context.Context, req VerifyRequest) (*PaymentResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, ok := m.records[req.Reference]
	if !ok {
		return nil, fmt.Errorf("%w: manual payment %q not found", ErrInvalidRequest, req.Reference)
	}

	result := &PaymentResult{
		Provider:    m.Name(),
		Reference:   record.Reference,
		Status:      record.Status,
		Amount:      record.Amount,
		Currency:    record.Currency,
		Method:      record.Method,
		FailureReason: record.FailureReason,
		Raw: map[string]any{
			"recorded_at": record.RecordedAt,
			"completed_at": record.CompletedAt,
			"refunded_at": record.RefundedAt,
			"notes": record.Notes,
		},
	}

	if record.CompletedAt != nil {
		result.ProviderPaymentID = req.Reference // use reference as payment id for refunds
	}

	return result, nil
}

// PushUSSD is not supported for manual payments.
func (m *manualProvider) PushUSSD(ctx context.Context, req PushRequest) (*PushResponse, error) {
	return nil, fmt.Errorf("%w: manual provider does not support push/USSD", ErrNotSupported)
}

// CancelPayment cancels a pending manual payment.
func (m *manualProvider) CancelPayment(ctx context.Context, req CancelRequest) (*CancelResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.records[req.Reference]
	if !ok {
		return nil, fmt.Errorf("%w: manual payment %q not found", ErrInvalidRequest, req.Reference)
	}

	if record.Status != StatusPending {
		return nil, fmt.Errorf("%w: only pending manual payments can be cancelled", ErrInvalidRequest)
	}

	record.Status = StatusFailed
	record.FailureReason = firstNonEmpty(req.Reason, "cancelled")

	return &CancelResult{
		Provider:      m.Name(),
		Reference:     req.Reference,
		Status:        StatusFailed,
		FailureReason: record.FailureReason,
		Raw: map[string]any{
			"cancelled_at": time.Now(),
		},
	}, nil
}

// ListPayments is not supported for manual payments.
func (m *manualProvider) ListPayments(ctx context.Context, req ListRequest) (*ListResult, error) {
	return nil, fmt.Errorf("%w: manual provider does not support listing payments", ErrNotSupported)
}

// Refund marks a payment as refunded; the caller owns tracking the refund in their DB.
func (m *manualProvider) Refund(ctx context.Context, req RefundRequest) (*RefundResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.records[req.Reference]
	if !ok {
		return nil, fmt.Errorf("%w: manual payment %q not found", ErrInvalidRequest, req.Reference)
	}

	if record.Status != StatusSucceeded {
		return nil, fmt.Errorf("%w: only succeeded manual payments can be refunded", ErrInvalidRequest)
	}

	now := time.Now()
	record.Status = StatusRefunded
	record.RefundedAt = &now

	return &RefundResult{
		Provider:          m.Name(),
		ProviderRefundID:  fmt.Sprintf("manual-refund-%d", now.Unix()),
		ProviderPaymentID: req.ProviderPaymentID,
		Status:            StatusRefunded,
		Amount:            req.Amount,
		Currency:          req.Currency,
		Reason:            req.Reason,
		Raw: map[string]any{
			"refunded_at": now,
		},
	}, nil
}

// GetRefund is not supported for manual payments.
func (m *manualProvider) GetRefund(ctx context.Context, req RefundQuery) (*RefundResult, error) {
	return nil, fmt.Errorf("%w: manual provider does not support refund lookup", ErrNotSupported)
}

// ListRefunds is not supported for manual payments.
func (m *manualProvider) ListRefunds(ctx context.Context, req RefundQuery) (*RefundList, error) {
	return nil, fmt.Errorf("%w: manual provider does not support listing refunds", ErrNotSupported)
}

// ParseWebhook is not supported for manual payments (no incoming webhooks).
func (m *manualProvider) ParseWebhook(ctx context.Context, r *http.Request) (*WebhookEvent, error) {
	return nil, fmt.Errorf("%w: manual provider does not handle webhooks", ErrNotSupported)
}

// UpdatePaymentStatus is a custom method (not in the Provider interface) for admins
// to manually mark a payment as succeeded or failed.
func (m *manualProvider) UpdatePaymentStatus(reference string, status Status, recordedBy string, notes string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.records[reference]
	if !ok {
		return fmt.Errorf("%w: manual payment %q not found", ErrInvalidRequest, reference)
	}

	if status == StatusSucceeded || status == StatusFailed {
		record.Status = status
		record.RecordedBy = recordedBy
		record.Notes = notes
		now := time.Now()
		record.CompletedAt = &now
		return nil
	}

	return fmt.Errorf("%w: can only update to SUCCESS or FAILED", ErrInvalidRequest)
}

// GetRecord returns the full ManualPaymentRecord for inspection/editing by admins.
func (m *manualProvider) GetRecord(reference string) (*ManualPaymentRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, ok := m.records[reference]
	if !ok {
		return nil, fmt.Errorf("%w: manual payment %q not found", ErrInvalidRequest, reference)
	}

	// return a copy to prevent external mutation
	copy := *record
	return &copy, nil
}

// GetAllRecords returns all manual payment records (useful for admin dashboards).
func (m *manualProvider) GetAllRecords() []ManualPaymentRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records := make([]ManualPaymentRecord, 0, len(m.records))
	for _, r := range m.records {
		records = append(records, *r)
	}
	return records
}
