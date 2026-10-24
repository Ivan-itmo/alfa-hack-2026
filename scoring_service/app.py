"""FastAPI-приложение микросервиса скоринга.

Эндпоинты:
  POST /api/v1/scoring/evaluate — основной контракт для Go-бэкенда.
  GET  /healthz                 — проверка живости.
"""

from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

from . import __version__
from .config import Settings
from .engine import ScoringEngine
from .llm import LLMExplainer
from .schemas import EvaluateRequest, EvaluateResponse

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
logger = logging.getLogger("scoring.app")

settings = Settings.from_env()
engine = ScoringEngine(settings)
explainer = LLMExplainer(settings)


@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("Scoring engine ready (model=%s, dsti_approve<=%.2f)", settings.ai_model, settings.dsti_approve_max)
    yield
    explainer.close()


app = FastAPI(
    title="BizCredit Scoring Engine",
    description="Прозрачный конвейер оценки кредитного риска с объяснением от LLM",
    version=__version__,
    lifespan=lifespan,
)


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok", "version": __version__}


@app.post("/api/v1/scoring/evaluate", response_model=EvaluateResponse)
def evaluate(req: EvaluateRequest) -> EvaluateResponse:
    """Оценка заявки: считаем риск, принимаем небинарное решение, объясняем."""
    result = engine.evaluate(req)
    result["ai_explanation"] = explainer.explain(req, result)
    logger.info(
        "company=%s decision=%s dsti=%.3f score=%d",
        req.company_inn,
        result["decision"].value,
        result["dsti"],
        result["score_value"],
    )
    return EvaluateResponse(**result)
