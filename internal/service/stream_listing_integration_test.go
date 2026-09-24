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
)

func TestStreamListExcludesKeyValueBuckets(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	suffix := time.Now().UnixNano()
	stream := fmt.Sprintf("TEST_STREAM_%d", suffix)
	bucket := fmt.Sprintf("TEST_KV_%d", suffix)
	if err := service.CreateStream(ctx, "default", models.CreateStreamRequest{Name: stream, Subjects: []string{fmt.Sprintf("test.list.%d", suffix)}, Storage: "memory", Replicas: 1}); err != nil {
		t.Fatal(err)
	}
	defer service.DeleteStream(context.Background(), "default", stream)
	if err := service.CreateBucket(ctx, "default", models.CreateBucketRequest{Name: bucket, Storage: "memory"}); err != nil {
		t.Fatal(err)
	}
	defer service.DeleteBucket(context.Background(), "default", bucket)

	streams, err := service.ListStreams(ctx, "default", "", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if streams.Total != 1 || len(streams.Items) != 1 || streams.Items[0].Name != stream {
		t.Fatalf("stream pagination includes KV bucket: %+v", streams)
	}
	hidden, err := service.ListStreams(ctx, "default", bucket, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if hidden.Total != 0 {
		t.Fatalf("KV bucket found by stream search: %+v", hidden)
	}
	buckets, err := service.ListBuckets(ctx, "default", bucket, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if buckets.Total != 1 || len(buckets.Items) != 1 || buckets.Items[0].Name != bucket {
		t.Fatalf("KV manager lost bucket: %+v", buckets)
	}
}
