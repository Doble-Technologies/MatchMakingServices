package ws

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

func Send(conn *websocket.Conn, msg string) {
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		log.Println("write error:", err)
	}
}

// Read Placeholder Function
func _(conn *websocket.Conn, rdb *redis.Client) {
	//This will be rewritten to watch redis for queue:ranked:elo matches:ranked:elo for our specific id.
	//Then send message back to user and close conn
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(string(p))
		err = conn.WriteMessage(messageType, p)
		if err != nil {
			return
		}
	}
}

func WaitForMatch(ctx context.Context, rdb *redis.Client, userID string) (string, error) {
	resultKey := "player:" + userID + ":match"

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()

		case <-ticker.C:
			matchID, err := rdb.Get(ctx, resultKey).Result()
			if err == redis.Nil {
				// Not matched yet, keep waiting
				log.Printf("INFO: user %s still waiting for match...", userID)
				continue
			}
			if err != nil {
				return "", fmt.Errorf("waitForMatch: %w", err)
			}

			// Matched — clean up the result key and return
			rdb.Del(ctx, resultKey)
			return matchID, nil
		}
	}
}
