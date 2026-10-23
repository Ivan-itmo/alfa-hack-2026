package models

// Rates are annual percentages; money is in whole rubles.
type Product struct {
	ID            uint    `json:"id"`
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	InterestRate  float64 `json:"interest_rate"`
	MinTermMonths int     `json:"min_term_months"`
	MaxTermMonths int     `json:"max_term_months"`
	MinAmount     int64   `json:"min_amount"`
	MaxAmount     int64   `json:"max_amount"`
}

func Products() []Product {
	return []Product{
		{1, "overdraft", "Овердрафт", 18, 1, 12, 10000, 5000000},
		{2, "revolving_credit", "Оборотный кредит", 20, 3, 36, 10000, 30000000},
		{3, "term_loan", "Инвестиционный кредит", 16, 12, 60, 10000, 100000000},
	}
}
func FindProduct(id uint) (Product, bool) {
	for _, p := range Products() {
		if p.ID == id {
			return p, true
		}
	}
	return Product{}, false
}
