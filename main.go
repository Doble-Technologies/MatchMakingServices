package main

import (
	"fmt"
	"mm/service/internal/server"
)

// @title Matchmaking Service
// @version 0.2
// @description Testing Swagger APIs.
// @termsOfService http://swagger.io/terms/
func main() {
	fmt.Println("Starting up")
	server.Start("0.0.0.0:9333")
}
