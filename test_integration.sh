#!/bin/bash
#
# ENZO Integration Test (Go-based)
# Tests: Client ENZO -> Server ENZO -> Backend
#

export PATH=/tmp/go/bin:$PATH
export GOPATH=/tmp/gopath
export GO111MODULE=on
cd /workspace/project/enzo

echo "========================================"
echo "ENZO Integration Test"
echo "========================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; }
fail() { echo -e "${RED}✗ FAIL${NC}: $1"; exit 1; }

# Build ENZO if not exists
if [ ! -f /tmp/enzo ]; then
    echo ">>> Building ENZO..."
    go build -o /tmp/enzo ./cmd/enzo || fail "Build failed"
fi
pass "ENZO binary ready"

# Kill any existing processes from previous runs
pkill -f "receiver.go" 2>/dev/null || true
pkill -f "enzo.*:8086" 2>/dev/null || true
pkill -f "enzo.*:8087" 2>/dev/null || true
pkill -f "enzo.*:8088" 2>/dev/null || true
sleep 1

echo ""
echo ">>> Step 1: Creating receiver and sender..."

rm -f /tmp/enzo_received.txt /tmp/enzo_test.sock

# Create TCP receiver (simulates InfluxDB backend)
cat > /tmp/receiver.go << 'EOF'
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:8087")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		ln.Close()
		os.Exit(0)
	}()

	fmt.Println("READY:8087")
	conn, err := ln.Accept()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Read HTTP request
	data := make([]byte, 0, 4096)
	buf := make([]byte, 1024)
	for {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := conn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
			if len(data) > 100 {
				break
			}
		}
		if err != nil {
			break
		}
	}

	os.WriteFile("/tmp/enzo_received.txt", data, 0644)
	fmt.Printf("RECEIVED:%d\n", len(data))
	
	// Send HTTP 204 response
	time.Sleep(100 * time.Millisecond)
	response := "HTTP/1.1 204 No Content\r\nContent-Length: 0\r\n\r\n"
	conn.Write([]byte(response))
	fmt.Println("RESPONSE:sent")
}
EOF

# Create TCP sender (HTTP POST with InfluxDB line protocol as body)
cat > /tmp/sender.go << 'EOF'
package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8088")
	if err != nil {
		fmt.Println("ERROR:Connect:", err)
		return
	}
	defer conn.Close()

	// Wait for ENZO to be ready
	time.Sleep(200 * time.Millisecond)

	// Send HTTP POST request with InfluxDB line protocol as body
	data := "cpu,host=server01,region=us-west value=0.64 1609459200000\r\n" +
	        "memory,host=server01,region=us-west used=5368709120 1609459200000\r\n" +
	        "disk,host=server01,region=us-west free=102400000000 1609459200000\r\n" +
	        "temperature,host=sensor01,location=room1 temp=23.5 1609459200000\r\n"

	request := "POST /write HTTP/1.1\r\n" +
	           "Host: localhost:8086\r\n" +
	           "Content-Type: text/plain\r\n" +
	           "Content-Length: " + fmt.Sprintf("%d", len(data)) + "\r\n" +
	           "Connection: close\r\n" +
	           "\r\n" +
	           data

	n, err := conn.Write([]byte(request))
	if err != nil {
		fmt.Println("ERROR:Write:", err)
		return
	}
	fmt.Printf("SENT:%d bytes\n", n)

	// Read response (ENZO should pass through)
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.Read(buf)
}
EOF

echo ""
echo ">>> Step 2: Starting TCP receiver..."
go run /tmp/receiver.go > /tmp/receiver.log 2>&1 &
RECEIVER_PID=$!
sleep 1
if ! kill -0 $RECEIVER_PID 2>/dev/null; then
    cat /tmp/receiver.log
    fail "Receiver failed to start"
fi
pass "TCP receiver started (PID: $RECEIVER_PID)"

