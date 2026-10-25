#!/usr/bin/env python3
"""
Комплексный автоматический скрипт сквозного тестирования и валидации кредитного конвейера
Alfa Biz Credit AI.

Проверяет:
1. Инфраструктуру, порты и регуляторные артефакты (OpenAPI, Swagger, ТЗ-файлы)
2. E2E бизнес-сценарии:
   - client1: DRAFT -> SCORING -> APPROVED -> ISSUED (ПЭП подписание, баланс)
   - client2: DRAFT -> COUNTER_OFFER -> ACCEPT -> APPROVED
   - client3: DRAFT -> DOCS_REQUIRED -> UPLOAD -> MANUAL_REVIEW -> DOWNLOAD
   - officer1: Dossier review & decision APPROVE
   - scoring_service: Прямой стоп-фактор банкротства (REJECTED)
   - AI-советник: Сравнение финансовых продуктов через LLM Gemma
"""

import io
import os
import sys
import time
import requests
import yaml

# Base URLs
FRONTEND_URL = os.getenv("FRONTEND_URL", "http://localhost:3000")
BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")
SCORING_URL = os.getenv("SCORING_URL", "http://localhost:8000")

# ANSI Colors
GREEN = "\033[92m"
RED = "\033[91m"
YELLOW = "\033[93m"
CYAN = "\033[96m"
BOLD = "\033[1m"
RESET = "\033[0m"

results_checklist = []


def record_result(category: str, test_name: str, passed: bool, details: str = ""):
    status_str = f"{GREEN}PASS{RESET}" if passed else f"{RED}FAIL{RESET}"
    results_checklist.append({
        "category": category,
        "name": test_name,
        "passed": passed,
        "details": details
    })
    print(f"  [{status_str}] {test_name}" + (f" ({details})" if details else ""))


def section_header(title: str):
    print(f"\n{BOLD}{CYAN}=== {title} ==={RESET}")


def check_artifacts_and_infra():
    section_header("1. ИНФРАСТРУКТУРНЫЕ ПРОВЕРКИ И РЕГУЛЯТОРНЫЕ АРТЕФАКТЫ")

    # 1.1 Доступность портов и эндпоинтов
    endpoints = [
        ("Frontend UI", f"{FRONTEND_URL}/", 200),
        ("Backend Healthz", f"{BACKEND_URL}/healthz", 200),
        ("Scoring Service Healthz", f"{SCORING_URL}/healthz", 200),
        ("Swagger UI Docs", f"{BACKEND_URL}/docs", 200),
        ("OpenAPI YAML Spec", f"{BACKEND_URL}/openapi.yaml", 200),
    ]

    for name, url, expected_code in endpoints:
        try:
            r = requests.get(url, timeout=5)
            ok = (r.status_code == expected_code)
            record_result("Инфраструктура", f"GET {name}", ok, f"HTTP {r.status_code}")
        except Exception as e:
            record_result("Инфраструктура", f"GET {name}", False, str(e))

    # 1.2 Валидация схемы OpenAPI
    try:
        r_yaml = requests.get(f"{BACKEND_URL}/openapi.yaml", timeout=5)
        spec = yaml.safe_load(r_yaml.text)
        paths_count = len(spec.get("paths", {}))
        valid_spec = paths_count >= 15 and spec.get("openapi", "").startswith("3.0")
        record_result("Спецификация", "OpenAPI 3.0 YAML валидация", valid_spec, f"Версия: {spec.get('openapi')}, путей: {paths_count}")
    except Exception as e:
        record_result("Спецификация", "OpenAPI 3.0 YAML валидация", False, str(e))

    # 1.3 Обязательные регуляторные файлы на диске
    required_files = [
        "THIRD_PARTY_NOTICES",
        "postman_collection.json",
        "docs/state_machine.svg",
        "docs/state_machine.md",
        "scoring_service/README.md",
    ]

    for rel_path in required_files:
        exists = os.path.isfile(rel_path)
        size = os.path.getsize(rel_path) if exists else 0
        record_result("Артефакты ТЗ", f"Файл {rel_path}", exists and size > 0, f"{size} байт" if exists else "Не найден")


