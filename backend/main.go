package main

import (
	"os"

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

	service := app.NewService(app.NewAIClient(aiEngineBaseURL), app.NewTraceStore())
	router := app.NewHTTPServer(service).Router()

	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}
