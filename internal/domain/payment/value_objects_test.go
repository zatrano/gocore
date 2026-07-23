package payment_test

import (
	"testing"

	domainpayment "github.com/zatrano/gocore/internal/domain/payment"
)

func TestMaskCardHolder(t *testing.T) {
	if got := domainpayment.MaskCardHolder("Ali Veli"); got != "A** V***" {
		t.Fatalf("got %q", got)
	}
	if domainpayment.MaskCardHolder("") != "" {
		t.Fatal("empty holder")
	}
}

func TestCardDisplay(t *testing.T) {
	if got := domainpayment.CardDisplay("526911", "1234"); got != "52****1234" {
		t.Fatalf("got %q", got)
	}
}
