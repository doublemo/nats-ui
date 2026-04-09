package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/doublemo/nats-ui/internal/models"
	"github.com/nats-io/nats.go"
)

type NATSService struct {
	manager    *ConnectionManager
	httpClient *http.Client
}

func NewNATSService(manager *ConnectionManager) *NATSService {
	return &NATSService{
		manager: manager,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *NATSService) GetClusterOverview(ctx context.Context, connectionID string) (*models.ClusterOverview, error) {
	config, _, err := s.manager.Resolve(connectionID)
	if err != nil {
		return nil, err
	}

	type result struct {
		varz  varzResponse
		connz connzResponse
		err   error
	}

	results := make([]result, len(config.MonitorEndpoints))
	var wg sync.WaitGroup

	for idx, endpoint := range config.MonitorEndpoints {
		wg.Add(1)
		go func(i int, base string) {
			defer wg.Done()

			varz, err := s.fetchVarz(ctx, base)
			if err != nil {
				results[i].err = err
				return
			}

			connz, err := s.fetchConnz(ctx, base)
			if err != nil {
				results[i].err = err
				return
			}

			results[i] = result{varz: varz, connz: connz}
		}(idx, endpoint)
	}
	wg.Wait()

	overview := &models.ClusterOverview{
		NodeCount: len(config.MonitorEndpoints),
		Nodes:     make([]models.ClusterNode, 0, len(config.MonitorEndpoints)),
	}

	for _, item := range results {
		if item.err != nil {
			overview.Summary.UnhealthyNodes++
			continue
		}

		overview.ClusterName = item.varz.Cluster.Name
		overview.ServerID = item.varz.ID
		overview.Version = item.varz.Version
		overview.Nodes = append(overview.Nodes, models.ClusterNode{
			Name:          item.varz.ServerName,
			Host:          item.varz.Host,
			Version:       item.varz.Version,
			Cluster:       item.varz.Cluster.Name,
			CPU:           item.varz.CPU,
			Mem:           item.varz.Mem,
			Connections:   item.varz.Connections,
			InMsgs:        item.varz.InMsgs,
			OutMsgs:       item.varz.OutMsgs,
			InBytes:       item.varz.InBytes,
			OutBytes:      item.varz.OutBytes,
			SlowConsumers: item.varz.SlowConsumers,
			Status:        "healthy",
		})
		overview.Summary.HealthyNodes++
		overview.Summary.TotalMem += item.varz.Mem
		overview.Summary.TotalConn += item.varz.Connections
		overview.Summary.TotalSubs += item.varz.Subscriptions
		overview.Traffic.TotalInMsgs += item.varz.InMsgs
		overview.Traffic.TotalOutMsgs += item.varz.OutMsgs
		overview.Traffic.TotalInBytes += item.varz.InBytes
		overview.Traffic.TotalOutBytes += item.varz.OutBytes
		overview.Connections.Active += item.connz.NumConnections
		overview.Connections.Total += item.connz.Total
		overview.Connections.SlowCount += item.varz.SlowConsumers

		for _, conn := range item.connz.Connections {
			overview.Connections.Items = append(overview.Connections.Items, models.ConnRecord{
				CID:      conn.CID,
				Name:     conn.Name,
				IP:       conn.IP,
				Port:     conn.Port,
				Subs:     conn.Subs,
				InMsgs:   conn.InMsgs,
				OutMsgs:  conn.OutMsgs,
				InBytes:  conn.InBytes,
				OutBytes: conn.OutBytes,
				Pending:  conn.Pending,
			})
		}
	}

	overview.Summary.UnhealthyNodes = overview.NodeCount - overview.Summary.HealthyNodes
	return overview, nil
}

func (s *NATSService) ListStreams(ctx context.Context, connectionID string, page, pageSize int) (*models.StreamListResponse, error) {
	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return nil, err
	}

	names := client.js.StreamNames()
	items := make([]models.StreamItem, 0)

	for name := range names {
		info, err := client.js.StreamInfo(name)
		if err != nil {
			return nil, err
		}

		item := models.StreamItem{
			Name:          info.Config.Name,
			Subjects:      info.Config.Subjects,
			Storage:       strings.ToLower(info.Config.Storage.String()),
			Replicas:      info.Config.Replicas,
			Retention:     strings.ToLower(info.Config.Retention.String()),
			Messages:      info.State.Msgs,
			Bytes:         info.State.Bytes,
			Consumers:     int(info.State.Consumers),
			Sealed:        info.Config.Sealed,
			SubjectsState: make([]models.SubjectState, 0, len(info.State.Subjects)),
		}

		for subject, count := range info.State.Subjects {
			item.SubjectsState = append(item.SubjectsState, models.SubjectState{
				Subject: subject,
				Count:   count,
			})
		}

		items = append(items, item)
	}

	total := len(items)
	page, pageSize = normalizePagination(page, pageSize)
	start, end := paginateBounds(total, page, pageSize)

	return &models.StreamListResponse{
		Items: items[start:end],
		Pagination: models.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

func (s *NATSService) CreateStream(ctx context.Context, connectionID string, req models.CreateStreamRequest) error {
	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return err
	}

	storage := toStorage(req.Storage)
	cfg := newStreamConfig(req, storage)
	_, err = client.js.AddStream(cfg)
	return err
}

func (s *NATSService) DeleteStream(ctx context.Context, connectionID, name string) error {
	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return err
	}
	return client.js.DeleteStream(name)
}

