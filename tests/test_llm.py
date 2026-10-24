"""Тесты слоя объяснений: шаблонный fallback работает без сети.

Мы НЕ дергаем реальную LLM в юнит-тестах — проверяем, что при недоступной
нейросети сервис всё равно отдаёт понятный текст, а именно это критично для
демо-надёжности.
"""

from __future__ import annotations

from scoring_service.config import Settings
from scoring_service.engine import ScoringEngine
from scoring_service.llm import LLMExplainer
from scoring_service.schemas import Decision, EvaluateRequest

ENGINE = ScoringEngine(Settings())
EXPLAINER = LLMExplainer(Settings(ai_base_url="http://127.0.0.1:1/v1", ai_timeout_seconds=0.5))


def make(**overrides) -> EvaluateRequest:
    base = dict(
        company_inn="7701000002",
        monthly_revenue=2_000_000,
        monthly_net_profit=600_000,
        company_age_months=24,
        requested_amount=5_000_000,
        requested_term_months=24,
        product_rate=16.5,
        bki_active_loans_monthly_payment=0,
        bki_delinquency_days=0,
        has_bankruptcy=False,
        documents_uploaded=False,
    )
    base.update(overrides)
    return EvaluateRequest(**base)


def test_template_explains_every_decision():
    for req in (
        make(monthly_net_profit=2_000_000, requested_amount=3_000_000),  # APPROVED
        make(),                                                          # COUNTER_OFFER
        make(company_age_months=3, bki_active_loans_monthly_payment=0),  # DOCS_REQUIRED
        make(monthly_net_profit=400_000, monthly_revenue=10_000_000,
             requested_amount=1_500_000, bki_active_loans_monthly_payment=50_000,
             bki_delinquency_days=20, company_age_months=18),            # MANUAL_REVIEW
        make(has_bankruptcy=True),                                       # REJECTED
    ):
        result = ENGINE.evaluate(req)
        text = EXPLAINER._template(req, result)
        assert text and isinstance(text, str)
        assert len(text) > 40  # содержательное объяснение, не пустышка
        assert "  " not in text  # числа отформатированы без двойных пробелов


def test_explain_falls_back_to_template_when_llm_unreachable():
    req = make()
    result = ENGINE.evaluate(req)
    # Порт :1 недоступен -> должен вернуться шаблон, без исключения наружу.
    text = EXPLAINER.explain(req, result)
    assert text.strip()
    assert "не можем" in text.lower() or "банк" in text.lower() or "условия" in text.lower()


def test_prompt_contains_offer_details():
    req = make()
    result = ENGINE.evaluate(req)
    prompt = EXPLAINER._build_prompt(req, result)
    assert "Решение: COUNTER_OFFER" in prompt
    assert "₽" in prompt
    assert "DSTI" in prompt


def test_clean_strips_markdown_artifacts():
    dirty = "### **Заголовок**\n\n- пункт один\n\nОсновной текст ответа."
    cleaned = EXPLAINER._clean(dirty)
    assert "#" not in cleaned
    assert "*" not in cleaned
    assert "Основной текст" in cleaned
