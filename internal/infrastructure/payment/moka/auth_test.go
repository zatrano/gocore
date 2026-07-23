package moka

import "testing"

func TestVerifyHashValue_documentationExample(t *testing.T) {
	code := "9FDFBDFC-42C5-417E-AA93-E4D9D5312AAC"
	successHash := "cdb7869505bdaaac2f4c891fc9ed889885fd7a0c880127ab5d508883efa3ee83"
	failHash := "acc929d261fdbf9c41de3db1ae854b1ee1e46344fad0292fd4bbbc43d094c2a3"

	ok, valid := VerifyHashValue(successHash, code)
	if !valid || !ok {
		t.Fatalf("success hash doğrulanamadı")
	}
	ok, valid = VerifyHashValue(failHash, code)
	if !valid || ok {
		t.Fatalf("fail hash yanlış yorumlandı")
	}
}

func TestCheckKey_deterministic(t *testing.T) {
	a := CheckKey("dealer", "user", "pass")
	b := CheckKey("dealer", "user", "pass")
	if a != b || len(a) != 64 {
		t.Fatalf("check key beklenen formatta değil: %q", a)
	}
}
