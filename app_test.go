package main

import "testing"

func TestIsRegisteredPort(t *testing.T) {
	if !isRegisteredPort(1024) {
		t.Fatal("expected 1024 to be accepted")
	}
	if !isRegisteredPort(49151) {
		t.Fatal("expected 49151 to be accepted")
	}
	if isRegisteredPort(1023) {
		t.Fatal("expected 1023 to be rejected")
	}
	if isRegisteredPort(49152) {
		t.Fatal("expected 49152 to be rejected")
	}
	if isRegisteredPort(0) {
		t.Fatal("expected 0 to be rejected")
	}
}
