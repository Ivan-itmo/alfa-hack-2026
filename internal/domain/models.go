package domain

import "time"

// CreditProduct represents different banking credit products
type CreditProduct string

const (
	ProductOverdraft       CreditProduct = "overdraft"
	ProductRevolvingCredit CreditProduct = "revolving_credit"
	ProductTermLoan        CreditProduct = "term_loan"
	ProductLeasing         CreditProduct = "leasing"
)

// CreditAdviceRequest contains parameters for financial advice
type CreditAdviceRequest struct {
	ClientID        string    `json:"client_id" validate:"required"`
	Revenue         float64   `json:"revenue" validate:"required,min=0"`
	NetProfit       float64   `json:"net_profit" validate:"required,min=0"`
	RequestedAmount float64   `json:"requested_amount" validate:"required,min=0"`
	RequestedTerm   int       `json:"requested_term" validate:"required,min=1"`
	PaymentType     string    `json:"payment_type"` // monthly, quarterly, etc.
	CreditProduct   string    `json:"credit_product"`
	Context         string    `json:"context,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// CreditAdviceResponse contains AI-generated financial advice
type CreditAdviceResponse struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Advice    string    `json:"advice"`
	RiskLevel string    `json:"risk_level"` // low, medium, high
	Reasoning string    `json:"reasoning"`
	CreatedAt time.Time `json:"created_at"`
}

// CreditComparison represents comparison between two credit products
type CreditComparison struct {
	ProductA       string `json:"product_a"`
	ProductB       string `json:"product_b"`
	Differences    string `json:"differences"`
	Recommendation string `json:"recommendation"`
	SuitableForA   string `json:"suitable_for_a"`
	SuitableForB   string `json:"suitable_for_b"`
}

// BankDecisionExplanation explains bank's counter-offer or rejection
type BankDecisionExplanation struct {
	DecisionType   string   `json:"decision_type"` // approval, rejection, counter_offer
	OriginalAmount float64  `json:"original_amount"`
	OriginalTerm   int      `json:"original_term"`
	CounterAmount  float64  `json:"counter_amount,omitempty"`
	CounterTerm    int      `json:"counter_term,omitempty"`
	Explanation    string   `json:"explanation"`
	Reasons        []string `json:"reasons"`
	Recommendation string   `json:"recommendation"`
}
