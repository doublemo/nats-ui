package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestSubjectValidation(t *testing.T) {
	for _, test := range []struct {
		subject        string
		wildcard, want bool
	}{
		{"orders.created", false, true}, {"orders.*", true, true}, {">", true, true},
		{"orders.>", true, true}, {"orders.*", false, false}, {"orders.>.new", true, false},
		{"orders..new", true, false}, {"orders.abc*", true, false}, {"orders\nnew", true, false},
		{"", true, false}, {"orders.\x00", true, false},
	} {
		if got := validSubject(test.subject, test.wildcard); got != test.want {
			t.Errorf("subject %q: got %v want %v", test.subject, got, test.want)
		}
	}
}

func TestMessagePreviewPreservesBinaryAndBoundsSize(t *testing.T) {
	data := []byte{0xff, 0x00, 0xfe}
	record := messageRecord(&nats.Msg{Subject: "binary", Data: data, Reply: "reply"})
	decoded, err := base64.StdEncoding.DecodeString(record.Payload)
	if err != nil || !bytes.Equal(decoded, data) || record.Encoding != "base64" || record.Reply != "reply" {
		t.Fatalf("binary payload was not preserved: %+v", record)
	}
	record = messageRecord(&nats.Msg{Data: bytes.Repeat([]byte("x"), 100000)})
	if !record.Truncated || record.Bytes != 100000 || len(record.Payload) != 65536 {
		t.Fatalf("preview must be bounded: bytes=%d preview=%d truncated=%v", record.Bytes, len(record.Payload), record.Truncated)
	}
}

func TestRejectWildcardPublishBeforeConnecting(t *testing.T) {
	service := &NATSService{}
	if _, err := service.SendMessage(context.Background(), "", MessageInput{Subject: "orders.*"}, false); err == nil {
		t.Fatal("wildcard publish must fail")
	}
	if _, err := service.SubscribeMessages(context.Background(), "", "orders.>.invalid", ""); err == nil {
		t.Fatal("invalid subscription must fail")
	}
}