def run_e2e_scenarios():
    section_header("2. СКВОЗНЫЕ БИЗНЕС-СЦЕНАРИИ КОНВЕЙЕРА (E2E)")
    session = requests.Session()

    # -------------------------------------------------------------
    # 2.1 Тест client1: DRAFT -> SCORING -> APPROVED -> ISSUED
    # -------------------------------------------------------------
    print(f"\n{BOLD}[2.1] Тестирование client1 (ООО 'Стабильный Бизнес'){RESET}")
    # Авторизация
    r = session.post(f"{BACKEND_URL}/api/auth/login", json={"email": "client1", "password": "123"})
    assert r.status_code == 200, f"Login client1 failed: {r.text}"
    token_c1 = r.json()["token"]
    headers_c1 = {"Authorization": f"Bearer {token_c1}"}
    record_result("E2E client1", "Авторизация client1", True, "Токен получен")

    # Дашборд
    r_dash = session.get(f"{BACKEND_URL}/api/dashboard", headers=headers_c1)
    dash_data = r_dash.json()
    okved = dash_data.get("company_info", {}).get("okved")
    balance_initial = dash_data.get("company_info", {}).get("balance", 0)
    has_news = len(dash_data.get("news_feed", [])) > 0
    record_result("E2E client1", "Чтение дашборда (ОКВЭД 62.01, новости)", okved == "62.01" and has_news, f"ОКВЭД={okved}, новостей={len(dash_data.get('news_feed', []))}")

    # Создание заявки на 1 500 000 руб на 12 мес
    r_app = session.post(f"{BACKEND_URL}/api/applications", json={
        "product_id": 2,
        "purpose": "Пополнение оборотных средств на закупку серверов",
        "amount": 1500000,
        "term_months": 12
    }, headers=headers_c1)
    assert r_app.status_code == 201, f"Create app failed: {r_app.text}"
    app1 = r_app.json()
    app1_id = app1["id"]
    record_result("E2E client1", f"Создание заявки #{app1_id}", app1["status"] == "DRAFT", f"Сумма: {app1['amount']} ₽")

    # Отправка на скоринг
    t0 = time.time()
    r_sub = session.post(f"{BACKEND_URL}/api/applications/{app1_id}/submit", headers=headers_c1)
    llm_duration = time.time() - t0
    assert r_sub.status_code == 200, f"Submit failed: {r_sub.text}"
    scored_c1 = r_sub.json()

    c1_ok = (
        scored_c1["status"] == "APPROVED" and
        scored_c1["scoring_source"] == "ml" and
        scored_c1["dsti"] <= 0.35 and
        len(scored_c1.get("ai_explanation", "")) > 30 and
        len(scored_c1.get("status_logs", [])) >= 3
    )
    record_result(
        "E2E client1",
        "Автоскоринг с живой LLM (DSTI <= 0.35, source=ml)",
        c1_ok,
        f"Статус={scored_c1['status']}, DSTI={scored_c1['dsti']:.2f}, Время={llm_duration:.2f}с, Logs={len(scored_c1.get('status_logs', []))}"
    )

    # Подписание договора ПЭП (код 1234)
    r_sign = session.post(f"{BACKEND_URL}/api/applications/{app1_id}/sign", json={"code": "1234"}, headers=headers_c1)
    assert r_sign.status_code == 200, f"Sign failed: {r_sign.text}"
    signed_c1 = r_sign.json()

    # Проверка баланса в дашборде
    r_dash2 = session.get(f"{BACKEND_URL}/api/dashboard", headers=headers_c1)
    balance_after = r_dash2.json().get("company_info", {}).get("balance", 0)
    delta = balance_after - balance_initial

    # Проверка status_logs на запись о подписании
    sign_logged = any("1234" in log.get("comment", "") or log.get("to_status") == "ISSUED" for log in signed_c1.get("status_logs", []))
    sign_ok = (signed_c1["status"] == "ISSUED" and delta == 1500000 and sign_logged)
    record_result(
        "E2E client1",
        "Подписание ПЭП кодом 1234 и зачисление средств",
        sign_ok,
        f"Статус=ISSUED, Баланс: +{delta:,} ₽, Записей в логе: {len(signed_c1.get('status_logs', []))}"
    )

    # -------------------------------------------------------------
    # 2.2 Тест client2: DRAFT -> COUNTER_OFFER -> ACCEPT -> APPROVED
    # -------------------------------------------------------------
    print(f"\n{BOLD}[2.2] Тестирование client2 (ИП 'Ритейл' / Встречное предложение){RESET}")
    r = session.post(f"{BACKEND_URL}/api/auth/login", json={"email": "client2", "password": "123"})
    token_c2 = r.json()["token"]
    headers_c2 = {"Authorization": f"Bearer {token_c2}"}

    # Подача заявки на заведомо завышенную сумму 10 000 000 руб
    r_app2 = session.post(f"{BACKEND_URL}/api/applications", json={
        "product_id": 2,
        "purpose": "Масштабирование розничной сети",
        "amount": 10000000,
        "term_months": 12
    }, headers=headers_c2)
    app2_id = r_app2.json()["id"]

    r_sub2 = session.post(f"{BACKEND_URL}/api/applications/{app2_id}/submit", headers=headers_c2)
    scored_c2 = r_sub2.json()

    counter_amount = scored_c2.get("counter_amount") or scored_c2.get("approved_amount")
    c2_counter_ok = (
        scored_c2["status"] == "COUNTER_OFFER" and
        counter_amount is not None and counter_amount < 10000000 and
        len(scored_c2.get("ai_explanation", "")) > 20
    )
    record_result(
        "E2E client2",
        "Генерация COUNTER_OFFER под безопасный DSTI",
        c2_counter_ok,
        f"Статус={scored_c2['status']}, Предложено: {counter_amount:,} ₽ вместо 10,000,000 ₽"
    )

    # Принятие встречного предложения клиентом
    r_accept = session.post(f"{BACKEND_URL}/api/applications/{app2_id}/counter-offer", json={"action": "accept"}, headers=headers_c2)
    accepted_c2 = r_accept.json()
    c2_accept_ok = (accepted_c2["status"] == "APPROVED" and accepted_c2["approved_amount"] == counter_amount)
    record_result("E2E client2", "Клиент принимает встречное предложение (POST /counter-offer)", c2_accept_ok, f"Итоговый статус={accepted_c2['status']}, Сумма={accepted_c2['approved_amount']:,} ₽")

    # -------------------------------------------------------------
    # 2.3 Тест client3: DRAFT -> DOCS_REQUIRED -> UPLOAD -> MANUAL_REVIEW
    # -------------------------------------------------------------
    print(f"\n{BOLD}[2.3] Тестирование client3 (Стартап / Дозапрос документов){RESET}")
    r = session.post(f"{BACKEND_URL}/api/auth/login", json={"email": "client3", "password": "123"})
    token_c3 = r.json()["token"]
    headers_c3 = {"Authorization": f"Bearer {token_c3}"}

    r_app3 = session.post(f"{BACKEND_URL}/api/applications", json={
        "product_id": 2,
        "purpose": "Закупка первого оборудования",
        "amount": 500000,
        "term_months": 12
    }, headers=headers_c3)
    app3_id = r_app3.json()["id"]

    r_sub3 = session.post(f"{BACKEND_URL}/api/applications/{app3_id}/submit", headers=headers_c3)
    scored_c3 = r_sub3.json()
    record_result("E2E client3", "Автодозапрос документов DOCS_REQUIRED для молодого бизнеса", scored_c3["status"] == "DOCS_REQUIRED", f"Статус={scored_c3['status']}")

    # Загрузка фиктивного PDF отчета
    fake_pdf = b"%PDF-1.4 1 0 obj << /Title (Financial Report Q3 2026) >> endobj xref 0 1 trailer << >> startxref 100 %%EOF"
    r_up = session.post(
        f"{BACKEND_URL}/api/applications/{app3_id}/documents",
        files={"file": ("balance_sheet_2026.pdf", fake_pdf, "application/pdf")},
        headers=headers_c3
    )
    up_data = r_up.json()
    has_docs = len(up_data.get("documents", [])) > 0
    doc_id = up_data["documents"][0]["id"] if has_docs else None
    docs_logged = any(log.get("to_status") == "DOCS_UPLOADED" for log in up_data.get("status_logs", []))

    upload_ok = (r_up.status_code == 200 and up_data["status"] == "MANUAL_REVIEW" and has_docs and docs_logged)
    record_result(
        "E2E client3",
        "Загрузка документов и маршрутизация в MANUAL_REVIEW",
        upload_ok,
        f"Статус={up_data.get('status')}, Документов={len(up_data.get('documents', []))}"
    )

    # Скачивание документа
    if doc_id:
        r_dl = session.get(f"{BACKEND_URL}/api/applications/{app3_id}/documents/{doc_id}?token={token_c3}")
        dl_ok = (r_dl.status_code == 200 and r_dl.headers.get("Content-Type") == "application/pdf" and len(r_dl.content) == len(fake_pdf))
        record_result("E2E client3", "Скачивание прикрепленного документа (GET /documents/:doc_id)", dl_ok, f"HTTP {r_dl.status_code}, {len(r_dl.content)} байт, Content-Type: application/pdf")

    # -------------------------------------------------------------
    # 2.4 Тест officer1: Досье и Решение андеррайтера
    # -------------------------------------------------------------
    print(f"\n{BOLD}[2.4] Тестирование officer1 (Рабочее место кредитного офицера){RESET}")
    r = session.post(f"{BACKEND_URL}/api/auth/login", json={"email": "officer1", "password": "123"})
    token_off = r.json()["token"]
    headers_off = {"Authorization": f"Bearer {token_off}"}

    # Очередь заявок
    r_queue = session.get(f"{BACKEND_URL}/api/officer/applications", headers=headers_off)
    apps = r_queue.json().get("applications", [])
    target_app = next((a for a in apps if a["id"] == app3_id), None)
    dossier_ok = (
        target_app is not None and
        target_app.get("company") is not None and
        len(target_app.get("documents", [])) > 0 and
        len(target_app.get("status_logs", [])) > 0
    )
    record_result("E2E officer1", f"Досье заявки #{app3_id} в кабинете офицера", dossier_ok, f"Компания: {target_app.get('company', {}).get('name') if target_app else 'None'}, документов: {len(target_app.get('documents', [])) if target_app else 0}")

    # Ручное вынесение решения APPROVE
    r_dec = session.post(f"{BACKEND_URL}/api/officer/applications/{app3_id}/decision", json={
        "decision": "APPROVE",
        "comment": "Одобрено андеррайтером под поручительство генерального директора",
        "approved_amount": 500000,
        "approved_term_months": 12
    }, headers=headers_off)
    dec_data = r_dec.json()
    dec_ok = (r_dec.status_code == 200 and dec_data["status"] == "APPROVED" and dec_data["approved_amount"] == 500000)
    record_result("E2E officer1", "Ручное согласование заявки (POST /officer/:id/decision)", dec_ok, f"Новый статус={dec_data.get('status')}, Одобрено={dec_data.get('approved_amount'):,} ₽")

    # -------------------------------------------------------------
    # 2.5 Тест стоп-факторов скоринга (has_bankruptcy -> REJECTED)
    # -------------------------------------------------------------
    print(f"\n{BOLD}[2.5] Тестирование стоп-факторов скоринг-сервиса (Python){RESET}")
    r_stop = session.post(f"{SCORING_URL}/api/v1/scoring/evaluate", json={
        "company_inn": "7701999999",
        "monthly_revenue": 10000000,
        "monthly_net_profit": 3000000,
        "company_age_months": 48,
        "requested_amount": 1000000,
        "requested_term_months": 12,
        "product_rate": 16.5,
        "bki_active_loans_monthly_payment": 0,
        "bki_delinquency_days": 0,
        "has_bankruptcy": True,
        "documents_uploaded": False
    })
    stop_data = r_stop.json()
    has_stop_factor = any("банкротств" in f.get("description", "").lower() or f.get("factor") == "STOP_FACTOR" for f in stop_data.get("key_factors", []))
    stop_ok = (r_stop.status_code == 200 and stop_data["decision"] == "REJECTED" and has_stop_factor)
    record_result(
        "Стоп-факторы",
        "Отказ по банкротству (has_bankruptcy=True -> REJECTED)",
        stop_ok,
        f"Decision={stop_data.get('decision')}, Причина: {stop_data.get('rejection_reason')}"
    )

    # -------------------------------------------------------------
    # 2.6 Тест AI-советника (POST /api/v1/advice/compare)
    # -------------------------------------------------------------
    print(f"\n{BOLD}[2.6] Тестирование AI-консультанта на LLM Gemma{RESET}")
    t_adv0 = time.time()
    r_adv = session.post(f"{BACKEND_URL}/api/v1/advice/compare", json={
        "product_a": "Овердрафт для бизнеса",
        "product_b": "Оборотный кредит"
    })
    adv_duration = time.time() - t_adv0
    adv_data = r_adv.json()
    adv_text = adv_data.get("advice", "")
    adv_ok = (r_adv.status_code == 200 and len(adv_text) > 100 and adv_data.get("type") == "comparison")
    record_result(
        "AI-Советник",
        "Сравнительный анализ кредитных продуктов через LLM",
        adv_ok,
        f"HTTP {r_adv.status_code}, Длина текста: {len(adv_text)} символов, Время: {adv_duration:.2f}с"
    )

    return llm_duration, adv_duration