func (s *NATSService) BatchDeleteStreams(ctx context.Context, connectionID string, names []string) models.BatchDeleteResult {
	result := models.BatchDeleteResult{Errors: make([]string, 0)}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err := s.DeleteStream(ctx, connectionID, name); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		result.Deleted++
	}
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result
}

func (s *NATSService) GetStreamDetail(ctx context.Context, connectionID, name string) (*models.StreamDetail, error) {
	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return nil, err
	}

	info, err := client.js.StreamInfo(name)
	if err != nil {
		return nil, err
	}

	detail := &models.StreamDetail{
		Stream: models.StreamItem{
			Name:          info.Config.Name,
			Subjects:      info.Config.Subjects,
			Storage:       strings.ToLower(info.Config.Storage.String()),
			Replicas:      info.Config.Replicas,
			Retention:     strings.ToLower(info.Config.Retention.String()),
			Messages:      info.State.Msgs,
			Bytes:         info.State.Bytes,
			Consumers:     int(info.State.Consumers),
			Sealed:        info.Config.Sealed,
			SubjectsState: make([]models.SubjectState, 0, len(info.State.Subjects)),
		},
		Consumers: make([]models.ConsumerItem, 0),
	}

	for subject, count := range info.State.Subjects {
		detail.Stream.SubjectsState = append(detail.Stream.SubjectsState, models.SubjectState{
			Subject: subject,
			Count:   count,
		})
	}

	for consumer := range client.js.ConsumersInfo(name) {
		detail.Consumers = append(detail.Consumers, models.ConsumerItem{
			Name:           consumer.Name,
			Durable:        consumer.Config.Durable,
			AckPolicy:      consumer.Config.AckPolicy.String(),
			Pending:        consumer.NumPending,
			Waiting:        consumer.NumWaiting,
			Delivered:      consumer.Delivered.Consumer,
			AckFloor:       consumer.AckFloor.Consumer,
			NumRedelivered: uint64(consumer.NumRedelivered),
		})
	}

	return detail, nil
}

func (s *NATSService) ListBuckets(ctx context.Context, connectionID string, page, pageSize int) (*models.BucketListResponse, error) {
	streams, err := s.ListStreams(ctx, connectionID, 1, 100000)
	if err != nil {
		return nil, err
	}

	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return nil, err
	}

	items := make([]models.BucketItem, 0)
	for _, stream := range streams.Items {
		if !strings.HasPrefix(stream.Name, "KV_") {
			continue
		}

		bucketName := strings.TrimPrefix(stream.Name, "KV_")
		kv, err := client.js.KeyValue(bucketName)
		if err != nil {
			return nil, err
		}

		status, err := kv.Status()
		if err != nil {
			return nil, err
		}

		items = append(items, models.BucketItem{
			Name:    bucketName,
			Storage: strings.ToLower(status.BackingStore()),
			Values:  status.Values(),
			History: int64(status.History()),
			Bytes:   uint64(status.Bytes()),
		})
	}

	total := len(items)
	page, pageSize = normalizePagination(page, pageSize)
	start, end := paginateBounds(total, page, pageSize)

	return &models.BucketListResponse{
		Items: items[start:end],
		Pagination: models.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

func (s *NATSService) CreateBucket(ctx context.Context, connectionID string, req models.CreateBucketRequest) error {
	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return err
	}

	history := req.History
	if history <= 0 {
		history = 1
	}

	kvConfig := modelsToKVConfig(req, history)
	_, err = client.js.CreateKeyValue(&kvConfig)
	return err
}

func (s *NATSService) DeleteBucket(ctx context.Context, connectionID, name string) error {
	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return err
	}
	return client.js.DeleteKeyValue(name)
}

func (s *NATSService) BatchDeleteBuckets(ctx context.Context, connectionID string, names []string) models.BatchDeleteResult {
	result := models.BatchDeleteResult{Errors: make([]string, 0)}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err := s.DeleteBucket(ctx, connectionID, name); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		result.Deleted++
	}
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result
}

