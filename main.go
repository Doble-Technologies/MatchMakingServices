package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// HTTP endpoint example
func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

// We'll need to define an Upgrader
// this will require a Read and Write buffer size
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// define a reader which will listen for
// new messages being sent to our WebSocket
// endpoint
func reader(conn *websocket.Conn) {
	for {
		// read in a message
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		// print out that message for clarity
		fmt.Println(string(p))

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}

	}
}

// Websocket for when a user wants to queue
func matchmakingWs(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	//Parameters
	playerID := r.URL.Query().Get("player_id")
	if playerID == "" {
		err := ws.Close()
		if err != nil {
			return
		}
	}

	log.Printf("User Connected: %s\n", playerID)
	err = ws.WriteMessage(1, []byte("Connected"))
	//Add them to the "match making pool"

	reader(ws)
}

//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

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/queue", matchmakingWs)
}

func main() {
	fmt.Println("Starting up")
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
