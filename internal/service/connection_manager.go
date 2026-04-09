package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
	nc            *nats.Conn
	js            nats.JetStreamContext
	lastStatus    string
	connectedURL  string
	lastError     string
	lastCheckedAt time.Time
}

type connectionStoreFile struct {
	ActiveID string                    `json:"activeId"`
	Items    []models.ConnectionConfig `json:"items"`
}

type ConnectionManager struct {
	cfg     config.Config
	secrets *SecretStore
	mu      sync.RWMutex
	active  string
	items   map[string]models.ConnectionConfig
	clients map[string]*clientBundle
}

func NewConnectionManager(cfg config.Config) (*ConnectionManager, error) {
	secrets, err := NewSecretStore(cfg)
	if err != nil {
		return nil, err
	}
	manager := &ConnectionManager{
		cfg:     cfg,
		secrets: secrets,
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
		item.Tags = compactStrings(item.Tags)
		item.Group = strings.TrimSpace(item.Group)
		if item.PasswordCipher == "" && item.Password != "" {
			item.PasswordCipher, err = m.secrets.Encrypt(item.Password)
			if err != nil {
				return err
			}
		}
		if item.TokenCipher == "" && item.Token != "" {
			item.TokenCipher, err = m.secrets.Encrypt(item.Token)
			if err != nil {
				return err
			}
		}
		if item.PasswordCipher != "" {
			item.Password, err = m.secrets.Decrypt(item.PasswordCipher)
			if err != nil {
				return err
			}
		}
		if item.TokenCipher != "" {
			item.Token, err = m.secrets.Decrypt(item.TokenCipher)
			if err != nil {
				return err
			}
		}
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
		saved := item
		saved.Password = ""
		saved.Token = ""
		var err error
		saved.PasswordCipher, err = m.secrets.Encrypt(item.Password)
		if err != nil {
			return err
		}
		saved.TokenCipher, err = m.secrets.Encrypt(item.Token)
		if err != nil {
			return err
		}
		items = append(items, saved)
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
			ID:               item.ID,
			Name:             item.Name,
			Group:            item.Group,
			Tags:             append([]string(nil), item.Tags...),
			NATSURLs:         append([]string(nil), item.NATSURLs...),
			MonitorEndpoints: append([]string(nil), item.MonitorEndpoints...),
			Username:         item.Username,
			HasPassword:      item.Password != "",
			HasToken:         item.Token != "",
			IsActive:         item.ID == m.active,
		}
		if client, ok := m.clients[item.ID]; ok && client != nil {
			if client.nc != nil {
				info.Status = client.nc.Status().String()
				info.ConnectedURL = client.nc.ConnectedUrl()
			} else {
				info.Status = client.lastStatus
				info.ConnectedURL = client.connectedURL
			}
			info.LastError = client.lastError
			if !client.lastCheckedAt.IsZero() {
				info.LastCheckedAt = client.lastCheckedAt.Format(time.RFC3339)
			}
		}
		items = append(items, info)
	}
	slices.SortFunc(items, func(a, b models.ConnectionInfo) int {
		return strings.Compare(a.Name+a.ID, b.Name+b.ID)
	})
	return items
}

