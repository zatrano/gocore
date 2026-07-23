package moka

import "testing"

func TestNormalizeBinNumber(t *testing.T) {
	tests := []struct{ in, want string }{
		{" 526911 ", "526911"},
		{"5269110012345678", "52691100"},
		{"5526-0800-0000-0006", "55260800"},
	}
	for _, tc := range tests {
		if got := NormalizeBinNumber(tc.in); got != tc.want {
			t.Fatalf("NormalizeBinNumber(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBankCardInformationData_validate(t *testing.T) {
	f := false
	if err := (BankCardInformationData{IsSuccessful: &f, ResultCode: "X"}).validate(); err == nil {
		t.Fatal("IsSuccessful=false hata dönmeli")
	}
	if err := (BankCardInformationData{BankName: "X"}).validate(); err != nil {
		t.Fatal("IsSuccessful yokken hata olmamalı")
	}
}
