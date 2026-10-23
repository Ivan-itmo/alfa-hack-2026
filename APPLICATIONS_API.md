# Кредитный конвейер: контракт для фронтенда

Все пути ниже требуют `Authorization: Bearer <token>` из `POST /api/auth/login`.
Клиент видит только собственные заявки своей компании. Офицер видит все заявки.
JSON-запросы отправляются с `Content-Type: application/json`; исключение — загрузка файла.
Деньги — целые рубли, ставка — годовые проценты (`20` = 20%), DSTI — доля (`0.5` = 50%).

## Каталог и создание заявки (client)

`GET /api/products` → `200`:

```json
{
  "products": [
    {"id":1,"code":"overdraft","name":"Овердрафт","interest_rate":18,"min_term_months":1,"max_term_months":12,"min_amount":10000,"max_amount":5000000},
    {"id":2,"code":"revolving_credit","name":"Оборотный кредит","interest_rate":20,"min_term_months":3,"max_term_months":36,"min_amount":10000,"max_amount":30000000},
    {"id":3,"code":"term_loan","name":"Инвестиционный кредит","interest_rate":16,"min_term_months":12,"max_term_months":60,"min_amount":10000,"max_amount":100000000}
  ]
}
```

`POST /api/applications` → `201`, объект заявки со статусом `DRAFT`:

```json
{"product_id":2,"purpose":"Закупка товара","amount":1500000,"term_months":12}
```

`product_id` — число из каталога; цель обязательна (до 2000 байт), сумма и срок должны входить в ограничения продукта.
`company_id`, `user_id`, статус и финансовые показатели берутся на сервере.

## Получение и отправка

`GET /api/applications/:id` → `200`, полный объект заявки. Чужой ID возвращает `404`.
Формат ответа одинаков для создания, получения и всех успешных действий с заявкой:

```json
{
  "id":2,
  "user_id":1,
  "company_id":1,
  "product_id":2,
  "purpose":"Закупка товара",
  "status":"APPROVED",
  "amount":1500000,
  "term_months":12,
  "requested_amount":1500000,
  "requested_term_months":12,
  "approved_amount":1500000,
  "approved_term_months":12,
  "monthly_payment":138952,
  "interest_rate":20,
  "score_value":800,
  "dsti":0.31492,
  "rejection_reason":"",
  "ai_explanation":"Демонстрационная оценка: прибыль позволяет обслуживать кредит.",
  "officer_comment":"",
  "scoring_source":"fallback",
  "created_at":"2026-10-23T12:31:00Z",
  "updated_at":"2026-10-23T12:31:01Z"
}
```

`amount` и `term_months` сохранены для совместимости и равны первоначально запрошенным значениям.
Для договора используйте `approved_amount`, `approved_term_months`, `monthly_payment`, `interest_rate`.
До решения банка эти поля равны нулю; текстовые поля могут быть пустыми.
`scoring_source` — `ml`, `fallback` или пустая строка до скоринга; `score_value` — шкала 0–1000.
`officer_comment` содержит последний комментарий сотрудника.

`POST /api/applications/:id/submit` — без тела. Только из `DRAFT`:

```text
DRAFT → DATA_ENRICHMENT → SCORING_IN_PROGRESS
  → APPROVED | COUNTER_OFFER | DOCS_REQUIRED | MANUAL_REVIEW | REJECTED
```

Обогащение использует локальный мок ФНС (выручка и возраст из записи компании).
Вызов синхронный: ответ `200` содержит итоговое решение. Во время запроса другой GET может увидеть промежуточный статус.
Для обновлений от офицера фронт повторяет GET заявки; WebSocket/SSE не используется.
Дашборд возвращает последнюю незавершенную заявку; `REJECTED`, `ISSUED`, `CANCELLED_BY_CLIENT` исключаются.

## Встречные условия

`POST /api/applications/:id/counter-offer` — только из `COUNTER_OFFER`:

```json
{"action":"accept"}
```

`accept` → `APPROVED`, одобренные банком условия сохраняются; `reject` → `CANCELLED_BY_CLIENT`.
Ответ `200` — обновленная заявка. Исходные запрошенные параметры остаются для сравнения.

## Дозапрос документов

`POST /api/applications/:id/documents` — только из `DOCS_REQUIRED`.
`multipart/form-data`, обязательное поле `file`, непустой синтетический файл до 5 МиБ, запрос до 6 МиБ:

```sh
curl -X POST http://localhost:8080/api/applications/2/documents \
  -H "Authorization: Bearer $TOKEN" \
  -F 'file=@statement.txt'
```

Файл сохраняется в PostgreSQL в `application_documents`, не исполняется и не разбирается.
Переход: `DOCS_REQUIRED → DOCS_UPLOADED → SCORING_IN_PROGRESS → решение`.
Ответ `200` — результат повторного скоринга. В локальных правилах молодой бизнес после загрузки переходит на ручную проверку.
Повторный запрос допустим, только если новое решение снова `DOCS_REQUIRED`.

## Подписание и виртуальная выдача

`POST /api/applications/:id/sign` — только из `APPROVED`:

```json
{"code":"1234"}
```

Показывайте код `1234` прямо на экране; реальных SMS нет.
`APPROVED → PENDING_SIGNATURE → ISSUED` и увеличение `company.balance` на `approved_amount`
происходят в одной транзакции. Ответ `200` — заявка в `ISSUED`.
Баланс доступен в `GET /api/dashboard` → `company_info.balance`.
Повторное или конкурентное подписание возвращает `409`; деньги зачисляются один раз.
`PENDING_SIGNATURE` внутри атомарной транзакции обычно не виден другому запросу.

