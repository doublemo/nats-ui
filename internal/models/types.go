package models

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ClusterOverview struct {
	ClusterName string           `json:"clusterName"`
	ServerID    string           `json:"serverId"`
	Version     string           `json:"version"`
	NodeCount   int              `json:"nodeCount"`
	Nodes       []ClusterNode    `json:"nodes"`
	Summary     ClusterSummary   `json:"summary"`
	Traffic     ClusterTraffic   `json:"traffic"`
	Connections ConnectionDetail `json:"connections"`
}

type ClusterNode struct {
	Name          string  `json:"name"`
	Host          string  `json:"host"`
	Version       string  `json:"version"`
	Cluster       string  `json:"cluster"`
	CPU           float64 `json:"cpu"`
	Mem           int64   `json:"mem"`
	Connections   int     `json:"connections"`
	InMsgs        int64   `json:"inMsgs"`
	OutMsgs       int64   `json:"outMsgs"`
	InBytes       int64   `json:"inBytes"`
	OutBytes      int64   `json:"outBytes"`
	SlowConsumers int64   `json:"slowConsumers"`
	Status        string  `json:"status"`
}

type ClusterSummary struct {
	HealthyNodes   int   `json:"healthyNodes"`
	UnhealthyNodes int   `json:"unhealthyNodes"`
	TotalMem       int64 `json:"totalMem"`
	TotalConn      int   `json:"totalConn"`
	TotalSubs      int64 `json:"totalSubs"`
}

type ClusterTraffic struct {
	TotalInMsgs   int64 `json:"totalInMsgs"`
	TotalOutMsgs  int64 `json:"totalOutMsgs"`
	TotalInBytes  int64 `json:"totalInBytes"`
	TotalOutBytes int64 `json:"totalOutBytes"`
}

type ConnectionDetail struct {
	Active    int          `json:"active"`
	Total     int          `json:"total"`
	SlowCount int64        `json:"slowCount"`
	Items     []ConnRecord `json:"items"`
}

type ConnRecord struct {
	CID      uint64 `json:"cid"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Subs     int64  `json:"subs"`
	InMsgs   int64  `json:"inMsgs"`
	OutMsgs  int64  `json:"outMsgs"`
	InBytes  int64  `json:"inBytes"`
	OutBytes int64  `json:"outBytes"`
	Pending  int64  `json:"pending"`
}

type StreamItem struct {
	Name          string         `json:"name"`
	Subjects      []string       `json:"subjects"`
	Storage       string         `json:"storage"`
	Replicas      int            `json:"replicas"`
	Retention     string         `json:"retention"`
	Messages      uint64         `json:"messages"`
	Bytes         uint64         `json:"bytes"`
	Consumers     int            `json:"consumers"`
	Sealed        bool           `json:"sealed"`
	SubjectsState []SubjectState `json:"subjectsState,omitempty"`
}

type SubjectState struct {
	Subject string `json:"subject"`
	Count   uint64 `json:"count"`
}

type ConsumerItem struct {
	Name           string `json:"name"`
	Durable        string `json:"durable"`
	AckPolicy      string `json:"ackPolicy"`
	Pending        uint64 `json:"pending"`
	Waiting        int    `json:"waiting"`
	Delivered      uint64 `json:"delivered"`
	AckFloor       uint64 `json:"ackFloor"`
	NumRedelivered uint64 `json:"numRedelivered"`
}

type StreamDetail struct {
	Stream    StreamItem     `json:"stream"`
	Consumers []ConsumerItem `json:"consumers"`
}

type CreateStreamRequest struct {
	Name      string   `json:"name" binding:"required"`
	Subjects  []string `json:"subjects" binding:"required"`
	Storage   string   `json:"storage"`
	Replicas  int      `json:"replicas"`
	MaxAgeSec int64    `json:"maxAgeSec"`
}

type StreamBatchDeleteRequest struct {
	Names []string `json:"names" binding:"required"`
}

type BatchDeleteResult struct {
	Deleted int      `json:"deleted"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type StreamListResponse struct {
	Items []StreamItem `json:"items"`
	Pagination
}

type BucketItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Storage     string `json:"storage"`
	Values      uint64 `json:"values"`
	History     int64  `json:"history"`
	Bytes       uint64 `json:"bytes"`
}

