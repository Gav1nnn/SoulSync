package main

import (
	"os"
	"strconv"
	"time"

	"soulsync/backend/internal/app"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	aiEngineBaseURL := os.Getenv("AI_ENGINE_BASE_URL")
	if aiEngineBaseURL == "" {
		aiEngineBaseURL = "http://localhost:8000"
	}

	aiEngineTimeout := 120 * time.Second
	if rawTimeout := os.Getenv("AI_ENGINE_TIMEOUT_SECONDS"); rawTimeout != "" {
		if seconds, err := strconv.Atoi(rawTimeout); err == nil && seconds > 0 {
			aiEngineTimeout = time.Duration(seconds) * time.Second
		}
	}

	memoryStorePath := os.Getenv("MEMORY_STORE_PATH")
	if memoryStorePath == "" {
		memoryStorePath = ".data/memory-store.json"
	}
	memoryStore, err := app.NewMemoryStore(memoryStorePath)
	if err != nil {
		panic(err)
	}

	traceStorePath := os.Getenv("TRACE_STORE_PATH")
	if traceStorePath == "" {
		traceStorePath = ".data/trace-store.json"
	}
	traceStore, err := app.NewTraceStoreWithPath(traceStorePath)
	if err != nil {
		panic(err)
	}

	workspaceStorePath := os.Getenv("WORKSPACE_STORE_PATH")
	if workspaceStorePath == "" {
		workspaceStorePath = ".data/workspaces.json"
	}
	workspaceStore, err := app.NewWorkspaceStore(workspaceStorePath)
	if err != nil {
		panic(err)
	}

	service := app.NewService(
		app.NewAIClientWithTimeout(aiEngineBaseURL, aiEngineTimeout),
		traceStore,
		memoryStore,
		workspaceStore,
	)
	router := app.NewHTTPServer(service).Router()

	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}
