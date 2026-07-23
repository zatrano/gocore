package recipients

import "github.com/zatrano/gocore/pkg/validation"

// Normalize, satırdaki e-posta ve telefon alanlarını normalize eder.
// Boş alanlar atlanır; geçersiz değerler hata döner.
func Normalize(r Row) (Row, error) {
	out := r
	if r.Email != "" {
		email, err := validation.NormalizeEmail(r.Email)
		if err != nil {
			return Row{}, err
		}
		out.Email = email
	}
	if r.Phone != "" {
		phone, err := validation.NormalizePhone(r.Phone)
		if err != nil {
			return Row{}, err
		}
		out.Phone = phone
	}
	return out, nil
}

// NormalizeAll, listedeki her satırı normalize eder; ilk hatada durur.
func NormalizeAll(list []Row) ([]Row, error) {
	out := make([]Row, 0, len(list))
	for _, r := range list {
		n, err := Normalize(r)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}
