#!/bin/bash

echo "🧪 Testing DDD Example API..."
echo "=================================="

# Base URL
BASE_URL="http://localhost:8080/api/v1"

# Test Health Endpoint
echo "1. Testing Health Check..."
curl -s "$BASE_URL/health" | jq .
echo ""

# Test Get All Users
echo "2. Testing Get All Users..."
curl -s "$BASE_URL/users" | jq .
echo ""

# Test Create New User
echo "3. Testing Create New User..."
CREATE_USER_RESPONSE=$(curl -s -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test User","email":"test@example.com","age":25}')
echo "$CREATE_USER_RESPONSE" | jq .
NEW_USER_ID=$(echo "$CREATE_USER_RESPONSE" | jq -r '.data.id')
echo "Created user ID: $NEW_USER_ID"
echo ""

# Test Get User by ID
echo "4. Testing Get User by ID..."
if [ "$NEW_USER_ID" != "null" ]; then
  curl -s "$BASE_URL/users/$NEW_USER_ID" | jq .
else
  echo "Skipping - no user ID available"
fi
echo ""

# Test Get User Orders
echo "5. Testing Get User Orders..."
curl -s "$BASE_URL/orders/user/user-1" | jq .
echo ""

# Test Get User Total Spent
echo "6. Testing Get User Total Spent..."
curl -s "$BASE_URL/users/user-1/total-spent" | jq .
echo ""

# Test Create Order
echo "7. Testing Create Order..."
CREATE_ORDER_RESPONSE=$(curl -s -X POST "$BASE_URL/orders" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-1",
    "items": [
      {
        "product_id": "prod-test",
        "product_name": "Test Product",
        "quantity": 2,
        "unit_price": 99900,
        "currency": "CNY"
      }
    ]
  }')
echo "$CREATE_ORDER_RESPONSE" | jq .
NEW_ORDER_ID=$(echo "$CREATE_ORDER_RESPONSE" | jq -r '.data.id')
echo "Created order ID: $NEW_ORDER_ID"
echo ""

# Test Get Order by ID
echo "8. Testing Get Order by ID..."
if [ "$NEW_ORDER_ID" != "null" ]; then
  curl -s "$BASE_URL/orders/$NEW_ORDER_ID" | jq .
else
  echo "Skipping - no order ID available"
fi
echo ""

# Test Update Order Status
echo "9. Testing Update Order Status..."
if [ "$NEW_ORDER_ID" != "null" ]; then
  curl -s -X PUT "$BASE_URL/orders/status" \
    -H "Content-Type: application/json" \
    -d "{\"order_id\": \"$NEW_ORDER_ID\", \"status\": \"CONFIRMED\"}" | jq .
else
  echo "Skipping - no order ID available"
fi
echo ""

echo "🎉 API Testing Complete!"
echo "=================================="