type KVEntry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Revision  uint64 `json:"revision"`
	CreatedAt string `json:"createdAt"`
	Operation string `json:"operation"`
}

type BucketListResponse struct {
	Items []BucketItem `json:"items"`
	Pagination
}

type KVEntryListResponse struct {
	Items []KVEntry `json:"items"`
	Pagination
}

type CreateBucketRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	History     int64  `json:"history"`
	Storage     string `json:"storage"`
}

type UpsertKVEntryRequest struct {
	Value string `json:"value" binding:"required"`
}

type KVEntryBatchDeleteRequest struct {
	Keys []string `json:"keys" binding:"required"`
}

type ConnectionConfig struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Group            string   `json:"group,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	NATSURLs         []string `json:"natsUrls"`
	MonitorEndpoints []string `json:"monitorEndpoints"`
	Username         string   `json:"username,omitempty"`
	Password         string   `json:"password,omitempty"`
	Token            string   `json:"token,omitempty"`
	PasswordCipher   string   `json:"passwordCipher,omitempty"`
	TokenCipher      string   `json:"tokenCipher,omitempty"`
}

type ConnectionInfo struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Group            string   `json:"group,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	NATSURLs         []string `json:"natsUrls"`
	MonitorEndpoints []string `json:"monitorEndpoints"`
	Username         string   `json:"username,omitempty"`
	HasPassword      bool     `json:"hasPassword"`
	HasToken         bool     `json:"hasToken"`
	IsActive         bool     `json:"isActive"`
	Status           string   `json:"status,omitempty"`
	ConnectedURL     string   `json:"connectedUrl,omitempty"`
	LastError        string   `json:"lastError,omitempty"`
	LastCheckedAt    string   `json:"lastCheckedAt,omitempty"`
}

type ConnectionListResponse struct {
	ActiveID string           `json:"activeId"`
	Items    []ConnectionInfo `json:"items"`
	Pagination
}

type ConnectionUpsertRequest struct {
	Name             string   `json:"name" binding:"required"`
	Group            string   `json:"group"`
	Tags             []string `json:"tags"`
	NATSURLs         []string `json:"natsUrls" binding:"required"`
	MonitorEndpoints []string `json:"monitorEndpoints"`
	Username         string   `json:"username"`
	Password         string   `json:"password"`
	Token            string   `json:"token"`
}

type ConnectionTestResult struct {
	OK            bool   `json:"ok"`
	Status        string `json:"status"`
	ConnectedURL  string `json:"connectedUrl,omitempty"`
	Message       string `json:"message,omitempty"`
	LastCheckedAt string `json:"lastCheckedAt"`
}

type ConnectionDiscoverRequest struct {
	NATSURLs []string `json:"natsUrls" binding:"required"`
}

type ConnectionDiscoverResult struct {
	MonitorEndpoints []string `json:"monitorEndpoints"`
	Method           string   `json:"method"`
}

type ConnectionImportRequest struct {
	Items    []ConnectionUpsertRequest `json:"items" binding:"required"`
	Strategy string                    `json:"strategy,omitempty"`
}

type ConnectionImportResult struct {
	Created  int      `json:"created"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Strategy string   `json:"strategy,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

type ConnectionBatchDeleteRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type ConnectionBatchDeleteResult struct {
	Deleted  int      `json:"deleted"`
	Failed   int      `json:"failed"`
	ActiveID string   `json:"activeId"`
	Errors   []string `json:"errors,omitempty"`
}

type ConnectionImportPreviewItem struct {
	Name      string   `json:"name"`
	Group     string   `json:"group,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	NATSURLs  []string `json:"natsUrls"`
	Action    string   `json:"action"`
	MatchedID string   `json:"matchedId,omitempty"`
}

type ConnectionImportPreviewResult struct {
	Items     []ConnectionImportPreviewItem `json:"items"`
	NewCount  int                           `json:"newCount"`
	Conflicts int                           `json:"conflicts"`
}
