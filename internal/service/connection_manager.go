package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/doublemo/nats-ui/internal/config"
	"github.com/doublemo/nats-ui/internal/models"
	"github.com/nats-io/nats.go"
)

var (
	ErrConnectionNotFound = errors.New("connection not found")
	ErrLastConnection     = errors.New("at least one connection is required")
)

type clientBundle struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

type connectionStoreFile struct {
	ActiveID string                    `json:"activeId"`
	Items    []models.ConnectionConfig `json:"items"`
}

type ConnectionManager struct {
	cfg     config.Config
	mu      sync.RWMutex
	active  string
	items   map[string]models.ConnectionConfig
	clients map[string]*clientBundle
}

func NewConnectionManager(cfg config.Config) (*ConnectionManager, error) {
	manager := &ConnectionManager{
		cfg:     cfg,
		items:   make(map[string]models.ConnectionConfig),
		clients: make(map[string]*clientBundle),
	}

	if err := manager.load(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *ConnectionManager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	storePath := filepath.Clean(m.cfg.ConnectionStore)
	data, err := os.ReadFile(storePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		defaultItem := models.ConnectionConfig{
			ID:               "default",
			Name:             "Default Cluster",
			NATSURLs:         []string{m.cfg.NATSURL},
			MonitorEndpoints: append([]string(nil), m.cfg.MonitorEndpoints...),
		}
		m.items[defaultItem.ID] = defaultItem
		m.active = defaultItem.ID
		return m.saveLocked()
	}

	var payload connectionStoreFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	for _, item := range payload.Items {
		item.NATSURLs = compactStrings(item.NATSURLs)
		item.MonitorEndpoints = compactStrings(item.MonitorEndpoints)
		if item.ID == "" || len(item.NATSURLs) == 0 {
			continue
		}
		m.items[item.ID] = item
	}

	if len(m.items) == 0 {
		return errors.New("no connection profiles found")
	}

	if _, ok := m.items[payload.ActiveID]; ok {
		m.active = payload.ActiveID
	}
	if m.active == "" {
		for id := range m.items {
			m.active = id
			break
		}
	}
	return nil
}

func (m *ConnectionManager) saveLocked() error {
	items := make([]models.ConnectionConfig, 0, len(m.items))
	for _, item := range m.items {
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b models.ConnectionConfig) int {
		return strings.Compare(a.Name+a.ID, b.Name+b.ID)
	})

	payload := connectionStoreFile{
		ActiveID: m.active,
		Items:    items,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	storePath := filepath.Clean(m.cfg.ConnectionStore)
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(storePath, data, 0o644)
}

func (m *ConnectionManager) List() []models.ConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]models.ConnectionInfo, 0, len(m.items))
	for _, item := range m.items {
		info := models.ConnectionInfo{
			ConnectionConfig: item,
			IsActive:         item.ID == m.active,
		}
		if client, ok := m.clients[item.ID]; ok && client.nc != nil {
			info.Status = client.nc.Status().String()
			info.ConnectedURL = client.nc.ConnectedUrl()
		}
		items = append(items, info)
	}
	slices.SortFunc(items, func(a, b models.ConnectionInfo) int {
		return strings.Compare(a.Name+a.ID, b.Name+b.ID)
	})
	return items
}

func (m *ConnectionManager) ActiveID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

func (m *ConnectionManager) Add(req models.ConnectionUpsertRequest) (models.ConnectionConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := models.ConnectionConfig{
		ID:               generateConnectionID(req.Name),
		Name:             strings.TrimSpace(req.Name),
		NATSURLs:         compactStrings(req.NATSURLs),
		MonitorEndpoints: compactStrings(req.MonitorEndpoints),
		Username:         strings.TrimSpace(req.Username),
		Password:         req.Password,
		Token:            strings.TrimSpace(req.Token),
	}
	if item.Name == "" || len(item.NATSURLs) == 0 {
		return models.ConnectionConfig{}, errors.New("name and natsUrls are required")
	}
	if _, exists := m.items[item.ID]; exists {
		item.ID = fmt.Sprintf("%s-%d", item.ID, time.Now().Unix())
	}

	m.items[item.ID] = item
	if m.active == "" {
		m.active = item.ID
	}
	if err := m.saveLocked(); err != nil {
		return models.ConnectionConfig{}, err
	}
	return item, nil
}

