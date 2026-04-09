package handlers

import (
	"net/http"

	"github.com/doublemo/nats-ui/internal/models"
	"github.com/doublemo/nats-ui/internal/service"
	"github.com/gin-gonic/gin"
)

type NATSHandler struct {
	service *service.NATSService
	manager *service.ConnectionManager
}

func NewNATSHandler(service *service.NATSService, manager *service.ConnectionManager) *NATSHandler {
	return &NATSHandler{service: service, manager: manager}
}

func (h *NATSHandler) Register(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.GET("/connections", h.ListConnections)
		api.POST("/connections", h.CreateConnection)
		api.PUT("/connections/:id", h.UpdateConnection)
		api.DELETE("/connections/:id", h.DeleteConnection)
		api.POST("/connections/:id/activate", h.ActivateConnection)

		api.GET("/cluster/overview", h.GetClusterOverview)

		api.GET("/streams", h.ListStreams)
		api.POST("/streams", h.CreateStream)
		api.GET("/streams/:name", h.GetStreamDetail)
		api.DELETE("/streams/:name", h.DeleteStream)

		api.GET("/kv/buckets", h.ListBuckets)
		api.POST("/kv/buckets", h.CreateBucket)
		api.DELETE("/kv/buckets/:name", h.DeleteBucket)
		api.GET("/kv/buckets/:name/entries", h.ListKVEntries)
		api.PUT("/kv/buckets/:name/entries/:key", h.PutKVEntry)
		api.DELETE("/kv/buckets/:name/entries/:key", h.DeleteKVEntry)
	}
}

// GetClusterOverview merges /varz and /connz data from every configured monitor node.
func (h *NATSHandler) GetClusterOverview(c *gin.Context) {
	data, err := h.service.GetClusterOverview(c.Request.Context(), connectionIDFromContext(c))
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, data)
}

func (h *NATSHandler) ListStreams(c *gin.Context) {
	data, err := h.service.ListStreams(c.Request.Context(), connectionIDFromContext(c))
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, data)
}

func (h *NATSHandler) CreateStream(c *gin.Context) {
	var req models.CreateStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.CreateStream(c.Request.Context(), connectionIDFromContext(c), req); err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}

	writeSuccess(c, gin.H{"created": true})
}

func (h *NATSHandler) GetStreamDetail(c *gin.Context) {
	data, err := h.service.GetStreamDetail(c.Request.Context(), connectionIDFromContext(c), c.Param("name"))
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, data)
}

func (h *NATSHandler) DeleteStream(c *gin.Context) {
	if err := h.service.DeleteStream(c.Request.Context(), connectionIDFromContext(c), c.Param("name")); err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, gin.H{"deleted": true})
}

func (h *NATSHandler) ListBuckets(c *gin.Context) {
	data, err := h.service.ListBuckets(c.Request.Context(), connectionIDFromContext(c))
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, data)
}

func (h *NATSHandler) CreateBucket(c *gin.Context) {
	var req models.CreateBucketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.CreateBucket(c.Request.Context(), connectionIDFromContext(c), req); err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, gin.H{"created": true})
}

func (h *NATSHandler) DeleteBucket(c *gin.Context) {
	if err := h.service.DeleteBucket(c.Request.Context(), connectionIDFromContext(c), c.Param("name")); err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, gin.H{"deleted": true})
}

func (h *NATSHandler) ListKVEntries(c *gin.Context) {
	data, err := h.service.ListKVEntries(c.Request.Context(), connectionIDFromContext(c), c.Param("name"))
	if err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, data)
}

func (h *NATSHandler) PutKVEntry(c *gin.Context) {
	var req models.UpsertKVEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.service.PutKVEntry(c.Request.Context(), connectionIDFromContext(c), c.Param("name"), c.Param("key"), req.Value); err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, gin.H{"updated": true})
}

func (h *NATSHandler) DeleteKVEntry(c *gin.Context) {
	if err := h.service.DeleteKVEntry(c.Request.Context(), connectionIDFromContext(c), c.Param("name"), c.Param("key")); err != nil {
		writeError(c, http.StatusBadGateway, err)
		return
	}
	writeSuccess(c, gin.H{"deleted": true})
}

func (h *NATSHandler) ListConnections(c *gin.Context) {
	writeSuccess(c, gin.H{
		"activeId": h.manager.ActiveID(),
		"items":    h.manager.List(),
	})
}

func (h *NATSHandler) CreateConnection(c *gin.Context) {
	var req models.ConnectionUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.manager.Add(req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	writeSuccess(c, item)
}

func (h *NATSHandler) UpdateConnection(c *gin.Context) {
	var req models.ConnectionUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.manager.Update(c.Param("id"), req)
	if err != nil {
		status := http.StatusBadRequest
		if err == service.ErrConnectionNotFound {
			status = http.StatusNotFound
		}
		writeError(c, status, err)
		return
	}
	writeSuccess(c, item)
}

func (h *NATSHandler) DeleteConnection(c *gin.Context) {
	err := h.manager.Delete(c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if err == service.ErrConnectionNotFound {
			status = http.StatusNotFound
		}
		writeError(c, status, err)
		return
	}
	writeSuccess(c, gin.H{"deleted": true, "activeId": h.manager.ActiveID()})
}

func (h *NATSHandler) ActivateConnection(c *gin.Context) {
	if err := h.manager.Activate(c.Param("id")); err != nil {
		status := http.StatusBadRequest
		if err == service.ErrConnectionNotFound {
			status = http.StatusNotFound
		}
		writeError(c, status, err)
		return
	}
	writeSuccess(c, gin.H{"activeId": h.manager.ActiveID()})
}

func connectionIDFromContext(c *gin.Context) string {
	return c.Query("connectionId")
}

func writeSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    data,
	})
}

func writeError(c *gin.Context, status int, err error) {
	c.JSON(status, models.APIResponse{
		Success: false,
		Message: err.Error(),
	})
}
