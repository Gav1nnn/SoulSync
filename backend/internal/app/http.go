package app

import (
	"errors"
	"net/http"

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
