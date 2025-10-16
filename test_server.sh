#!/bin/bash

echo "=========================================="
echo "🚀 RapidRTMP Server Test Suite"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

test_count=0
pass_count=0

run_test() {
    local name="$1"
    local command="$2"
    local expected_code="${3:-0}"
    
    test_count=$((test_count + 1))
    echo -e "${YELLOW}Test $test_count:${NC} $name"
    
    response=$(eval "$command" 2>&1)
    exit_code=$?
    
    if [ $exit_code -eq $expected_code ]; then
        echo -e "${GREEN}✓ PASS${NC}"
        echo "$response" | head -10
        pass_count=$((pass_count + 1))
    else
        echo -e "${RED}✗ FAIL${NC} (exit code: $exit_code, expected: $expected_code)"
        echo "$response"
    fi
    echo ""
}

# Test 1: Ping endpoint
run_test "Ping API" \
    "curl -s -w '\nHTTP_CODE:%{http_code}' http://localhost:8080/api/ping | grep -q 'pong' && echo 'Response contains pong'"

# Test 2: Health check
run_test "Health Check" \
    "curl -s http://localhost:8080/health | python3 -m json.tool"

# Test 3: Readiness check
run_test "Readiness Check" \
    "curl -s http://localhost:8080/ready | python3 -m json.tool"

# Test 4: List streams (should be empty)
run_test "List Streams (Empty)" \
    "curl -s http://localhost:8080/api/v1/streams | python3 -m json.tool"

# Test 5: Create publish token
echo -e "${YELLOW}Test 5:${NC} Generate Publish Token"
TOKEN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/publish \
    -H "Content-Type: application/json" \
    -d '{"stream_key":"test-stream-123"}')

echo "$TOKEN_RESPONSE" | python3 -m json.tool
TOKEN=$(echo "$TOKEN_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['token'])" 2>/dev/null)

if [ -n "$TOKEN" ]; then
    echo -e "${GREEN}✓ Token generated:${NC} $TOKEN"
    pass_count=$((pass_count + 1))
else
    echo -e "${RED}✗ Failed to generate token${NC}"
fi
test_count=$((test_count + 1))
echo ""

# Test 6: Metrics endpoint
run_test "Prometheus Metrics" \
    "curl -s http://localhost:8080/metrics | head -20 && echo '...'"

# Test 7: Check if RTMP port is listening
run_test "RTMP Port Listening" \
    "lsof -i :1935 | grep -q LISTEN && echo 'RTMP port 1935 is listening'"

# Test 8: Check if HTTP port is listening
run_test "HTTP Port Listening" \
    "lsof -i :8080 | grep -q LISTEN && echo 'HTTP port 8080 is listening'"

echo "=========================================="
echo -e "Results: ${GREEN}$pass_count/$test_count tests passed${NC}"
echo "=========================================="

if [ $pass_count -eq $test_count ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed${NC}"
    exit 1
fi
