"""Ядро движка скоринга: DSTI, скоринг-балл и логика 5 исходов.

Модуль намеренно не зависит от FastAPI и LLM — это чистая детерминированная
функция от финансовых показателей, что делает решения воспроизводимыми и
тестируемыми отдельно от сети.
"""

from __future__ import annotations

import math

from .config import Settings
from .schemas import Decision, EvaluateRequest, KeyFactor, Offer


def _annual_fraction(rate: float) -> float:
    """Приводим ставку к годовой доле.

    Контракт допускает и проценты (16.5), и долю (0.165): Go-бэкенд шлёт 16.5,
    пример из задания — 0.165. Значения > 1 трактуем как проценты.
    """
    return rate / 100.0 if rate > 1 else rate


def annuity_payment(amount: float, term_months: int, rate: float) -> float:
    """Ежемесячный аннуитетный платёж."""
    if term_months <= 0 or amount <= 0:
        return 0.0
    monthly_rate = _annual_fraction(rate) / 12.0
    if monthly_rate == 0:
        return amount / term_months
    return amount * monthly_rate / (1 - (1 + monthly_rate) ** (-term_months))


def solve_max_amount(max_payment: float, term_months: int, rate: float) -> float:
    """Максимальная сумма кредита при заданном предельном платеже."""
    if max_payment <= 0 or term_months <= 0:
        return 0.0
    monthly_rate = _annual_fraction(rate) / 12.0
    if monthly_rate == 0:
        return max_payment * term_months
    return max_payment * (1 - (1 + monthly_rate) ** (-term_months)) / monthly_rate


def _round_down(value: float, step: int) -> int:
    if step <= 0:
        return int(value)
    return int(math.floor(value / step) * step)


