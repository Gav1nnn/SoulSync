package app

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type HTTPServer struct {
	service *Service
}

func NewHTTPServer(service *Service) *HTTPServer {
	return &HTTPServer{service: service}
}

func (s *HTTPServer) Router() *gin.Engine {
	router := gin.Default()
	router.GET("/healthz", s.healthz)
	router.GET("/api/memories", s.memories)
	router.GET("/api/messages", s.messages)
	router.GET("/api/traces/:trace_id", s.trace)
	router.POST("/api/chat", s.chat)
	return router
}

func (s *HTTPServer) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "backend",
		"status":  "ok",
	})
}

func (s *HTTPServer) chat(c *gin.Context) {
	var request ChatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	response, err := s.service.Chat(c.Request.Context(), request.Message)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidMessage):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, ErrAIUnavailable):
			c.JSON(http.StatusBadGateway, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

func (s *HTTPServer) memories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"memories": s.service.Memories(),
	})
}

func (s *HTTPServer) messages(c *gin.Context) {
	limit := 50
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "limit must be a positive integer",
			})
			return
		}
		limit = parsedLimit
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": s.service.RecentMessages(limit),
	})
}

func (s *HTTPServer) trace(c *gin.Context) {
	traceID := strings.TrimSpace(c.Param("trace_id"))
	if traceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "trace_id is required",
		})
		return
	}

	trace, ok := s.service.Trace(traceID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "trace not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trace": trace,
	})
}
