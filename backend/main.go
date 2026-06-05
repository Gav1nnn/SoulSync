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

	service := app.NewService(app.NewAIClientWithTimeout(aiEngineBaseURL, aiEngineTimeout), app.NewTraceStore(), memoryStore)
	router := app.NewHTTPServer(service).Router()

	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}
