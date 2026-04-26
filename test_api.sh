#!/bin/bash

# Test script for Forex Trading Simulator Web Interface

BASE_URL="http://localhost:8080"
EMAIL="user1@test.com"
PASSWORD="User1234"

echo "========================================="
echo "Forex Trading Simulator - API Test Tool"
echo "========================================="

# 1. Login
echo ""
echo ">>> 1. Testing Login..."
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

TOKEN=$(echo "$LOGIN_RESP" | grep -o '"access_token":"[^"]*' | sed 's/"access_token":"//')

if [ -z "$TOKEN" ]; then
  echo "❌ Login FAILED"
  echo "Response: $LOGIN_RESP"
  exit 1
fi

echo "✅ Login SUCCESS"
echo "Token: ${TOKEN:0:50}..."

# 2. Get Profile
echo ""
echo ">>> 2. Testing Get Profile..."
PROFILE=$(curl -s "$BASE_URL/api/v1/users/me" -H "Authorization: Bearer $TOKEN")
if echo "$PROFILE" | grep -q "email"; then
  echo "✅ Profile: $(echo $PROFILE | grep -o '"email":"[^"]*')"
else
  echo "❌ Profile FAILED"
fi

# 3. Get Accounts
echo ""
echo ">>> 3. Testing Get Accounts..."
ACCOUNTS=$(curl -s "$BASE_URL/api/v1/trading/accounts" -H "Authorization: Bearer $TOKEN")
if echo "$ACCOUNTS" | grep -q "account_number"; then
  # Use account with highest balance (usually the main trading account)
  ACCOUNT_ID=$(echo "$ACCOUNTS" | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')
  # Or try to find account with highest balance
  ACCOUNT_ID=$(echo "$ACCOUNTS" | grep -o '"id":[0-9]*,"user_id":[0-9]*,"account_number":"DEMO[^"]*","balance":[0-9.]*' | sort -t: -k4 -rn | head -1 | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
  if [ -z "$ACCOUNT_ID" ]; then
    ACCOUNT_ID=5  # Default to account 5 which is usually the main account
  fi
  echo "✅ Using Account ID: $ACCOUNT_ID"
  BALANCE=$(echo "$ACCOUNTS" | grep -o '"balance":[0-9.]*' | head -1)
  echo "   Balance: $BALANCE"
else
  echo "❌ Get Accounts FAILED"
  ACCOUNT_ID="5"
fi

# 4. Get Positions
echo ""
echo ">>> 4. Testing Get Positions..."
POSITIONS=$(curl -s "$BASE_URL/api/v1/trading/positions?account_id=$ACCOUNT_ID" -H "Authorization: Bearer $TOKEN")
if echo "$POSITIONS" | grep -q "id"; then
  POSITION_COUNT=$(echo "$POSITIONS" | grep -o '"id":' | wc -l)
  echo "✅ Found $POSITION_COUNT positions"
else
  echo "✅ No open positions"
fi

# 5. Get Trade History
echo ""
echo ">>> 5. Testing Get Trade History..."
HISTORY=$(curl -s "$BASE_URL/api/v1/trading/trades?account_id=$ACCOUNT_ID" -H "Authorization: Bearer $TOKEN")
if echo "$HISTORY" | grep -q "id"; then
  TRADE_COUNT=$(echo "$HISTORY" | grep -o '"id":' | wc -l)
  echo "✅ Found $TRADE_COUNT trades in history"
else
  echo "✅ No trade history"
fi

# 6. Execute Trade
echo ""
echo ">>> 6. Testing Execute Trade..."
TRADE_RESP=$(curl -s -X POST "$BASE_URL/api/v1/trading/trade" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"account_id\":$ACCOUNT_ID,\"currency_pair\":\"EUR/USD\",\"type\":\"BUY\",\"quantity\":1000,\"entry_price\":1.085}")

if echo "$TRADE_RESP" | grep -q '"id":'; then
  TRADE_ID=$(echo "$TRADE_RESP" | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')
  echo "✅ Trade executed! Trade ID: $TRADE_ID"
else
  echo "❌ Trade failed: $(echo $TRADE_RESP | head -c 100)"
fi

# 7. Get Positions After Trade
echo ""
echo ">>> 7. Testing Get Positions (after trade)..."
POSITIONS=$(curl -s "$BASE_URL/api/v1/trading/positions?account_id=$ACCOUNT_ID" -H "Authorization: Bearer $TOKEN")
if echo "$POSITIONS" | grep -q "id"; then
  POSITION_COUNT=$(echo "$POSITIONS" | grep -o '"id":' | wc -l)
  echo "✅ Found $POSITION_COUNT positions"
  # Get the first position ID to close
  POSITION_TO_CLOSE=$(echo "$POSITIONS" | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')
else
  echo "✅ No open positions"
  POSITION_TO_CLOSE=""
fi

# 8. Close Position
if [ ! -z "$POSITION_TO_CLOSE" ]; then
  echo ""
  echo ">>> 8. Testing Close Position (ID: $POSITION_TO_CLOSE)..."
  CLOSE_RESP=$(curl -s -X DELETE "$BASE_URL/api/v1/trading/positions/$POSITION_TO_CLOSE" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"exit_price":1.09}')
  
  if echo "$CLOSE_RESP" | grep -q '"status":"closed"'; then
    echo "✅ Position closed successfully"
    PNL=$(echo "$CLOSE_RESP" | grep -o '"pnl":[0-9.-]*' | head -1)
    echo "   PnL: $PNL"
  else
    echo "❌ Close position failed"
    echo "   Response: $(echo $CLOSE_RESP | head -c 200)"
  fi
fi

# 9. Predictions
echo ""
echo ">>> 9. Testing Predictions..."
PRED_RESP=$(curl -s -X POST "$BASE_URL/api/v1/predictions/predict" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"currency_pair":"EUR/USD","periods":10}')

if echo "$PRED_RESP" | grep -q "signal"; then
  SIGNAL=$(echo "$PRED_RESP" | grep -o '"signal":"[^"]*"')
  CONF=$(echo "$PRED_RESP" | grep -o '"confidence":[0-9.]*')
  echo "✅ Prediction: $SIGNAL, $CONF"
else
  echo "⚠️ Prediction: $(echo $PRED_RESP | head -c 100)"
fi

# 10. Get Balance After Trades
echo ""
echo ">>> 10. Testing Get Balance (after trades)..."
BALANCE_RESP=$(curl -s "$BASE_URL/api/v1/trading/accounts/$ACCOUNT_ID/balance" -H "Authorization: Bearer $TOKEN")
if echo "$BALANCE_RESP" | grep -q "balance"; then
  echo "✅ Current Balance: $(echo $BALANCE_RESP | grep -o '"balance":[0-9.]*')"
else
  echo "❌ Get balance failed"
fi

# 11. Currency Pairs (Public)
echo ""
echo ">>> 11. Testing Currency Pairs (Public)..."
PAIRS=$(curl -s "$BASE_URL/api/v1/currency-pairs")
if echo "$PAIRS" | grep -q "symbol"; then
  PAIR_COUNT=$(echo "$PAIRS" | grep -o '"symbol":"' | wc -l)
  echo "✅ Found $PAIR_COUNT currency pairs"
else
  echo "❌ Currency pairs failed"
fi

echo ""
echo "========================================="
echo "All tests completed!"
echo "========================================="
