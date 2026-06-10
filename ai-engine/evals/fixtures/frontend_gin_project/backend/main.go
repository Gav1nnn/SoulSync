package main

import "github.com/gin-gonic/gin"

type UserResponse struct {
	ID   string
	Name string
}

type OrderResponse struct {
	ID    string
	Title string
}

func main() {
	router := gin.Default()
	api := router.Group("/api")
	api.GET("/users", listUsers)
	api.GET("/orders", listOrders)
}

func listUsers(c *gin.Context) {
	c.JSON(200, []UserResponse{})
}

func listOrders(c *gin.Context) {
	c.JSON(200, []OrderResponse{})
}
