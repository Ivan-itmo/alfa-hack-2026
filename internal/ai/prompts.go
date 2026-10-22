package ai

import (
	"fmt"

	"biz-credit-ai/internal/domain"
)

// PromptTemplates contains all prompt templates for financial advice scenarios

// SystemPrompt is the base system prompt for the financial advisor
const SystemPrompt = `Ты — финансовый консультант банка Альфа-Банк. Твоя задача — помогать клиентам бизнеса понимать кредитные продукты и давать понятные советы.

Правила общения:
1. Используй простой язык, без банковского сленга
2. Объясняй сложные термины простыми словами
3. Будь конкретным и давай практические рекомендации
4. Если данные клиента показывают риски — говори об этом прямо, но вежливо
5. Отвечай на русском языке`

// PromptCompareProducts explains differences between credit products
func PromptCompareProducts(productA, productB string) []ChatMessage {
	return []ChatMessage{
		{Role: "system", Content: SystemPrompt},
		{
			Role: "user",
			Content: fmt.Sprintf(`Объясни простыми словами, чем %s отличается от %s.

Для каждого продукта опиши:
1. Что это такое (простая аналогия)
2. Когда лучше использовать
3. Для кого подходит больше всего
4. Ключевые отличия в таблице`, productA, productB),
		},
	}
}

// PromptContextualHint gives advice based on client parameters
func PromptContextualHint(request domain.CreditAdviceRequest) []ChatMessage {
	content := fmt.Sprintf(`Клиент запрашивает кредит. Вот его параметры:

- Выручка: %.0f руб/мес
- Чистая прибыль: %.0f руб/мес
- Запрашиваемая сумма: %.0f руб
- Срок: %d месяцев
- Тип платежа: %s
- Продукт: %s`,
		request.Revenue,
		request.NetProfit,
		request.RequestedAmount,
		request.RequestedTerm,
		request.PaymentType,
		request.CreditProduct,
	)

	if request.Context != "" {
		content += "\n\nДополнительный контекст: " + request.Context
	}

	content += `

Проанализируй данные и дай совет:
1. Проверь, насколько платёж соответствует доходам клиента
2. Укажи на риски, если есть
3. Дай конкретную рекомендацию по сумме, сроку или продукту
4. Предложи альтернативы, если текущий запрос нерационален

Пример вывода: «Ежемесячный платёж превышает 40% вашей чистой прибыли, рекомендуем увеличить срок кредита»`

	return []ChatMessage{
		{Role: "system", Content: SystemPrompt},
		{Role: "user", Content: content},
	}
}

// PromptExplainBankDecision explains bank's counter-offer or rejection
func PromptExplainBankDecision(decision domain.BankDecisionExplanation) []ChatMessage {
	content := fmt.Sprintf(`Банк принял следующее решение: %s

Исходные параметры заявки:
- Сумма: %.0f руб
- Срок: %d месяцев`,
		decision.DecisionType,
		decision.OriginalAmount,
		decision.OriginalTerm,
	)

	if decision.CounterAmount > 0 {
		content += fmt.Sprintf("\nВстречное предложение:\n- Сумма: %.0f руб\n- Срок: %d месяцев",
			decision.CounterAmount, decision.CounterTerm)
	}

	if len(decision.Reasons) > 0 {
		content += "\n\nПричины решения:"
		for _, reason := range decision.Reasons {
			content += fmt.Sprintf("\n- %s", reason)
		}
	}

	content += `

Объясни клиенту простыми словами:
1. Почему банк принял такое решение
2. Что это значит для него
3. Какие у него есть варианты дальше
4. Стоит ли принимать встречное предложение`

	return []ChatMessage{
		{Role: "system", Content: SystemPrompt},
		{Role: "user", Content: content},
	}
}
