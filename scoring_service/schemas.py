"""Pydantic-схемы входного и выходного контракта API скоринга.

Входной контракт (POST /api/v1/scoring/evaluate) соответствует заданию.
Выходной контракт — надмножество: помимо вложенного объекта `offer`
отдаём плоские поля (approved_amount, counter_amount и т.д.), которые уже ожидает
существующий Go-бэкенд, чтобы интеграция работала без переписывания.
"""

from __future__ import annotations

from enum import Enum

from pydantic import BaseModel, Field


class Decision(str, Enum):
    """Пять возможных исходов — небинарное решение."""

    APPROVED = "APPROVED"
    COUNTER_OFFER = "COUNTER_OFFER"
    DOCS_REQUIRED = "DOCS_REQUIRED"
    MANUAL_REVIEW = "MANUAL_REVIEW"
    REJECTED = "REJECTED"


class EvaluateRequest(BaseModel):
    """Входные финансовые показатели заявки."""

    company_inn: str = Field(..., description="ИНН компании")
    monthly_revenue: int = Field(..., ge=0, description="Среднемесячная выручка, руб")
    monthly_net_profit: int = Field(..., ge=0, description="Среднемесячная чистая прибыль, руб")
    company_age_months: int = Field(..., ge=0, description="Возраст бизнеса, месяцев")
    requested_amount: int = Field(..., gt=0, description="Запрошенная сумма, руб")
    requested_term_months: int = Field(..., gt=0, description="Запрошенный срок, месяцев")
    product_rate: float = Field(..., ge=0, description="Ставка продукта, годовые % (напр. 16.5)")
    bki_active_loans_monthly_payment: int = Field(0, ge=0, description="Текущие платежи по БКИ, руб/мес")
    bki_delinquency_days: int = Field(0, ge=0, description="Максимальная просрочка в БКИ, дней")
    has_bankruptcy: bool = Field(False, description="Признаки банкротства по данным ФНС")
    documents_uploaded: bool = Field(False, description="Клиент уже приложил документы")


class KeyFactor(BaseModel):
    """Один объяснимый фактор, повлиявший на решение."""

    factor: str
    impact: str = Field(..., description="positive | negative | neutral")
    description: str


class Offer(BaseModel):
    """Условия одобрения или встречного предложения."""

    amount: int
    term_months: int
    monthly_payment: int
    interest_rate: float


class EvaluateResponse(BaseModel):
    """Небинарный исход + объяснимые факторы + текстовое объяснение LLM."""

    decision: Decision
    score_value: int
    dsti: float
    offer: Offer | None = None
    required_documents: list[str] = Field(default_factory=list)
    key_factors: list[KeyFactor] = Field(default_factory=list)
    ai_explanation: str = ""

    # Плоские поля для обратной совместимости с Go-клиентом scoring/client.go.
    approved_amount: int = 0
    approved_term_months: int = 0
    counter_amount: int = 0
    counter_term_months: int = 0
    monthly_payment: int = 0
    interest_rate: float = 0.0
    rejection_reason: str = ""