func (s *NATSService) ListKVEntries(ctx context.Context, connectionID, bucket string, page, pageSize int) (*models.KVEntryListResponse, error) {
	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return nil, err
	}
	kv, err := client.js.KeyValue(bucket)
	if err != nil {
		return nil, err
	}

	keys, err := kv.Keys()
	if err != nil && !strings.Contains(err.Error(), "no keys found") {
		return nil, err
	}

	entries := make([]models.KVEntry, 0, len(keys))
	for _, key := range keys {
		entry, err := kv.Get(key)
		if err != nil {
			return nil, err
		}

		entries = append(entries, models.KVEntry{
			Key:       entry.Key(),
			Value:     string(entry.Value()),
			Revision:  entry.Revision(),
			CreatedAt: entry.Created().Format(time.RFC3339),
			Operation: entry.Operation().String(),
		})
	}

	total := len(entries)
	page, pageSize = normalizePagination(page, pageSize)
	start, end := paginateBounds(total, page, pageSize)

	return &models.KVEntryListResponse{
		Items: entries[start:end],
		Pagination: models.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

func (s *NATSService) PutKVEntry(ctx context.Context, connectionID, bucket, key, value string) error {
	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return err
	}
	kv, err := client.js.KeyValue(bucket)
	if err != nil {
		return err
	}
	_, err = kv.Put(key, []byte(value))
	return err
}

func (s *NATSService) DeleteKVEntry(ctx context.Context, connectionID, bucket, key string) error {
	_, client, err := s.manager.Resolve(connectionID)
	if err != nil {
		return err
	}
	kv, err := client.js.KeyValue(bucket)
	if err != nil {
		return err
	}
	return kv.Delete(key)
}

func (s *NATSService) BatchDeleteKVEntries(ctx context.Context, connectionID, bucket string, keys []string) models.BatchDeleteResult {
	result := models.BatchDeleteResult{Errors: make([]string, 0)}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if err := s.DeleteKVEntry(ctx, connectionID, bucket, key); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		result.Deleted++
	}
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result
}

func toStorage(storage string) nats.StorageType {
	if strings.EqualFold(storage, "memory") {
		return nats.MemoryStorage
	}
	return nats.FileStorage
}

func newStreamConfig(req models.CreateStreamRequest, storage nats.StorageType) *nats.StreamConfig {
	cfg := &nats.StreamConfig{
		Name:      req.Name,
		Subjects:  req.Subjects,
		Replicas:  req.Replicas,
		Storage:   storage,
		Retention: nats.LimitsPolicy,
	}
	if req.MaxAgeSec > 0 {
		cfg.MaxAge = time.Duration(req.MaxAgeSec) * time.Second
	}
	return cfg
}

func modelsToKVConfig(req models.CreateBucketRequest, history int64) nats.KeyValueConfig {
	return nats.KeyValueConfig{
		Bucket:      req.Name,
		Description: req.Description,
		History:     uint8(history),
		Storage:     toStorage(req.Storage),
	}
}

func normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func paginateBounds(total, page, pageSize int) (int, int) {
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

type varzResponse struct {
	ID            string  `json:"server_id"`
	ServerName    string  `json:"server_name"`
	Version       string  `json:"version"`
	Host          string  `json:"host"`
	CPU           float64 `json:"cpu"`
	Mem           int64   `json:"mem"`
	Connections   int     `json:"connections"`
	Subscriptions int64   `json:"subscriptions"`
	InMsgs        int64   `json:"in_msgs"`
	OutMsgs       int64   `json:"out_msgs"`
	InBytes       int64   `json:"in_bytes"`
	OutBytes      int64   `json:"out_bytes"`
	SlowConsumers int64   `json:"slow_consumers"`
	Cluster       struct {
		Name string `json:"name"`
	} `json:"cluster"`
}

type connzResponse struct {
	Total          int          `json:"total"`
	NumConnections int          `json:"num_connections"`
	Connections    []connRecord `json:"connections"`
}

type connRecord struct {
	CID      uint64 `json:"cid"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Subs     int64  `json:"subscriptions"`
	InMsgs   int64  `json:"in_msgs"`
	OutMsgs  int64  `json:"out_msgs"`
	InBytes  int64  `json:"in_bytes"`
	OutBytes int64  `json:"out_bytes"`
	Pending  int64  `json:"pending_bytes"`
}

func (s *NATSService) fetchVarz(ctx context.Context, endpoint string) (varzResponse, error) {
	var payload varzResponse
	if err := s.fetchJSON(ctx, endpoint+"/varz", &payload); err != nil {
		return varzResponse{}, err
	}
	return payload, nil
}

func (s *NATSService) fetchConnz(ctx context.Context, endpoint string) (connzResponse, error) {
	u, err := url.Parse(endpoint + "/connz")
	if err != nil {
		return connzResponse{}, err
	}
	query := u.Query()
	query.Set("limit", "256")
	query.Set("offset", "0")
	query.Set("subs", "1")
	u.RawQuery = query.Encode()

	var payload connzResponse
	if err := s.fetchJSON(ctx, u.String(), &payload); err != nil {
		return connzResponse{}, err
	}
	return payload, nil
}

func (s *NATSService) fetchJSON(ctx context.Context, endpoint string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request %s failed: status=%d body=%s", endpoint, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
