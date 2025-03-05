package websocket

import (
	"log"
	"net/http"

	"github.com/ethanamaher/tuiperacer/internal/race"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func JoinRace(manager *race.RaceManager, response_writer http.ResponseWriter, request *http.Request) {
    raceID := request.URL.Query().Get("race_id")

    conn, err := upgrader.Upgrade(response_writer, request, nil)
    if err != nil {
        log.Println("WebSocket Upgrade Failed", err)
        return
    }

    race := manager.GetRace(raceID)
    if race.Started {
        conn.Close()
        log.Println("Race already started, rejecting connection...")
        return
    }

    race.Mutex.Lock()
    race.Clients[conn] = true
    race.Mutex.Unlock()

    go handleClient(conn, race)
}


func handleClient(conn *websocket.Conn, race *race.Race) {
    defer conn.Close()

    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            log.Println("Client communication error, disconnecting client...", err)
            race.Mutex.Lock()
            delete(race.Clients, conn)
            race.Mutex.Unlock()
            return
        }
        race.Mutex.Lock()

        for client := range race.Clients {
            err := client.WriteMessage(websocket.TextMessage, msg)
            if err != nil {
                log.Println("Error sending message", err)
            }
        }
        race.Mutex.Unlock()
    }
}
