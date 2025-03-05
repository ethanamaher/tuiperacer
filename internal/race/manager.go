package race

import (
    "sync"
    "log"
    "github.com/gorilla/websocket"
)

type Race struct {
    ID      string
    Clients map[*websocket.Conn]bool
    Started bool
    Mutex   sync.Mutex
}

type RaceManager struct {
    Races   map[string]*Race
    Mutex   sync.Mutex
}

func NewRaceManager() *RaceManager {
    return &RaceManager{ Races: make(map[string]*Race) }
}

func (rm *RaceManager) GetRace(raceID string) *Race {
    rm.Mutex.Lock()
    defer rm.Mutex.Lock()

    _, exists := rm.Races[raceID]
    if !exists {
        rm.Races[raceID] = &Race{ ID: raceID, Clients: make(map[*websocket.Conn]bool), Started: false }
    }
    return rm.Races[raceID]
}

func (rm *RaceManager) StartRace(raceID string) bool {
    rm.Mutex.Lock()
    race, exists := rm.Races[raceID]

    if !exists || race.Started {
        rm.Mutex.Unlock()
        log.Println("Error starting race")
        return false
    }

    race.Started = true
    rm.Mutex.Unlock()
    race.Mutex.Lock()

    for client := range race.Clients {
        client.WriteMessage(websocket.TextMessage, []byte("Race Started!"))
    }
    race.Mutex.Unlock()
    return race.Started
}
