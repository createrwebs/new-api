package controller

import (
	"strings"
	"testing"
)

func TestGenPromptPayPayload(t *testing.T) {
	phone := "0639304353"
	
	// Test static QR without amount
	payloadStatic := GenPromptPayPayload(phone, 0)
	if !strings.HasPrefix(payloadStatic, "000201010211") {
		t.Fatalf("expected static prefix, got %s", payloadStatic)
	}
	if !strings.Contains(payloadStatic, "0066639304353") {
		t.Fatalf("expected phone in payload, got %s", payloadStatic)
	}
	if len(payloadStatic) < 60 {
		t.Fatalf("payload length suspiciously short: %d", len(payloadStatic))
	}

	// Test dynamic QR with amount 100 THB
	payloadDynamic := GenPromptPayPayload(phone, 100.0)
	if !strings.HasPrefix(payloadDynamic, "000201010212") {
		t.Fatalf("expected dynamic prefix, got %s", payloadDynamic)
	}
	if !strings.Contains(payloadDynamic, "5406100.00") {
		t.Fatalf("expected amount tag 5406100.00, got %s", payloadDynamic)
	}
	if !strings.Contains(payloadDynamic, "5802TH") {
		t.Fatalf("expected country TH, got %s", payloadDynamic)
	}
}
