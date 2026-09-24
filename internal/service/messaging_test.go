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

func TestStoredMessagePreview(t *testing.T) {
	cases := []struct {
		name, body, contentType, wantType, wantPreview string
	}{
		{"json", `{"a":1}`, "application/json", "json", "{\n  \"a\": 1\n}"},
		{"text", "hello 世界", "", "text", "hello 世界"},
		{"xml", "<?xml version=\"1.0\"?><root/>", "", "xml", "<?xml version=\"1.0\"?><root/>"},
		{"binary", string([]byte{0, 0xff, 0x10}), "application/octet-stream", "binary", "00 FF 10"},
		{"pdf", "%PDF-1.7", "application/pdf", "pdf", "25 50 44 46 2D 31 2E 37"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			preview := storedMessagePreview(&nats.RawStreamMsg{Sequence: 7, Subject: "demo", Data: []byte(test.body), Header: nats.Header{"Content-Type": {test.contentType}}})
			if preview.Type != test.wantType || preview.Preview != test.wantPreview || preview.Sequence != 7 {
				t.Fatalf("unexpected preview: %+v", preview)
			}
			decoded, err := base64.StdEncoding.DecodeString(preview.RawBase64)
			if err != nil || !bytes.Equal(decoded, []byte(test.body)) {
				t.Fatalf("raw bytes unavailable for manual parsing: %v", err)
			}
		})
	}
}