func (m *ConnectionManager) Update(id string, req models.ConnectionUpsertRequest) (models.ConnectionConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.items[id]
	if !ok {
		return models.ConnectionConfig{}, ErrConnectionNotFound
	}

	current.Name = strings.TrimSpace(req.Name)
	current.NATSURLs = compactStrings(req.NATSURLs)
	current.MonitorEndpoints = compactStrings(req.MonitorEndpoints)
	current.Username = strings.TrimSpace(req.Username)
	current.Password = req.Password
	current.Token = strings.TrimSpace(req.Token)
	if current.Name == "" || len(current.NATSURLs) == 0 {
		return models.ConnectionConfig{}, errors.New("name and natsUrls are required")
	}

	if client, ok := m.clients[id]; ok {
		client.nc.Close()
		delete(m.clients, id)
	}

	m.items[id] = current
	if err := m.saveLocked(); err != nil {
		return models.ConnectionConfig{}, err
	}
	return current, nil
}

func (m *ConnectionManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.items[id]; !ok {
		return ErrConnectionNotFound
	}
	if len(m.items) == 1 {
		return ErrLastConnection
	}
	if client, ok := m.clients[id]; ok {
		client.nc.Close()
		delete(m.clients, id)
	}
	delete(m.items, id)

	if m.active == id {
		for nextID := range m.items {
			m.active = nextID
			break
		}
	}
	return m.saveLocked()
}

func (m *ConnectionManager) Activate(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.items[id]; !ok {
		return ErrConnectionNotFound
	}
	m.active = id
	return m.saveLocked()
}

func (m *ConnectionManager) Resolve(id string) (models.ConnectionConfig, *clientBundle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id == "" {
		id = m.active
	}
	item, ok := m.items[id]
	if !ok {
		return models.ConnectionConfig{}, nil, ErrConnectionNotFound
	}
	if client, ok := m.clients[id]; ok && client.nc != nil && !client.nc.IsClosed() {
		return item, client, nil
	}

	nc, err := connectNATS(item)
	if err != nil {
		return item, nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return item, nil, err
	}

	client := &clientBundle{nc: nc, js: js}
	m.clients[id] = client
	return item, client, nil
}

func (m *ConnectionManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, client := range m.clients {
		client.nc.Close()
		delete(m.clients, id)
	}
}

func connectNATS(item models.ConnectionConfig) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name("nats-ui-backend"),
		nats.Timeout(5 * time.Second),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, disconnectErr error) {
			fmt.Printf("nats disconnected(%s): %v\n", item.Name, disconnectErr)
		}),
	}
	if item.Username != "" {
		opts = append(opts, nats.UserInfo(item.Username, item.Password))
	}
	if item.Token != "" {
		opts = append(opts, nats.Token(item.Token))
	}
	return nats.Connect(strings.Join(item.NATSURLs, ","), opts...)
}

func compactStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, strings.TrimRight(item, "/"))
	}
	return result
}

func generateConnectionID(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Sprintf("conn-%d", time.Now().Unix())
	}
	name = strings.ReplaceAll(name, " ", "-")
	var b strings.Builder
	for _, ch := range name {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			b.WriteRune(ch)
		}
	}
	id := strings.Trim(b.String(), "-_")
	if id == "" {
		return fmt.Sprintf("conn-%d", time.Now().Unix())
	}
	return id
}

func (m *ConnectionManager) TestConnection(ctx context.Context, id string) error {
	_, _, err := m.Resolve(id)
	return err
}
