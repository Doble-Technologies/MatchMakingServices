package handlers

import (
	"context"
	"log"
	"math/rand/v2"
	"mm/service/pkg/ws"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type MmHandler struct {
	Redis *redis.Client
}

var ctx = context.Background()

func addPlayer(rdb *redis.Client, userID string) {
	//Todo: Replace with db lookup in future
	elo := rand.IntN(11)
	now := time.Now().Unix()
	// Store player info
	err := rdb.HSet(ctx,
		"player:"+userID,
		map[string]interface{}{
			"elo":       elo,
			"joined_at": now,
		},
	).Err()
	if err != nil {
		log.Fatalf("Redis Error: %s", err)
	}

	// Add to matchmaking queue
	err = rdb.ZAdd(ctx,
		"queue:ranked:elo",
		redis.Z{
			Score:  float64(elo),
			Member: userID,
		},
	).Err()
	if err != nil {
		log.Fatalf("Redis Add Error: %s", err)
	}
}

func NewMMHandler(rdb *redis.Client) *MmHandler {
	return &MmHandler{
		Redis: rdb,
	}
}

func (h *MmHandler) MatchmakingWs(c *gin.Context) {
	conn, err := ws.Upgrade(c.Writer, c.Request)
	if err != nil {
		log.Println(err)
		return
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	//Placeholder value we will get player id from jks, this is just to have a placeholder for loigic
	playerID := c.Query("player_id")
	if playerID == "" {
		log.Println("No player_id provided, closing connection")
		return
	}
	log.Printf("Adding to Queue: %s\n\n", playerID)
	addPlayer(h.Redis, playerID)

	log.Printf("User Connected: %s\n", playerID)
	ws.Send(conn, "Queued")

	//Wait for response from redis MATCHED stack(dlpop) and Then
	//ws.Read(conn, h.Redis)
	_, matchErr := ws.WaitForMatch(ctx, h.Redis, playerID)
	if matchErr != nil {
		return
	}
	// Match is found at this point, from here we send connection back to websocket request
	//Return connection here
}