class ScoringEngine:
    """Прозрачный конвейер оценки кредитного риска."""

    def __init__(self, settings: Settings | None = None) -> None:
        self.s = settings or Settings.from_env()

    # ------------------------------------------------------------------ #
    # Публичный вход                                                     #
    # ------------------------------------------------------------------ #
    def evaluate(self, req: EvaluateRequest) -> dict:
        """Возвращает результат БЕЗ ai_explanation (его добавляет слой LLM)."""
        factors: list[KeyFactor] = []

        requested_payment = annuity_payment(
            req.requested_amount, req.requested_term_months, req.product_rate
        )
        total_payment = requested_payment + req.bki_active_loans_monthly_payment
        dsti = total_payment / req.monthly_net_profit if req.monthly_net_profit > 0 else 1.0

        score = self._score(req, dsti, factors)

        # Порядок решения (важен и намеренно прозрачен):
        #   1. Стоп-факторы перебивают всё.
        #   2. Молодой бизнес без истории -> дозапрос документов.
        #   3. DSTI выше верхней границы -> отказ (слишком высокая нагрузка).
        #   4. Пограничный скор -> ручной разбор офицером.
        #   5. DSTI <= 0.35 -> одобрение запрошенных условий.
        #   6. 0.35 < DSTI <= 0.55 -> встречное предложение с безопасными условиями.
        stop_reason = self._stop_factor(req)
        if stop_reason:
            return self._reject(stop_reason, score, dsti, factors)

        if self._needs_documents(req):
            factors.append(
                KeyFactor(
                    factor="AGE_OR_HISTORY",
                    impact="negative",
                    description="Молодой бизнес без кредитной истории",
                )
            )
            return self._result(
                Decision.DOCS_REQUIRED,
                score,
                dsti,
                factors,
                None,
                [
                    "Бухгалтерская отчётность за последний квартал",
                    "Выписка по расчётному счёту за 6 месяцев",
                ],
            )

        if dsti > self.s.dsti_counter_max:
            return self._reject(
                f"Долговая нагрузка {dsti:.0%} превышает допустимый предел банка",
                score,
                dsti,
                factors,
            )

        if self.s.score_manual_review_low <= score <= self.s.score_manual_review_high:
            factors.append(
                KeyFactor(
                    factor="SCORE",
                    impact="neutral",
                    description=f"Пограничный скоринг-балл {score}: нужна ручная оценка",
                )
            )
            return self._result(Decision.MANUAL_REVIEW, score, dsti, factors, None, [])

        if dsti <= self.s.dsti_approve_max:
            factors.append(
                KeyFactor(
                    factor="DSTI",
                    impact="positive",
                    description=f"Долговая нагрузка {dsti:.0%} в пределах нормы",
                )
            )
            offer = Offer(
                amount=req.requested_amount,
                term_months=req.requested_term_months,
                monthly_payment=int(math.ceil(requested_payment)),
                interest_rate=req.product_rate,
            )
            return self._result(Decision.APPROVED, score, dsti, factors, offer, [])

        offer = self._build_counter_offer(req, factors)
        if offer is None:
            return self._reject(
                "Недостаточно свободной прибыли для безопасного платежа",
                score,
                dsti,
                factors,
            )
        return self._result(Decision.COUNTER_OFFER, score, dsti, factors, offer, [])

    # ------------------------------------------------------------------ #
    # Стоп-факторы и дозапрос                                            #
    # ------------------------------------------------------------------ #
    def _stop_factor(self, req: EvaluateRequest) -> str | None:
        if req.has_bankruptcy:
            return "По данным ФНС у компании есть признаки банкротства"
        if req.bki_delinquency_days > self.s.delinquency_stop_days:
            return (
                f"Просрочка {req.bki_delinquency_days} дней превышает "
                f"порог {self.s.delinquency_stop_days} дней"
            )
        if req.monthly_net_profit <= 0:
            return "Отсутствует положительная чистая прибыль для обслуживания кредита"
        return None

    def _needs_documents(self, req: EvaluateRequest) -> bool:
        if req.documents_uploaded:
            return False
        # Молодой бизнес без кредитной истории -> просим отчётность.
        return (
            req.company_age_months < self.s.young_business_months
            and req.bki_active_loans_monthly_payment == 0
        )

    # ------------------------------------------------------------------ #
    # Встречное предложение                                              #
    # ------------------------------------------------------------------ #
    def _build_counter_offer(
        self, req: EvaluateRequest, factors: list[KeyFactor]
    ) -> Offer | None:
        """Подбираем безопасные условия под предельный платёж.

        PMT_max = (Чистая прибыль * safe_payment_ratio) - Платежи в БКИ.

        Приоритет для клиента:
          1. Сохранить запрошенную сумму, увеличив срок в пределах продукта.
          2. Если срок не спасает — снизить сумму при запрошенном сроке.
        """
        pmt_max = (
            req.monthly_net_profit * self.s.safe_payment_ratio
            - req.bki_active_loans_monthly_payment
        )
        if pmt_max <= 0:
            return None

        # Вариант 1: увеличиваем срок, сохраняя запрошенную сумму.
        for term in range(req.requested_term_months, self.s.max_term_months + 1):
            payment = annuity_payment(req.requested_amount, term, req.product_rate)
            if payment <= pmt_max:
                if term > req.requested_term_months:
                    factors.append(
                        KeyFactor(
                            factor="TERM",
                            impact="positive",
                            description=f"Срок увеличен до {term} мес. для комфортного платежа",
                        )
                    )
                factors.append(
                    KeyFactor(
                        factor="DSTI",
                        impact="negative",
                        description="Высокая долговая нагрузка: подобраны безопасные условия",
                    )
                )
                return Offer(
                    amount=req.requested_amount,
                    term_months=term,
                    monthly_payment=int(math.ceil(payment)),
                    interest_rate=req.product_rate,
                )

        # Вариант 2: снижаем сумму при запрошенном сроке.
        amount = _round_down(
            solve_max_amount(pmt_max, req.requested_term_months, req.product_rate),
            self.s.amount_round_step,
        )
        amount = min(amount, req.requested_amount)
        if amount < self.s.min_offer_amount:
            return None

        payment = annuity_payment(amount, req.requested_term_months, req.product_rate)
        reduction = 1 - amount / req.requested_amount
        factors.append(
            KeyFactor(
                factor="DSTI",
                impact="negative",
                description=f"Высокая долговая нагрузка: сумма снижена на {reduction:.0%}",
            )
        )
        return Offer(
            amount=amount,
            term_months=req.requested_term_months,
            monthly_payment=int(math.ceil(payment)),
            interest_rate=req.product_rate,
        )

    # ------------------------------------------------------------------ #
    # Скоринг-балл                                                       #
    # ------------------------------------------------------------------ #
    def _score(
        self,
        req: EvaluateRequest,
        dsti: float,
        factors: list[KeyFactor],
    ) -> int:
        """Простой прозрачный скоринг 0..1000 из понятных слагаемых."""
        score = 500

        # DSTI — главный фактор (до +/-250).
        if dsti <= 0.35:
            score += 250
            factors.append(KeyFactor(factor="DSTI", impact="positive", description="Низкая долговая нагрузка"))
        elif dsti <= 0.55:
            score += 50
        else:
            score -= 250
            factors.append(KeyFactor(factor="DSTI", impact="negative", description="Высокая долговая нагрузка"))

        # Возраст бизнеса (до +150 / -100).
        if req.company_age_months >= 24:
            score += 150
            factors.append(KeyFactor(factor="AGE", impact="positive", description="Бизнес работает более 2 лет"))
        elif req.company_age_months >= 12:
            score += 100
            factors.append(KeyFactor(factor="AGE", impact="positive", description="Бизнес работает более года"))
        elif req.company_age_months < self.s.young_business_months:
            score -= 100
            factors.append(KeyFactor(factor="AGE", impact="negative", description="Молодой бизнес"))

        # Кредитная дисциплина.
        if req.bki_delinquency_days == 0:
            score += 100
        elif req.bki_delinquency_days <= 30:
            score -= 50
            factors.append(KeyFactor(factor="BKI", impact="negative", description="Есть небольшие просрочки"))
        else:
            score -= 200
            factors.append(KeyFactor(factor="BKI", impact="negative", description="Существенные просрочки в БКИ"))

        # Рентабельность (прибыль к выручке).
        if req.monthly_revenue > 0:
            margin = req.monthly_net_profit / req.monthly_revenue
            if margin >= 0.2:
                score += 50
            elif margin < 0.05:
                score -= 50

        # Действующая долговая нагрузка.
        if req.bki_active_loans_monthly_payment > 0:
            score -= 50

        return max(self.s.score_min, min(self.s.score_max, score))

    # ------------------------------------------------------------------ #
    # Сборка ответа                                                      #
    # ------------------------------------------------------------------ #
    def _reject(
        self, reason: str, score: int, dsti: float, factors: list[KeyFactor]
    ) -> dict:
        factor_name = "STOP_FACTOR" if self._is_stop(reason) else "POLICY"
        factors.append(KeyFactor(factor=factor_name, impact="negative", description=reason))
        result = self._result(Decision.REJECTED, min(int(score), 300), dsti, factors, None, [])
        result["rejection_reason"] = reason
        return result

    @staticmethod
    def _is_stop(reason: str) -> bool:
        return "банкротства" in reason or "Просрочка" in reason

    def _result(
        self,
        decision: Decision,
        score: int,
        dsti: float,
        factors: list[KeyFactor],
        offer: Offer | None,
        required_documents: list[str],
    ) -> dict:
        return {
            "decision": decision,
            "score_value": int(score),
            "dsti": round(float(dsti), 4),
            "offer": offer,
            "required_documents": required_documents,
            "key_factors": factors,
            "approved_amount": offer.amount if offer else 0,
            "approved_term_months": offer.term_months if offer else 0,
            "monthly_payment": offer.monthly_payment if offer else 0,
            "interest_rate": offer.interest_rate if offer else 0.0,
            "rejection_reason": "",
        }
