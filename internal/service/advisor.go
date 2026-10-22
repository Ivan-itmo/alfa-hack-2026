package service

import (
	"context"

	"biz-credit-ai/internal/ai"
	"biz-credit-ai/internal/config"
	"biz-credit-ai/internal/domain"
)

// AdvisorService handles AI-based financial advice
type AdvisorService struct {
	client *ai.OpenAICompatibleClient
	cfg    *config.Config
}

// NewAdvisorService creates a new advisor service
func NewAdvisorService(cfg *config.Config) *AdvisorService {
	return &AdvisorService{
		client: ai.NewOpenAICompatibleClient(&cfg.AI),
		cfg:    cfg,
	}
}

// CompareProducts explains differences between credit products
func (s *AdvisorService) CompareProducts(ctx context.Context, productA, productB string) (string, error) {
	messages := ai.PromptCompareProducts(productA, productB)
	return s.client.Chat(ctx, messages, 0.7, 1000)
}

// GetContextualAdvice gives advice based on client parameters
func (s *AdvisorService) GetContextualAdvice(ctx context.Context, request domain.CreditAdviceRequest) (string, error) {
	messages := ai.PromptContextualHint(request)
	return s.client.Chat(ctx, messages, 0.7, 1000)
}

// ExplainBankDecision explains bank's counter-offer or rejection
func (s *AdvisorService) ExplainBankDecision(ctx context.Context, decision domain.BankDecisionExplanation) (string, error) {
	messages := ai.PromptExplainBankDecision(decision)
	return s.client.Chat(ctx, messages, 0.7, 1000)
}
