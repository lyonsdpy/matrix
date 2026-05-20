package main

import (
	"net"
	"testing"
)

func TestTain(m *testing.T) {
	ip := net.ParseIP("127.0.0.1")
	ip.To16()
}
