#!/bin/bash

echo "Testing DDD Example API..."
echo "=================================="

# Base URL
BASE_URL="http://localhost:8080/api/v1"

# Generate unique email to avoid duplicates
UNIQUE_EMAIL="test+$(date +%s)@example.com"

# Test Health Endpoint
echo "1. Testing Health Check..."
curl -s "$BASE_URL/health" | jq .
echo ""

# Test Create New User (with unique email)
echo "2. Testing Create New User..."
CREATE_USER_RESPONSE=$(curl -s -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Test User\",\"email\":\"$UNIQUE_EMAIL\",\"age\":25}")
echo "$CREATE_USER_RESPONSE" | jq .
NEW_USER_ID=$(echo "$CREATE_USER_RESPONSE" | jq -r '.data.id')
echo "Created user ID: $NEW_USER_ID"
echo ""

# Test Get User by ID
echo "3. Testing Get User by ID..."
if [ "$NEW_USER_ID" != "null" ] && [ "$NEW_USER_ID" != "" ]; then
  curl -s "$BASE_URL/users/$NEW_USER_ID" | jq .
else
  echo "Skipping - no user ID available"
fi
echo ""

# Test Get User Orders (using new user ID)
echo "5. Testing Get User Orders..."
if [ "$NEW_USER_ID" != "null" ] && [ "$NEW_USER_ID" != "" ]; then
  curl -s "$BASE_URL/orders/user/$NEW_USER_ID" | jq .
else
  echo "Skipping - no user ID available"
fi
echo ""

# Test Get User Total Spent (using new user ID)
echo "6. Testing Get User Total Spent..."
if [ "$NEW_USER_ID" != "null" ] && [ "$NEW_USER_ID" != "" ]; then
  curl -s "$BASE_URL/users/$NEW_USER_ID/total-spent" | jq .
else
  echo "Skipping - no user ID available"
fi
echo ""

# Test Create Order (using new user ID)
echo "7. Testing Create Order..."
if [ "$NEW_USER_ID" != "null" ] && [ "$NEW_USER_ID" != "" ]; then
  CREATE_ORDER_RESPONSE=$(curl -s -X POST "$BASE_URL/orders" \
    -H "Content-Type: application/json" \
    -d "{
      \"user_id\": \"$NEW_USER_ID\",
      \"items\": [
        {
          \"product_id\": \"prod-test\",
          \"product_name\": \"Test Product\",
          \"quantity\": 2,
          \"unit_price\": 99900,
          \"currency\": \"CNY\"
        }
      ]
    }")
  echo "$CREATE_ORDER_RESPONSE" | jq .
  NEW_ORDER_ID=$(echo "$CREATE_ORDER_RESPONSE" | jq -r '.data.id')
  echo "Created order ID: $NEW_ORDER_ID"
else
  echo "Skipping - no user ID available"
  NEW_ORDER_ID="null"
fi
echo ""

# Test Get Order by ID
echo "8. Testing Get Order by ID..."
if [ "$NEW_ORDER_ID" != "null" ] && [ "$NEW_ORDER_ID" != "" ] && [ "$NEW_ORDER_ID" != "null" ]; then
  curl -s "$BASE_URL/orders/$NEW_ORDER_ID" | jq .
else
  echo "Skipping - no order ID available"
fi
echo ""

# Test Update Order Status (use process instead to avoid optimistic lock)
echo "9. Testing Process Order..."
if [ "$NEW_ORDER_ID" != "null" ] && [ "$NEW_ORDER_ID" != "" ] && [ "$NEW_ORDER_ID" != "null" ]; then
  curl -s -X POST "$BASE_URL/orders/$NEW_ORDER_ID/process" \
    -H "Content-Type: application/json" \
    -d "{}" | jq .
else
  echo "Skipping - no order ID available"
fi
echo ""

echo "API Testing Complete!"
echo "=================================="