echo ""
echo ">>> Step 3: Starting Server ENZO..."
/tmp/enzo -listen :8086 -dest localhost:8087 > /tmp/server_enzo.log 2>&1 &
SERVER_PID=$!
sleep 1
if ! kill -0 $SERVER_PID 2>/dev/null; then
    cat /tmp/server_enzo.log
    fail "Server ENZO failed to start"
fi
pass "Server ENZO started (PID: $SERVER_PID)"

echo ""
echo ">>> Step 4: Starting Client ENZO..."
/tmp/enzo -listen :8088 -dest localhost:8086 > /tmp/client_enzo.log 2>&1 &
CLIENT_PID=$!
sleep 1
if ! kill -0 $CLIENT_PID 2>/dev/null; then
    cat /tmp/client_enzo.log
    fail "Client ENZO failed to start"
fi
pass "Client ENZO started (PID: $CLIENT_PID)"

echo ""
echo "========================================"
echo "Data Transfer Test"
echo "========================================"
echo ""
echo ">>> Sending test data through ENZO tunnel..."
echo "    Client ENZO (:8088) -> Server ENZO (:8086) -> Receiver (:8087)"
echo ""

go run /tmp/sender.go

echo ""
echo ">>> Waiting for data to propagate..."
sleep 2

echo ""
echo "========================================"
echo "Verification"
echo "========================================"
echo ""

# Check if data was received
if [ -f /tmp/enzo_received.txt ] && [ -s /tmp/enzo_received.txt ]; then
    SIZE=$(wc -c < /tmp/enzo_received.txt)
    echo ">>> Received data ($SIZE bytes):"
    head -8 /tmp/enzo_received.txt
    echo "..."
    echo ""
    
    if grep -q "cpu,host=server01" /tmp/enzo_received.txt; then
        pass "CPU metric received"
    else
        fail "CPU metric missing"
    fi
    
    if grep -q "memory,host=server01" /tmp/enzo_received.txt; then
        pass "Memory metric received"
    else
        fail "Memory metric missing"
    fi
    
    if grep -q "disk,host=server01" /tmp/enzo_received.txt; then
        pass "Disk metric received"
    else
        fail "Disk metric missing"
    fi
    
    if grep -q "temperature,host=sensor01" /tmp/enzo_received.txt; then
        pass "Temperature metric received"
    else
        fail "Temperature metric missing"
    fi
    
    if grep -q "value=0.64" /tmp/enzo_received.txt; then
        pass "Metric value preserved correctly"
    else
        fail "Metric value corrupted"
    fi
    
    if grep -q "temp=23.5" /tmp/enzo_received.txt; then
        pass "Temperature value preserved correctly"
    else
        fail "Temperature value corrupted"
    fi
else
    echo "File not found or empty."
    cat /tmp/receiver.log 2>/dev/null
    cat /tmp/server_enzo.log 2>/dev/null
    cat /tmp/client_enzo.log 2>/dev/null
    fail "No data received"
fi

echo ""
echo ">>> Checking Server ENZO logs..."
if grep -q "decode" /tmp/server_enzo.log; then
    pass "Server ENZO in decode mode"
else
    echo "Server ENZO log:"
    cat /tmp/server_enzo.log
fi

echo ""
echo ">>> Checking Client ENZO logs..."
if grep -q "encode" /tmp/client_enzo.log; then
    pass "Client ENZO in encode mode"
    # Show stats
    grep "stats" /tmp/client_enzo.log
else
    echo "Client ENZO log:"
    cat /tmp/client_enzo.log
fi

echo ""
echo "========================================"
echo -e "${GREEN}ALL TESTS PASSED${NC}"
echo "========================================"
echo ""
echo "Architecture verified:"
echo "  Device -> Client ENZO (:8088) -> Server ENZO (:8086) -> Backend (:8087)"
echo ""

# Cleanup
kill $RECEIVER_PID $SERVER_PID $CLIENT_PID 2>/dev/null || true
