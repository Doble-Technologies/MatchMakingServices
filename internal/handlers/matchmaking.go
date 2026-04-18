package handlers

import (
	"log"
	"mm/service/pkg/ws"

	"github.com/gin-gonic/gin"
)

func MatchmakingWs(c *gin.Context) {
	conn, err := ws.Upgrade(c.Writer, c.Request)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	playerID := c.Query("player_id")
	if playerID == "" {
		log.Println("No player_id provided, closing connection")
		return
	}
	log.Println("Sending to Redis Cache, Matchmaking worker will match")

	log.Printf("User Connected: %s\n", playerID)
	ws.Send(conn, "Connected")

	ws.Read(conn)
}
