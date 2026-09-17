package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/doublemo/nats-ui/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

func (h *NATSHandler) SendMessage(c *gin.Context) {
	if c.Param("mode") != "publish" && c.Param("mode") != "request" {
		c.Status(http.StatusNotFound)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2*1024*1024)
	var input service.MessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, 400, err)
		return
	}
	result, err := h.service.SendMessage(c.Request.Context(), connectionIDFromContext(c), input, c.Param("mode") == "request")
	if err != nil {
		writeError(c, 502, err)
		return
	}
	writeSuccess(c, result)
}

func (h *NATSHandler) SubscribeMessages(c *gin.Context) {
	sub, err := h.service.SubscribeMessages(c.Request.Context(), connectionIDFromContext(c), c.Query("subject"), c.Query("queue"))
	if err != nil {
		writeError(c, 400, err)
		return
	}
	defer sub.Unsubscribe()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.SSEvent("ready", gin.H{"subject": c.Query("subject")})
	c.Writer.Flush()
	for {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		msg, err := sub.NextMsgWithContext(ctx)
		cancel()
		if c.Request.Context().Err() != nil {
			return
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, nats.ErrTimeout) {
			c.SSEvent("heartbeat", gin.H{"time": time.Now().UTC()})
		} else if err != nil {
			c.SSEvent("failure", gin.H{"message": err.Error()})
			c.Writer.Flush()
			return
		} else {
			c.SSEvent("message", service.ReadMessage(msg))
		}
		c.Writer.Flush()
	}
}

func (h *NATSHandler) CreateConsumer(c *gin.Context) {
	var input service.ConsumerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, 400, err)
		return
	}
	data, err := h.service.CreateConsumer(c.Request.Context(), connectionIDFromContext(c), c.Param("name"), input)
	if err != nil {
		writeError(c, 502, err)
		return
	}
	writeSuccess(c, data)
}

func (h *NATSHandler) DeleteConsumer(c *gin.Context) {
	if err := h.service.DeleteConsumer(c.Request.Context(), connectionIDFromContext(c), c.Param("name"), c.Param("consumer")); err != nil {
		writeError(c, 502, err)
		return
	}
	writeSuccess(c, gin.H{"deleted": true})
}

func (h *NATSHandler) StreamMessage(c *gin.Context) {
	sequence, err := strconv.ParseUint(c.Param("sequence"), 10, 64)
	if err != nil {
		writeError(c, 400, err)
		return
	}
	data, err := h.service.StreamMessage(c.Request.Context(), connectionIDFromContext(c), c.Param("name"), sequence)
	if err != nil {
		writeError(c, 502, err)
		return
	}
	writeSuccess(c, data)
}

func (h *NATSHandler) JetStreamAccount(c *gin.Context) {
	data, err := h.service.JetStreamAccount(c.Request.Context(), connectionIDFromContext(c))
	if err != nil {
		writeError(c, 502, err)
		return
	}
	writeSuccess(c, data)
}
