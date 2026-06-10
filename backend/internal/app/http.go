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
	router.GET("/api/workspaces/current", s.currentWorkspace)
	router.GET("/api/workspaces/current/summary", s.currentWorkspaceSummary)
	router.GET("/api/agent/tasks/:id", s.agentTask)
	router.POST("/api/agent/tasks", s.createAgentTask)
	router.POST("/api/workspaces", s.connectWorkspace)
	router.POST("/api/chat", s.chat)
	router.PATCH("/api/memories/:id", s.updateMemory)
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

func (s *HTTPServer) updateMemory(c *gin.Context) {
	var request MemoryUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	memory, err := s.service.UpdateMemoryStatus(c.Param("id"), request.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidMemory):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, ErrMemoryMissing):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"memory": memory,
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

func (s *HTTPServer) connectWorkspace(c *gin.Context) {
	var request WorkspaceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	workspace, err := s.service.ConnectWorkspace(request.Path)
	if err != nil {
		if errors.Is(err, ErrInvalidWorkspace) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workspace": workspace,
	})
}

func (s *HTTPServer) currentWorkspace(c *gin.Context) {
	workspace, ok, err := s.service.CurrentWorkspace()
	if err != nil {
		if errors.Is(err, ErrInvalidWorkspace) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"workspace": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workspace": workspace,
	})
}

func (s *HTTPServer) currentWorkspaceSummary(c *gin.Context) {
	summary, ok, err := s.service.CurrentWorkspaceSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ErrWorkspaceMissing.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": summary,
	})
}

func (s *HTTPServer) createAgentTask(c *gin.Context) {
	var request AgentTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	task, err := s.service.CreateAgentTask(request.Goal)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAgentTask), errors.Is(err, ErrWorkspaceMissing):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"task": task,
	})
}

func (s *HTTPServer) agentTask(c *gin.Context) {
	task, err := s.service.AgentTask(c.Param("id"))
	if err != nil {
		if errors.Is(err, ErrAgentTaskMissing) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task": task,
	})
}