func (m *ConnectionManager) ListPaged(page, pageSize int, keyword, group, tag, status string) models.ConnectionListResponse {
	items := m.List()
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	group = strings.TrimSpace(group)
	tag = strings.TrimSpace(tag)
	status = strings.TrimSpace(status)

	filtered := make([]models.ConnectionInfo, 0, len(items))
	for _, item := range items {
		displayStatus := item.Status
		if displayStatus == "" {
			displayStatus = "未检测"
		}
		if item.IsActive {
			displayStatus = "当前连接"
		}
		if group != "" && (item.Group != group && !(group == "未分组" && item.Group == "")) {
			continue
		}
		if tag != "" && !slices.Contains(item.Tags, tag) {
			continue
		}
		if status != "" && displayStatus != status {
			continue
		}
		if keyword != "" {
			searchText := strings.ToLower(strings.Join(append([]string{
				item.Name,
				item.Group,
				item.Username,
				item.ConnectedURL,
				item.Status,
				item.LastError,
			}, append(append(item.Tags, item.NATSURLs...), item.MonitorEndpoints...)...), " "))
			if !strings.Contains(searchText, keyword) {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	page, pageSize = normalizeConnectionPagination(page, pageSize)
	total := len(filtered)
	start, end := connectionPaginateBounds(total, page, pageSize)

	return models.ConnectionListResponse{
		ActiveID: m.ActiveID(),
		Items:    filtered[start:end],
		Pagination: models.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}
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
		Group:            strings.TrimSpace(req.Group),
		Tags:             compactStrings(req.Tags),
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

func (m *ConnectionManager) findByNameLocked(name string) (models.ConnectionConfig, bool) {
	name = strings.TrimSpace(name)
	for _, item := range m.items {
		if strings.EqualFold(strings.TrimSpace(item.Name), name) {
			return item, true
		}
	}
	return models.ConnectionConfig{}, false
}

func (m *ConnectionManager) Update(id string, req models.ConnectionUpsertRequest) (models.ConnectionConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.items[id]
	if !ok {
		return models.ConnectionConfig{}, ErrConnectionNotFound
	}

	current.Name = strings.TrimSpace(req.Name)
	current.Group = strings.TrimSpace(req.Group)
	current.Tags = compactStrings(req.Tags)
	current.NATSURLs = compactStrings(req.NATSURLs)
	current.MonitorEndpoints = compactStrings(req.MonitorEndpoints)
	current.Username = strings.TrimSpace(req.Username)
	if req.Password != "" {
		current.Password = req.Password
	}
	if req.Token != "" {
		current.Token = strings.TrimSpace(req.Token)
	}
	if current.Name == "" || len(current.NATSURLs) == 0 {
		return models.ConnectionConfig{}, errors.New("name and natsUrls are required")
	}

	if client, ok := m.clients[id]; ok {
		if client.nc != nil {
			client.nc.Close()
		}
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
		if client.nc != nil {
			client.nc.Close()
		}
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

func (m *ConnectionManager) Import(req models.ConnectionImportRequest) models.ConnectionImportResult {
	result := models.ConnectionImportResult{
		Strategy: strings.ToLower(strings.TrimSpace(req.Strategy)),
		Errors:   make([]string, 0),
	}
	if result.Strategy == "" {
		result.Strategy = "skip"
	}
	for _, item := range req.Items {
		preview := m.PreviewImport(models.ConnectionImportRequest{Items: []models.ConnectionUpsertRequest{item}})
		if len(preview.Items) == 0 {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid item", item.Name))
			continue
		}
		conflict := preview.Items[0]
		if conflict.Action == "conflict" {
			if result.Strategy == "skip" {
				result.Skipped++
				continue
			}
			if result.Strategy == "overwrite" {
				if _, err := m.Update(conflict.MatchedID, item); err != nil {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Name, err))
					continue
				}
				result.Updated++
				continue
			}
		}
		if _, err := m.Add(item); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Name, err))
			continue
		}
		result.Created++
	}
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result
}

func (m *ConnectionManager) PreviewImport(req models.ConnectionImportRequest) models.ConnectionImportPreviewResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := models.ConnectionImportPreviewResult{
		Items: make([]models.ConnectionImportPreviewItem, 0, len(req.Items)),
	}
	for _, item := range req.Items {
		previewItem := models.ConnectionImportPreviewItem{
			Name:     strings.TrimSpace(item.Name),
			Group:    strings.TrimSpace(item.Group),
			Tags:     compactStrings(item.Tags),
			NATSURLs: compactStrings(item.NATSURLs),
			Action:   "create",
		}
		if matched, ok := m.findByNameLocked(previewItem.Name); ok {
			previewItem.Action = "conflict"
			previewItem.MatchedID = matched.ID
			result.Conflicts++
		} else {
			result.NewCount++
		}
		result.Items = append(result.Items, previewItem)
	}
	return result
}

func (m *ConnectionManager) BatchDelete(ids []string) models.ConnectionBatchDeleteResult {
	result := models.ConnectionBatchDeleteResult{
		Errors: make([]string, 0),
	}
	for _, id := range compactStrings(ids) {
		if err := m.Delete(id); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", id, err))
			continue
		}
		result.Deleted++
	}
	result.ActiveID = m.ActiveID()
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result
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
	client.lastStatus = nc.Status().String()
	client.connectedURL = nc.ConnectedUrl()
	client.lastCheckedAt = time.Now()
	m.clients[id] = client
	return item, client, nil
}