def print_summary_checklist(llm_scoring_time: float, llm_advisor_time: float):
    print("\n" + "=" * 80)
    print(f"{BOLD}ИТОГОВЫЙ ЧЕК-ЛИСТ ГОТОВНОСТИ ПРОЕКТА К ЗАЩИТЕ:{RESET}")
    print("=" * 80)

    total_tests = len(results_checklist)
    passed_tests = sum(1 for t in results_checklist if t["passed"])
    failed_tests = total_tests - passed_tests

    # Группировка по категориям
    categories = {}
    for t in results_checklist:
        categories.setdefault(t["category"], []).append(t)

    for cat, items in categories.items():
        print(f"\n{BOLD}{cat}:{RESET}")
        for item in items:
            mark = f"{GREEN}✓ PASS{RESET}" if item["passed"] else f"{RED}✗ FAIL{RESET}"
            print(f"  [{mark}] {item['name']:<55} | {item['details']}")

    print("\n" + "-" * 80)
    print(f"{BOLD}СВОДКА МЕТРИК ПРОИЗВОДИТЕЛЬНОСТИ LLM:{RESET}")
    print(f"  • Время генерации XAI-объяснения скоринга: {BOLD}{llm_scoring_time:.2f} сек{RESET}")
    print(f"  • Время генерации сравнительного анализа AI-советника: {BOLD}{llm_advisor_time:.2f} сек{RESET}")
    print("-" * 80)

    success_rate = (passed_tests / total_tests) * 100
    print(f"\n{BOLD}РЕЗУЛЬТАТ: {passed_tests}/{total_tests} тестов пройдено ({success_rate:.1f}%){RESET}")

    if failed_tests == 0:
        print(f"\n{GREEN}{BOLD} СИСТЕМА ПОЛНОСТЬЮ РАБОТОСПОСОБНА И ГОТОВА К ЗАЩИТЕ!{RESET}\n")
        return 0
    else:
        print(f"\n{RED}{BOLD} ОБНАРУЖЕНЫ ОШИБКИ ({failed_tests} тестов провалено). ТРЕБУЕТСЯ ВНИМАНИЕ!{RESET}\n")
        return 1


def main():
    check_artifacts_and_infra()
    llm_sc_time, llm_adv_time = run_e2e_scenarios()
    exit_code = print_summary_checklist(llm_sc_time, llm_adv_time)
    sys.exit(exit_code)


if __name__ == "__main__":
    main()
