#!/usr/bin/env python3
"""Скрипт сквозной интеграционной диагностики кредитного конвейера с проверкой status_logs и OpenAPI."""

import json
import sys
import urllib.error
import urllib.request

API_BASE = "http://localhost:8080"
SCORING_BASE = "http://localhost:8000"


def req(method, url, data=None, token=None, is_raw=False):
    headers = {}
    if not is_raw:
        headers["Content-Type"] = "application/json"
    if token:
        headers["Authorization"] = f"Bearer {token}"
    body = None
    if data is not None:
        if isinstance(data, (dict, list)):
            body = json.dumps(data).encode("utf-8")
        else:
            body = data
    request = urllib.request.Request(url, data=body, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            content = response.read().decode("utf-8")
            if is_raw:
                return response.status, content
            return response.status, json.loads(content)
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8")
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"raw": raw}
    except Exception as e:
        return 0, {"error": str(e)}


def print_timeline(title, logs):
    print(f"    [Timeline] {title} ({len(logs)} записей):")
    for log in logs:
        from_st = log.get("from_status") or "START"
        to_st = log.get("to_status")
        comment = log.get("comment", "")
        if len(comment) > 60:
            comment = comment[:57] + "..."
        created = log.get("created_at", "")[:19]
        print(f"      • {created} | {from_st} ➜ {to_st} | {comment}")


