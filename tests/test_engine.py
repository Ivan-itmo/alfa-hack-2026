"""Юнит-тесты ядра скоринга: 5 исходов, DSTI и подбор встречных условий.

Тесты не трогают сеть и LLM — движок детерминирован. Числа подобраны так,
чтобы каждый исход срабатывал по задуманной ветке политики.
"""

from __future__ import annotations

from scoring_service.config import Settings
from scoring_service.engine import ScoringEngine, annuity_payment, solve_max_amount
from scoring_service.schemas import Decision, EvaluateRequest

ENGINE = ScoringEngine(Settings())


def make(**overrides) -> EvaluateRequest:
    base = dict(
        company_inn="7701000002",
        monthly_revenue=2_000_000,
        monthly_net_profit=1_000_000,
        company_age_months=30,
        requested_amount=4_000_000,
        requested_term_months=12,
        product_rate=16.5,
        bki_active_loans_monthly_payment=0,
        bki_delinquency_days=0,
        has_bankruptcy=False,
        documents_uploaded=False,
    )
    base.update(overrides)
    return EvaluateRequest(**base)


# --------------------------------------------------------------------------- #
# Финансовая математика                                                       #
# --------------------------------------------------------------------------- #
def test_annuity_payment_has_interest_above_principal_share():
    payment = annuity_payment(1_000_000, 12, 16.5)
    assert payment > 1_000_000 / 12
    assert 90_000 < payment < 92_000


def test_rate_accepts_percent_and_fraction_equally():
    # 16.5 (проценты) и 0.165 (доля) — одно и то же.
    assert abs(annuity_payment(1_000_000, 12, 16.5) - annuity_payment(1_000_000, 12, 0.165)) < 1e-6


def test_solve_max_amount_inverts_annuity():
    amount = solve_max_amount(100_000, 12, 16.5)
    assert abs(annuity_payment(amount, 12, 16.5) - 100_000) < 1.0


# --------------------------------------------------------------------------- #
# 5 исходов                                                                   #
# --------------------------------------------------------------------------- #
def test_approved_when_dsti_low():
    result = ENGINE.evaluate(
        make(
            monthly_net_profit=2_000_000,
            monthly_revenue=4_000_000,
            requested_amount=3_000_000,
            bki_active_loans_monthly_payment=60_000,
            company_age_months=18,
        )
    )
    assert result["decision"] is Decision.APPROVED
    assert result["dsti"] <= 0.35
    assert result["offer"].amount == 3_000_000
    assert result["approved_amount"] == 3_000_000
    assert result["approved_term_months"] == 12


def test_rejected_on_bankruptcy_stop_factor():
    result = ENGINE.evaluate(make(has_bankruptcy=True))
    assert result["decision"] is Decision.REJECTED
    assert "банкротства" in result["rejection_reason"]
    assert result["approved_amount"] == 0
    assert result["offer"] is None


def test_rejected_on_long_delinquency():
    result = ENGINE.evaluate(make(bki_delinquency_days=120))
    assert result["decision"] is Decision.REJECTED
    assert "Просрочка" in result["rejection_reason"]


def test_docs_required_for_young_business_without_history():
    result = ENGINE.evaluate(
        make(company_age_months=3, bki_active_loans_monthly_payment=0, documents_uploaded=False)
    )
    assert result["decision"] is Decision.DOCS_REQUIRED
    assert len(result["required_documents"]) >= 1
    assert result["approved_amount"] == 0


def test_docs_uploaded_skips_docs_required():
    result = ENGINE.evaluate(
        make(company_age_months=3, bki_active_loans_monthly_payment=0, documents_uploaded=True)
    )
    assert result["decision"] is not Decision.DOCS_REQUIRED


def test_counter_offer_extends_term_keeping_amount():
    # DSTI ~0.36 -> встречное предложение; срок растягивается, сумма сохраняется.
    result = ENGINE.evaluate(
        make(monthly_net_profit=1_000_000, monthly_revenue=2_000_000, requested_amount=4_000_000,
             requested_term_months=12)
    )
    assert result["decision"] is Decision.COUNTER_OFFER
    offer = result["offer"]
    assert 0.35 < result["dsti"] <= 0.55
    assert offer.amount == 4_000_000
    assert offer.term_months >= 13
    assert offer.monthly_payment <= 1_000_000 * 0.35 + 1


def test_counter_offer_reduces_amount_when_term_maxed():
    # Срок уже максимальный -> снижаем сумму под безопасный платёж.
    result = ENGINE.evaluate(
        make(monthly_net_profit=1_000_000, monthly_revenue=2_000_000, requested_amount=20_000_000,
             requested_term_months=60)
    )
    assert result["decision"] is Decision.COUNTER_OFFER
    offer = result["offer"]
    assert offer.term_months == 60
    assert offer.amount < 20_000_000
    assert offer.monthly_payment <= 1_000_000 * 0.35 + 1


def test_rejected_when_dsti_too_high():
    result = ENGINE.evaluate(
        make(monthly_net_profit=200_000, monthly_revenue=1_000_000,
             requested_amount=5_000_000, requested_term_months=12,
             bki_active_loans_monthly_payment=100_000)
    )
    assert result["decision"] is Decision.REJECTED


def test_manual_review_on_boundary_score():
    # Подобранные факторы дают ровно пограничный балл (зона 450..550).
    result = ENGINE.evaluate(
        make(monthly_net_profit=400_000, monthly_revenue=10_000_000,
             requested_amount=1_500_000, requested_term_months=12,
             bki_active_loans_monthly_payment=50_000, bki_delinquency_days=20,
             company_age_months=18)
    )
    assert result["score_value"] == 500
    assert result["decision"] is Decision.MANUAL_REVIEW
    assert result["approved_amount"] == 0


# --------------------------------------------------------------------------- #
# Объяснимость и границы                                                      #
# --------------------------------------------------------------------------- #
def test_key_factors_always_explain_decision():
    for req in (make(), make(has_bankruptcy=True), make(company_age_months=3, bki_active_loans_monthly_payment=0)):
        result = ENGINE.evaluate(req)
        assert len(result["key_factors"]) >= 1
        assert all(f.factor and f.description for f in result["key_factors"])
        assert all(f.impact in {"positive", "negative", "neutral"} for f in result["key_factors"])


def test_score_never_out_of_range():
    for profit in (0, 100_000, 350_000, 1_000_000, 5_000_000):
        result = ENGINE.evaluate(make(monthly_net_profit=profit))
        assert 0 <= result["score_value"] <= 1000
        assert result["dsti"] >= 0
