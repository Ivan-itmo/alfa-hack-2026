#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
echo "=========================================="
echo "Starting Smoke Test against: $BASE_URL"
echo "=========================================="

# 1. Login as client1
echo -n "1. Login as client1 (POST /api/auth/login)... "
LOGIN_RESP=$(curl -s -f -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"client1","password":"123"}')
CLIENT_TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$CLIENT_TOKEN" ]; then
  echo "FAILED"
  echo "Response: $LOGIN_RESP"
  exit 1
fi
echo "SUCCESS (token received)"

# 2. Get Dashboard
echo -n "2. Get Dashboard (GET /api/dashboard)... "
DASHBOARD_RESP=$(curl -s -f -X GET "$BASE_URL/api/dashboard" \
  -H "Authorization: Bearer $CLIENT_TOKEN")
echo "SUCCESS"

# 3. Get Products
echo -n "3. Get Products catalog (GET /api/products)... "
PRODUCTS_RESP=$(curl -s -f -X GET "$BASE_URL/api/products" \
  -H "Authorization: Bearer $CLIENT_TOKEN")
echo "SUCCESS"

# 4. Create Application in DRAFT
echo -n "4. Create Application (POST /api/applications)... "
CREATE_RESP=$(curl -s -f -X POST "$BASE_URL/api/applications" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id":2,"purpose":"Пополнение оборотных средств для закупки товара","amount":1500000,"term_months":12}')
APP_ID=$(echo "$CREATE_RESP" | grep -o '"id":[0-9]*' | head -n1 | cut -d':' -f2)

if [ -z "$APP_ID" ]; then
  echo "FAILED"
  echo "Response: $CREATE_RESP"
  exit 1
fi
echo "SUCCESS (Application ID: $APP_ID created in DRAFT)"

# 5. Submit for Scoring
echo -n "5. Submit Application for scoring (POST /api/applications/$APP_ID/submit)... "
SUBMIT_RESP=$(curl -s -f -X POST "$BASE_URL/api/applications/$APP_ID/submit" \
  -H "Authorization: Bearer $CLIENT_TOKEN")
STATUS=$(echo "$SUBMIT_RESP" | grep -o '"status":"[^"]*' | cut -d'"' -f4)
echo "SUCCESS (Status after scoring: $STATUS)"

# 6. Sign Contract
echo -n "6. Sign Contract with PEP code 1234 (POST /api/applications/$APP_ID/sign)... "
SIGN_RESP=$(curl -s -f -X POST "$BASE_URL/api/applications/$APP_ID/sign" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code":"1234"}')
SIGN_STATUS=$(echo "$SIGN_RESP" | grep -o '"status":"[^"]*' | cut -d'"' -f4)
echo "SUCCESS (Status after signing: $SIGN_STATUS)"

# 7. Login as officer1
echo -n "7. Login as officer1 (POST /api/auth/login)... "
OFFICER_LOGIN_RESP=$(curl -s -f -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"officer1","password":"123"}')
OFFICER_TOKEN=$(echo "$OFFICER_LOGIN_RESP" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo "SUCCESS (officer token received)"

# 8. Officer Applications List
echo -n "8. Officer Applications list (GET /api/officer/applications)... "
OFFICER_APPS_RESP=$(curl -s -f -X GET "$BASE_URL/api/officer/applications" \
  -H "Authorization: Bearer $OFFICER_TOKEN")
echo "SUCCESS"

echo "=========================================="
echo "ALL SMOKE TESTS PASSED SUCCESSFULLY!"
echo "=========================================="
