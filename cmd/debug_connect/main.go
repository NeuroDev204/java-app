package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

func main() {
	host := "112.213.85.108"
	port := 19129

	fmt.Printf("Testing connection to %s:%d\n\n", host, port)

	// Method 1: Try reading first
	fmt.Println("=== Method 1: Read first, then write ===")
	test1(host, port)

	// Method 2: Try writing first
	fmt.Println("\n=== Method 2: Write first, then read ===")
	test2(host, port)

	// Method 3: Try with no delay
	fmt.Println("\n=== Method 3: Write + immediate read ===")
	test3(host, port)
}

func test1(host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		fmt.Printf("  Dial failed: %v\n", err)
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		fmt.Printf("  Read failed: %v\n", err)
		return
	}
	fmt.Printf("  Received %d bytes: % x\n", n, buf[:n])
}

func test2(host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		fmt.Printf("  Dial failed: %v\n", err)
		return
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	// Send login packet (opcode 1)
	// Format: opcode(1) + length(2) + payload
	payload := new(bytes.Buffer)
	payload.WriteString("vjncible@gmail.com") // username
	payload.WriteString("660007")              // password
	payload.WriteString("4.0.2")               // version

	buf := new(bytes.Buffer)
	buf.WriteByte(1) // opcode
	binary.Write(buf, binary.BigEndian, uint16(payload.Len())) // length
	buf.Write(payload.Bytes())

	fmt.Printf("  Sending: % x\n", buf.Bytes()[:min(50, buf.Len())])
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		fmt.Printf("  Write failed: %v\n", err)
		return
	}

	// Read response
	resp := make([]byte, 2048)
	n, err := conn.Read(resp)
	if err != nil && err != io.EOF {
		fmt.Printf("  Read failed: %v\n", err)
		return
	}
	fmt.Printf("  Received %d bytes: % x\n", n, resp[:n])
}

func test3(host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		fmt.Printf("  Dial failed: %v\n", err)
		return
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	// Send raw bytes without length prefix first
	raw := []byte{0x01, 0x00, 0x10} // opcode 1, some length
	fmt.Printf("  Sending raw: % x\n", raw)
	_, err = conn.Write(raw)
	if err != nil {
		fmt.Printf("  Write failed: %v\n", err)
		return
	}

	// Read response
	resp := make([]byte, 2048)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(resp)
	if err != nil && err != io.EOF {
		fmt.Printf("  Read failed: %v\n", err)
		return
	}
	if n == 0 {
		fmt.Printf("  No response\n")
		return
	}
	fmt.Printf("  Received %d bytes: % x\n", n, resp[:n])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
