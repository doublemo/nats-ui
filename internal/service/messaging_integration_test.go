package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doublemo/nats-ui/internal/config"
	"github.com/doublemo/nats-ui/internal/models"
	"github.com/nats-io/nats.go"
)

// Run against a disposable JetStream server with NATS_TEST_URL set.
func TestMessagingIntegration(t *testing.T) {
	url := os.Getenv("NATS_TEST_URL")
	if url == "" {
		t.Skip("set NATS_TEST_URL to a disposable JetStream server")
	}
	dir := t.TempDir()
	manager, err := NewConnectionManager(config.Config{NATSURL: url, ConnectionStore: filepath.Join(dir, "connections.json"), SecretKeyFile: filepath.Join(dir, "secret.key")})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	service := NewNATSService(manager)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	subject := fmt.Sprintf("test.console.%d", time.Now().UnixNano())
	sub, err := service.SubscribeMessages(ctx, "default", subject, "")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()
	_, err = service.SendMessage(ctx, "default", MessageInput{Subject: subject, Payload: "hello", Headers: map[string]string{"X-Test": "yes"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := sub.NextMsgWithContext(ctx)
	if err != nil || string(msg.Data) != "hello" || msg.Header.Get("X-Test") != "yes" {
		t.Fatalf("publish/subscribe: msg=%v err=%v", msg, err)
	}

	// A manually published reply must satisfy RequestMsgWithContext's inbox.
	replyErr := make(chan error, 1)
	go func() {
		request, err := sub.NextMsgWithContext(ctx)
		if err == nil {
			_, err = service.SendMessage(ctx, "default", MessageInput{Subject: request.Reply, Payload: "response"}, false)
		}
		replyErr <- err
	}()
	result, err := service.SendMessage(ctx, "default", MessageInput{Subject: subject, Payload: "request", TimeoutMS: 2000}, true)
	if err != nil {
		t.Fatal(err)
	}
	if err = <-replyErr; err != nil {
		t.Fatal(err)
	}
	if result.(map[string]interface{})["message"].(MessageRecord).Payload != "response" {
		t.Fatal("request/reply mismatch")
	}
	if _, err = service.SendMessage(ctx, "default", MessageInput{Subject: subject + ".absent", TimeoutMS: 100}, true); err == nil {
		t.Fatal("request without responders must fail")
	}

	stream := fmt.Sprintf("CONSOLE_TEST_%d", time.Now().UnixNano())
	if err = service.CreateStream(ctx, "default", models.CreateStreamRequest{Name: stream, Subjects: []string{subject + ".stored"}, Storage: "memory", Replicas: 1}); err != nil {
		t.Fatal(err)
	}
	defer service.DeleteStream(context.Background(), "default", stream)
	_, client, err := manager.Resolve("default")
	if err != nil {
		t.Fatal(err)
	}
	ack, err := client.js.Publish(subject+".stored", []byte("persisted"), nats.Context(ctx))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateConsumer(ctx, "default", stream, ConsumerInput{Name: "inspector", DeliverPolicy: "all", AckWaitSec: 30, MaxDeliver: 5}); err != nil {
		t.Fatal(err)
	}
	stored, err := service.StreamMessage(ctx, "default", stream, ack.Sequence)
	if err != nil {
		t.Fatal(err)
	}
	if stored.(map[string]interface{})["message"].(MessageRecord).Payload != "persisted" {
		t.Fatal("stored payload mismatch")
	}
	for i := 0; i < 12; i++ {
		if _, err := client.js.Publish(subject+".stored", []byte(fmt.Sprintf(`{"index":%d}`, i)), nats.Context(ctx)); err != nil {
			t.Fatal(err)
		}
	}
	if err := client.js.DeleteMsg(stream, ack.Sequence+1, nats.Context(ctx)); err != nil {
		t.Fatal(err)
	}
	recent, err := service.RecentStreamMessages(ctx, "default", stream)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 10 || recent[0].Sequence != ack.Sequence+12 || recent[9].Sequence != ack.Sequence+3 || recent[0].Type != "json" {
		t.Fatalf("unexpected recent messages: %+v", recent)
	}
	consumer, err := client.js.ConsumerInfo(stream, "inspector", nats.Context(ctx))
	if err != nil || consumer.NumPending != 12 || consumer.Delivered.Consumer != 0 {
		t.Fatalf("inspector must not consume messages: %v %v", consumer, err)
	}
	if _, err = service.JetStreamAccount(ctx, "default"); err != nil {
		t.Fatal(err)
	}
	if err = service.DeleteConsumer(ctx, "default", stream, "inspector"); err != nil {
		t.Fatal(err)
	}
	if err = sub.Unsubscribe(); err != nil {
		t.Fatal(err)
	}
	if sub.IsValid() {
		t.Fatal("subscription must be released")
	}
}