def main():
    results = {}
    print("=" * 60)
    print("ЗАПУСК СКВОЗНОЙ ДИАГНОСТИКИ СИСТЕМЫ (AUDIT LOGS & OPENAPI)")
    print("=" * 60)

    # 1. Healthcheck scoring service & Backend OpenAPI
    print("\n[Шаг 1] Проверка Python scoring_service GET /healthz и OpenAPI...")
    status, body = req("GET", f"{SCORING_BASE}/healthz")
    print(f"  Scoring /healthz: HTTP {status}, Ответ: {body}")
    results["scoring_healthz"] = status == 200 and body.get("status") == "ok"

    status_yaml, yaml_content = req("GET", f"{API_BASE}/openapi.yaml", is_raw=True)
    yaml_ok = status_yaml == 200 and "openapi: 3.0" in yaml_content
    print(f"  Backend /openapi.yaml: HTTP {status_yaml}, Bytes={len(yaml_content)}, Valid={yaml_ok}")
    results["openapi_yaml"] = yaml_ok

    status_docs, docs_content = req("GET", f"{API_BASE}/docs", is_raw=True)
    docs_ok = status_docs == 200 and "SwaggerUIBundle" in docs_content
    print(f"  Backend /docs (Swagger UI): HTTP {status_docs}, Valid={docs_ok}")
    results["swagger_docs"] = docs_ok

    # 2. Тест client1
    print("\n[Шаг 2] Тестирование client1 (ООО 'Стабильный Бизнес')...")
    status, login_c1 = req("POST", f"{API_BASE}/api/auth/login", {"email": "client1", "password": "123"})
    token_c1 = login_c1.get("token")
    print(f"  Логин client1: HTTP {status}, token={bool(token_c1)}")

    # Баланс до подписания
    status, dash_before = req("GET", f"{API_BASE}/api/dashboard", token=token_c1)
    balance_before = dash_before.get("company_info", {}).get("balance", 0)
    print(f"  Текущий баланс компании: {balance_before:,} ₽")

    # Создание заявки на 1 500 000 на 12 мес.
    status, app_c1 = req(
        "POST",
        f"{API_BASE}/api/applications",
        {
            "product_id": 2,
            "purpose": "Пополнение оборотных средств",
            "amount": 1500000,
            "term_months": 12,
        },
        token=token_c1,
    )
    app1_id = app_c1.get("id")
    print(f"  Создана заявка ID={app1_id}, status={app_c1.get('status')}")
    print_timeline(f"Заявка ID={app1_id} после создания", app_c1.get("status_logs", []))

    # Отправка на скоринг
    status, scored_c1 = req("POST", f"{API_BASE}/api/applications/{app1_id}/submit", token=token_c1)
    print(f"  Скоринг заявки ID={app1_id}:")
    print(f"    - Статус: {scored_c1.get('status')}")
    print(f"    - scoring_source: {scored_c1.get('scoring_source')}")
    print(f"    - DSTI: {scored_c1.get('dsti')}")
    print(f"    - score_value: {scored_c1.get('score_value')}")
    print(f"    - ai_explanation: {scored_c1.get('ai_explanation', '')[:80]}...")
    logs_c1 = scored_c1.get("status_logs", [])
    print_timeline(f"Заявка ID={app1_id} после скоринга", logs_c1)
    results["client1_logs_count"] = len(logs_c1)

    # 3. Тест client2 (ИП Смирнов / Ритейл)
    print("\n[Шаг 3] Тестирование client2 (ИП 'Ритейл')...")
    status, login_c2 = req("POST", f"{API_BASE}/api/auth/login", {"email": "client2", "password": "123"})
    token_c2 = login_c2.get("token")
    print(f"  Логин client2: HTTP {status}, token={bool(token_c2)}")

    status, app_c2 = req(
        "POST",
        f"{API_BASE}/api/applications",
        {
            "product_id": 2,
            "purpose": "Масштабирование розничной сети",
            "amount": 10000000,
            "term_months": 12,
        },
        token=token_c2,
    )
    app2_id = app_c2.get("id")
    print(f"  Создана заявка ID={app2_id}, status={app_c2.get('status')}")

    status, scored_c2 = req("POST", f"{API_BASE}/api/applications/{app2_id}/submit", token=token_c2)
    print(f"  Скоринг заявки ID={app2_id}:")
    print(f"    - Статус: {scored_c2.get('status')}")
    print(f"    - counter_amount: {scored_c2.get('approved_amount') or scored_c2.get('counter_amount')}")
    print(f"    - counter_term: {scored_c2.get('approved_term_months') or scored_c2.get('counter_term_months')}")
    logs_c2 = scored_c2.get("status_logs", [])
    print_timeline(f"Заявка ID={app2_id} (COUNTER_OFFER)", logs_c2)
    results["client2_logs_count"] = len(logs_c2)

    # 4. Тест client3 (Стартап)
    print("\n[Шаг 4] Тестирование client3 (Стартап)...")
    status, login_c3 = req("POST", f"{API_BASE}/api/auth/login", {"email": "client3", "password": "123"})
    token_c3 = login_c3.get("token")
    print(f"  Логин client3: HTTP {status}, token={bool(token_c3)}")

    status, app_c3 = req(
        "POST",
        f"{API_BASE}/api/applications",
        {
            "product_id": 2,
            "purpose": "Закупка первого оборудования",
            "amount": 500000,
            "term_months": 12,
        },
        token=token_c3,
    )
    app3_id = app_c3.get("id")
    print(f"  Создана заявка ID={app3_id}, status={app_c3.get('status')}")

    status, scored_c3 = req("POST", f"{API_BASE}/api/applications/{app3_id}/submit", token=token_c3)
    print(f"  Скоринг заявки ID={app3_id}:")
    print(f"    - Статус: {scored_c3.get('status')}")
    logs_c3 = scored_c3.get("status_logs", [])
    print_timeline(f"Заявка ID={app3_id} (DOCS_REQUIRED)", logs_c3)
    results["client3_logs_count"] = len(logs_c3)

    # 5. Подписание заявки client1
    print("\n[Шаг 5] Подписание одобренной заявки client1 (POST /sign с кодом 1234)...")
    status, signed_c1 = req("POST", f"{API_BASE}/api/applications/{app1_id}/sign", {"code": "1234"}, token=token_c1)
    print(f"  Результат подписания: HTTP {status}, status={signed_c1.get('status')}")
    logs_signed = signed_c1.get("status_logs", [])
    print_timeline(f"Заявка ID={app1_id} после подписания", logs_signed)
    results["signed_logs_count"] = len(logs_signed)

    status, dash_after = req("GET", f"{API_BASE}/api/dashboard", token=token_c1)
    balance_after = dash_after.get("company_info", {}).get("balance", 0)
    print(f"  Баланс до: {balance_before:,} ₽, баланс после: {balance_after:,} ₽ (дельта: +{balance_after - balance_before:,} ₽)")
    results["signing_delta"] = balance_after - balance_before

    # 6. Кабинет кредитного офицера
    print("\n[Шаг 6] Кабинет офицера (officer1)...")
    status, login_off = req("POST", f"{API_BASE}/api/auth/login", {"email": "officer1", "password": "123"})
    token_off = login_off.get("token")
    print(f"  Логин officer1: HTTP {status}, token={bool(token_off)}")

    status, off_apps = req("GET", f"{API_BASE}/api/officer/applications", token=token_off)
    apps_list = off_apps.get("applications", [])
    print(f"  Список заявок офицера: получено {len(apps_list)} заявок")

    # Ручное согласование заявки client3
    target_id = app3_id
    status, decision_resp = req(
        "POST",
        f"{API_BASE}/api/officer/applications/{target_id}/decision",
        {"decision": "APPROVED", "comment": "Согласовано под поручительство основателя"},
        token=token_off,
    )
    print(f"  Решение офицера по заявке ID={target_id}: HTTP {status}, новый статус={decision_resp.get('status')}")
    logs_officer = decision_resp.get("status_logs", [])
    print_timeline(f"Заявка ID={target_id} после решения офицера", logs_officer)
    results["officer_decision_status"] = decision_resp.get("status")
    results["officer_logs_count"] = len(logs_officer)

    # 7. Детальная проверка GET /api/applications/:id
    print(f"\n[Шаг 7] Проверка GET /api/applications/{app1_id} на наличие status_logs...")
    status, get_app1 = req("GET", f"{API_BASE}/api/applications/{app1_id}", token=token_c1)
    fetched_logs = get_app1.get("status_logs", [])
    print(f"  HTTP {status}, status_logs содержит {len(fetched_logs)} записей:")
    for log in fetched_logs:
        print(f"    - [{log.get('id')}] {log.get('from_status')} -> {log.get('to_status')}: {log.get('comment')}")
    results["get_application_status_logs_count"] = len(fetched_logs)

    print("\n" + "=" * 60)
    print("ИТОГОВЫЙ СТАТУС ДИАГНОСТИКИ:")
    print(json.dumps(results, indent=2, ensure_ascii=False))
    print("=" * 60)

    # Валидация всех ключевых критериев
    assert results["scoring_healthz"] is True, "Scoring healthz failed"
    assert results["openapi_yaml"] is True, "OpenAPI YAML not served"
    assert results["swagger_docs"] is True, "Swagger UI docs not served"
    assert results["client1_logs_count"] >= 3, f"Expected >= 3 logs for client1, got {results['client1_logs_count']}"
    assert results["signed_logs_count"] >= 4, f"Expected >= 4 logs after signing, got {results['signed_logs_count']}"
    assert results["signing_delta"] == 1500000, f"Expected +1,500,000 balance delta, got {results['signing_delta']}"
    assert results["officer_decision_status"] == "APPROVED", "Officer decision failed"
    assert results["officer_logs_count"] >= 4, f"Expected >= 4 logs for officer decision, got {results['officer_logs_count']}"
    assert results["get_application_status_logs_count"] >= 4, "GET /applications/:id must return status_logs"

    print("\n ВСЕ ПРОВЕРКИ УСПЕШНО ПРОЙДЕНЫ!")


if __name__ == "__main__":
    main()
