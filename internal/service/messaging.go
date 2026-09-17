package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nats-io/nats.go"
)

type MessageInput struct {
	Subject   string            `json:"subject" binding:"required"`
	Payload   string            `json:"payload"`
	Headers   map[string]string `json:"headers"`
	TimeoutMS int               `json:"timeoutMs"`
}

type MessageRecord struct {
	Subject   string      `json:"subject"`
	Reply     string      `json:"reply,omitempty"`
	Payload   string      `json:"payload"`
	Encoding  string      `json:"encoding"`
	Headers   nats.Header `json:"headers"`
	Bytes     int         `json:"bytes"`
	Truncated bool        `json:"truncated"`
	Time      time.Time   `json:"time"`
}

func validSubject(subject string, wildcard bool) bool {
	if subject == "" || strings.ContainsFunc(subject, unicode.IsSpace) {
		return false
	}
	tokens := strings.Split(subject, ".")
	for i, token := range tokens {
		if token == "" {
			return false
		}
		if wildcard && (token == "*" || (token == ">" && i == len(tokens)-1)) {
			continue
		}
		if strings.ContainsAny(token, "*>\x00") {
			return false
		}
	}
	return true
}

func messageRecord(msg *nats.Msg) MessageRecord {
	data := msg.Data
	truncated := len(data) > 65536
	if truncated {
		data = data[:65536]
	}
	encoding, payload := "utf8", string(data)
	if !utf8.Valid(data) {
		encoding, payload = "base64", base64.StdEncoding.EncodeToString(data)
	}
	return MessageRecord{Subject: msg.Subject, Reply: msg.Reply, Payload: payload, Encoding: encoding, Headers: msg.Header, Bytes: len(msg.Data), Truncated: truncated, Time: time.Now().UTC()}
}

func (s *NATSService) SendMessage(ctx context.Context, id string, input MessageInput, request bool) (interface{}, error) {
	if !validSubject(input.Subject, false) {
		return nil, fmt.Errorf("invalid publish subject")
	}
	if len(input.Payload) > 1024*1024 {
		return nil, fmt.Errorf("payload exceeds 1 MiB")
	}
	_, client, err := s.manager.Resolve(id)
	if err != nil {
		return nil, err
	}
	msg := &nats.Msg{Subject: input.Subject, Data: []byte(input.Payload), Header: nats.Header{}}
	for key, value := range input.Headers {
		if key == "" || strings.ContainsAny(key, "\r\n:") || strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("invalid message header")
		}
		msg.Header.Set(key, value)
	}
	timeout := input.TimeoutMS
	if timeout == 0 {
		timeout = 3000
	}
	if timeout < 100 || timeout > 30000 {
		return nil, fmt.Errorf("timeout must be 100–30000 ms")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()
	start := time.Now()
	if request {
		response, err := client.nc.RequestMsgWithContext(ctx, msg)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"message": messageRecord(response), "elapsedMs": float64(time.Since(start).Microseconds()) / 1000}, nil
	}
	if err := client.nc.PublishMsg(msg); err != nil {
		return nil, err
	}
	if err := client.nc.FlushWithContext(ctx); err != nil {
		return nil, err
	}
	return map[string]interface{}{"published": true, "elapsedMs": float64(time.Since(start).Microseconds()) / 1000}, nil
}

// The subscription belongs to the HTTP stream, never to the shared connection.
func (s *NATSService) SubscribeMessages(ctx context.Context, id, subject, queue string) (*nats.Subscription, error) {
	if !validSubject(subject, true) {
		return nil, fmt.Errorf("invalid subscription subject")
	}
	if queue != "" && !validSubject(queue, false) {
		return nil, fmt.Errorf("invalid queue group")
	}
	_, client, err := s.manager.Resolve(id)
	if err != nil {
		return nil, err
	}
	var sub *nats.Subscription
	if queue == "" {
		sub, err = client.nc.SubscribeSync(subject)
	} else {
		sub, err = client.nc.QueueSubscribeSync(subject, queue)
	}
	if err != nil {
		return nil, err
	}
	if err = sub.SetPendingLimits(256, 4*1024*1024); err == nil {
		flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		err = client.nc.FlushWithContext(flushCtx)
	}
	if err != nil {
		sub.Unsubscribe()
		return nil, err
	}
	return sub, nil
}

func ReadMessage(msg *nats.Msg) MessageRecord { return messageRecord(msg) }

type ConsumerInput struct {
	Name          string `json:"name" binding:"required"`
	Filter        string `json:"filter"`
	DeliverPolicy string `json:"deliverPolicy"`
	AckWaitSec    int    `json:"ackWaitSec"`
	MaxDeliver    int    `json:"maxDeliver"`
}

func (s *NATSService) CreateConsumer(ctx context.Context, id, stream string, input ConsumerInput) (interface{}, error) {
	if strings.ContainsAny(input.Name, ".*>/\\ \t\r\n") || input.Name == "" {
		return nil, fmt.Errorf("invalid consumer name")
	}
	if input.Filter != "" && !validSubject(input.Filter, true) {
		return nil, fmt.Errorf("invalid filter subject")
	}
	if input.AckWaitSec < 1 || input.AckWaitSec > 86400 || input.MaxDeliver < 1 || input.MaxDeliver > 1000 {
		return nil, fmt.Errorf("invalid acknowledgment limits")
	}
	policy := nats.DeliverAllPolicy
	switch input.DeliverPolicy {
	case "all":
	case "new":
		policy = nats.DeliverNewPolicy
	case "last":
		policy = nats.DeliverLastPolicy
	default:
		return nil, fmt.Errorf("invalid delivery policy")
	}
	_, client, err := s.manager.Resolve(id)
	if err != nil {
		return nil, err
	}
	return client.js.AddConsumer(stream, &nats.ConsumerConfig{Durable: input.Name, FilterSubject: input.Filter, DeliverPolicy: policy, AckPolicy: nats.AckExplicitPolicy, AckWait: time.Duration(input.AckWaitSec) * time.Second, MaxDeliver: input.MaxDeliver, MaxAckPending: 1000}, nats.Context(ctx))
}

func (s *NATSService) DeleteConsumer(ctx context.Context, id, stream, name string) error {
	_, client, err := s.manager.Resolve(id)
	if err != nil {
		return err
	}
	return client.js.DeleteConsumer(stream, name, nats.Context(ctx))
}

func (s *NATSService) StreamMessage(ctx context.Context, id, stream string, sequence uint64) (interface{}, error) {
	if sequence == 0 {
		return nil, fmt.Errorf("sequence must be positive")
	}
	_, client, err := s.manager.Resolve(id)
	if err != nil {
		return nil, err
	}
	msg, err := client.js.GetMsg(stream, sequence, nats.Context(ctx))
	if err != nil {
		return nil, err
	}
	record := messageRecord(&nats.Msg{Subject: msg.Subject, Data: msg.Data, Header: msg.Header})
	record.Time = msg.Time
	return map[string]interface{}{"sequence": msg.Sequence, "message": record}, nil
}

func (s *NATSService) JetStreamAccount(ctx context.Context, id string) (interface{}, error) {
	_, client, err := s.manager.Resolve(id)
	if err != nil {
		return nil, err
	}
	return client.js.AccountInfo(nats.Context(ctx))
}