## Кабинет риск-офицера (officer)

`GET /api/officer/applications` → `200`:

```json
{"applications":[]}
```

Элементы — полные объекты заявок, порядок по ID от новых к старым.
Фильтр одного или нескольких статусов:
`GET /api/officer/applications?status=MANUAL_REVIEW,SCORING_IN_PROGRESS`.
Неизвестный статус → `400`. Без фильтра возвращаются все заявки.

`POST /api/officer/applications/:id/decision`:

```json
{"decision":"COUNTER_OFFER","amount":1000000,"term_months":24,"comment":"Предлагаем снизить долговую нагрузку"}
```

Другие примеры:

```json
{"decision":"APPROVED","comment":"Финансовые показатели проверены"}
```

```json
{"decision":"DOCS_REQUIRED","comment":"Приложите выписку за последние 6 месяцев"}
```

```json
{"decision":"REJECTED","comment":"Недостаточно прибыли для обслуживания кредита"}
```

Допустимые исходные статусы: `SCORING_IN_PROGRESS`, `MANUAL_REVIEW`, `DOCS_REQUIRED`, `COUNTER_OFFER`, `APPROVED`, `REJECTED`.
Черновики, обогащение, загрузка и завершенные выдачей/отменой заявки не редактируются.
Комментарий обязателен (до 10000 байт). Для `APPROVED`/`COUNTER_OFFER` пропущенные сумма/срок берутся из запрошенных;
условия проверяются по каталогу, сумма не может превышать запрошенную. Платеж рассчитывает сервер.
Для `REJECTED`/`DOCS_REQUIRED` сумма/срок не передаются. При отказе комментарий становится `rejection_reason`.
Текст решения также доступен клиенту в `ai_explanation`, авторский комментарий — в `officer_comment`.
Ответ `200` — обновленная заявка. Уже выполняющийся скоринг не перезаписывает ручное решение (его запрос завершится с `409`).

## Интеграция ML

`SCORING_BASE_URL` — базовый URL сервиса, без пути (например `http://ml:8000` внутри Docker).
Бэкенд вызывает `POST {SCORING_BASE_URL}/api/v1/scoring/evaluate`, таймаут 4 секунды.
Пустая переменная сразу включает локальные правила. Сетевые ошибки, не-200 и некорректные ответы также включают их.
Fallback не требует запуска AI API и отмечается `scoring_source: "fallback"`.

Тело запроса:

```json
{
  "company_inn":"7701000001",
  "monthly_revenue":2000000,
  "monthly_net_profit":600000,
  "company_age_months":102,
  "requested_amount":1500000,
  "requested_term_months":12,
  "product_rate":20,
  "bki_active_loans_monthly_payment":50000,
  "bki_delinquency_days":0,
  "has_bankruptcy":false
}
```

Ожидаемый ответ ML (прямой объект, без оболочки):

```json
{
  "decision":"APPROVED",
  "approved_amount":1500000,
  "approved_term_months":12,
  "monthly_payment":138952,
  "interest_rate":20,
  "score_value":800,
  "dsti":0.31492,
  "rejection_reason":"",
  "ai_explanation":"Прибыль компании позволяет обслуживать запрошенный кредит."
}
```

`decision`: `APPROVED | COUNTER_OFFER | DOCS_REQUIRED | MANUAL_REVIEW | REJECTED`.
`ai_explanation` обязательна, для отказа обязательна `rejection_reason`.
Для положительного решения обязательны положительные сумма, срок и платеж, корректная ставка и условия продукта.
Для остальных решений одобренные сумма/срок/платеж должны быть нулевыми.
DSTI неотрицательный; ставка от 0 до 100; score от 0 до 1000.
Файл остается в бэкенде; текущий ML-контракт передает только финансовые признаки.

Локальные правила применяются в таком порядке:

1. Банкротство, просрочка >30 дней или прибыль ≤0 → отказ.
2. Возраст <6 месяцев без загруженного документа → дозапрос.
3. Возраст <6 месяцев с документом либо просрочка 1–30 дней → ручная проверка.
4. DSTI >50% → уменьшение суммы до доступного платежа; если сумма <10000 рублей → отказ.
5. Иначе → одобрение. Платеж аннуитетный, округляется вверх до рубля.

Демо-сценарии без ML: `client1` + 1 500 000/12 → одобрение; `client1` + 10 000 000/12 → встречные условия;
`client3` + 100 000/12 → документы → ручная проверка → решение офицера.
Для проверки отказа у компании можно задать `has_bankruptcy=true` в тестовой БД.

## Ошибки

Общий формат: `{"error":"Понятное сообщение"}`.

| Код | Значение |
|---|---|
| 400 | Невалидный JSON/параметры/код/файл/статус фильтра |
| 401 | Нет валидного JWT |
| 403 | Неверная роль |
| 404 | Заявка отсутствует или принадлежит другому клиенту |
| 409 | Действие недопустимо в текущем статусе или конфликтует с другим действием |
| 413 | Превышен размер JSON (1 МиБ) или файла/запроса |
| 500 | Ошибка БД/обработки |

Миграции при старте переводят старые `SUBMITTED`/`SCORING` в новые статусы, переносят исходные сумму/срок
и дополняют финансовые признаки трех известных демо-компаний. Повторный запуск не обнуляет баланс.