func (m *ConnectionManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, client := range m.clients {
		if client.nc != nil {
			client.nc.Close()
		}
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
	m.mu.RLock()
	item, ok := m.items[id]
	m.mu.RUnlock()
	if !ok {
		return ErrConnectionNotFound
	}

	nc, err := connectNATS(item)
	checkedAt := time.Now()
	if err != nil {
		m.setTestState(id, "", "", err, checkedAt)
		return err
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		m.setTestState(id, "", "", err, checkedAt)
		return err
	}
	_ = js
	connectedURL := nc.ConnectedUrl()
	status := nc.Status().String()
	m.setTestState(id, status, connectedURL, nil, checkedAt)
	return nil
}

func (m *ConnectionManager) Probe(ctx context.Context, req models.ConnectionUpsertRequest) error {
	item := models.ConnectionConfig{
		ID:               "probe",
		Name:             strings.TrimSpace(req.Name),
		Group:            strings.TrimSpace(req.Group),
		Tags:             compactStrings(req.Tags),
		NATSURLs:         compactStrings(req.NATSURLs),
		MonitorEndpoints: compactStrings(req.MonitorEndpoints),
		Username:         strings.TrimSpace(req.Username),
		Password:         req.Password,
		Token:            strings.TrimSpace(req.Token),
	}
	if item.Name == "" || len(item.NATSURLs) == 0 {
		return errors.New("name and natsUrls are required")
	}
	nc, err := connectNATS(item)
	if err != nil {
		return err
	}
	defer nc.Close()
	_, err = nc.JetStream()
	return err
}

func (m *ConnectionManager) DiscoverMonitorEndpoints(req models.ConnectionDiscoverRequest) models.ConnectionDiscoverResult {
	return models.ConnectionDiscoverResult{
		MonitorEndpoints: deriveMonitorEndpoints(req.NATSURLs),
		Method:           "host-match-plus-4000-port",
	}
}

func effectiveMonitorEndpoints(config models.ConnectionConfig, connectedURL string) []string {
	if len(config.MonitorEndpoints) > 0 {
		return compactStrings(config.MonitorEndpoints)
	}

	candidates := make([]string, 0, len(config.NATSURLs)+1)
	if strings.TrimSpace(connectedURL) != "" {
		candidates = append(candidates, connectedURL)
	}
	candidates = append(candidates, config.NATSURLs...)
	return deriveMonitorEndpoints(candidates)
}

func deriveMonitorEndpoints(natsURLs []string) []string {
	endpoints := make([]string, 0, len(natsURLs))
	seen := make(map[string]struct{})

	for _, raw := range compactStrings(natsURLs) {
		u, err := parseServerURL(raw)
		if err != nil {
			continue
		}

		host := u.Hostname()
		if host == "" {
			continue
		}

		monitorPort := "8222"
		if port := u.Port(); port != "" {
			if parsedPort, err := strconv.Atoi(port); err == nil {
				monitorPort = strconv.Itoa(parsedPort + 4000)
			}
		}

		endpoint := fmt.Sprintf("http://%s:%s", host, monitorPort)
		if _, ok := seen[endpoint]; ok {
			continue
		}
		seen[endpoint] = struct{}{}
		endpoints = append(endpoints, endpoint)
	}

	return endpoints
}

func parseServerURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("empty server url")
	}
	if !strings.Contains(raw, "://") {
		raw = "nats://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "nats", "tls", "ws", "wss":
		return parsed, nil
	default:
		return nil, fmt.Errorf("unsupported server scheme: %s", parsed.Scheme)
	}
}

func (m *ConnectionManager) setTestState(id, status, connectedURL string, err error, checkedAt time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.clients[id]
	if ok && existing != nil && existing.nc != nil && existing.nc.IsClosed() {
		delete(m.clients, id)
		existing = nil
		ok = false
	}

	if err != nil {
		if !ok {
			m.clients[id] = &clientBundle{}
			existing = m.clients[id]
		}
		existing.lastError = err.Error()
		existing.lastCheckedAt = checkedAt
		existing.lastStatus = "ERROR"
		return
	}
	if !ok {
		m.clients[id] = &clientBundle{}
		existing = m.clients[id]
	}
	existing.lastStatus = status
	existing.connectedURL = connectedURL
	existing.lastError = ""
	existing.lastCheckedAt = checkedAt
}

func normalizeConnectionPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 12
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func connectionPaginateBounds(total, page, pageSize int) (int, int) {
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
