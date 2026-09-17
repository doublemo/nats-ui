package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/doublemo/nats-ui/internal/config"
	"github.com/doublemo/nats-ui/internal/service"
	"github.com/gin-gonic/gin"
)

func TestMessageHTTPStream(t *testing.T) {
	url := os.Getenv("NATS_TEST_URL")
	if url == "" {
		t.Skip("set NATS_TEST_URL to a disposable JetStream server")
	}
	dir := t.TempDir()
	manager, err := service.NewConnectionManager(config.Config{NATSURL: url, ConnectionStore: filepath.Join(dir, "connections.json"), SecretKeyFile: filepath.Join(dir, "secret.key")})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	router := gin.New()
	NewNATSHandler(service.NewNATSService(manager), manager).Register(router)
	server := httptest.NewServer(router)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	subject := fmt.Sprintf("test.http.%d", time.Now().UnixNano())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/messages/subscribe?connectionId=default&subject="+subject, nil)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || !strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("unexpected stream response: %v", response.Status)
	}
	scanner := bufio.NewScanner(response.Body)
	foundReady := false
	for scanner.Scan() {
		if scanner.Text() == "event:ready" {
			foundReady = true
			break
		}
	}
	if !foundReady {
		t.Fatalf("missing ready event: %v", scanner.Err())
	}
	body, _ := json.Marshal(service.MessageInput{Subject: subject, Payload: "http-stream"})
	publish, _ := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/v1/messages/publish?connectionId=default", strings.NewReader(string(body)))
	publish.Header.Set("Content-Type", "application/json")
	result, err := http.DefaultClient.Do(publish)
	if err != nil {
		t.Fatal(err)
	}
	result.Body.Close()
	if result.StatusCode != 200 {
		t.Fatalf("publish failed: %s", result.Status)
	}
	foundMessage := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), `"payload":"http-stream"`) {
			foundMessage = true
			break
		}
	}
	if !foundMessage {
		t.Fatalf("missing message event: %v", scanner.Err())
	}
	// Closing a browser stream must cancel its handler, allowing server shutdown.
	response.Body.Close()
	cancel()
	closed := make(chan struct{})
	go func() { server.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("subscription handler did not stop after disconnect")
	}
}
