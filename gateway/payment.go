package gateway

import (
	"errors"
	"net/http"
	"time"

	"github.com/robertkonga/yekonga-server-go/gateway/payment"
)

// NewPaymentProvider creates a payment provider for defaultConfig.Provider
// (flutterwave, paypal, pesapal, selcom, clickpesa, azampay, stripe,
// twocheckout, manual, or any gateway added via payment.RegisterFactory). It fails
// when the provider is unknown or its required credentials are missing.
func NewPaymentProvider(defaultConfig *payment.Config) (payment.Provider, error) {
	if defaultConfig == nil {
		return nil, errors.New("payment config is required")
	}

	if defaultConfig.HTTPClient == nil {
		defaultConfig.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return payment.New(*defaultConfig)
}

// GetManualPaymentProvider extracts the manual payment provider from a controller.
// It returns ErrNotSupported (from payment package) if the controller doesn't
// have a manual provider configured.
func GetManualPaymentProvider(ctrl *payment.Controller, name string) (ManualPaymentOps, error) {
	p, err := ctrl.Get(name)
	if err != nil {
		return nil, err
	}
	manual, ok := p.(ManualPaymentOps)
	if !ok {
		return nil, errors.New("provider does not support manual payment operations")
	}
	return manual, nil
}

// ManualPaymentOps is the interface for manual payment operations (not in the main
// Provider interface, since manual is optional).
type ManualPaymentOps interface {
	// UpdatePaymentStatus marks a payment as succeeded/failed and notes who did it.
	UpdatePaymentStatus(reference string, status payment.Status, recordedBy string, notes string) error
	// GetRecord returns the full payment record for admin inspection.
	GetRecord(reference string) (*payment.ManualPaymentRecord, error)
	// GetAllRecords returns all manual payment records.
	GetAllRecords() []payment.ManualPaymentRecord
}
