package main

import (
    "database/sql"
    "log"
    "fmt"
)

func initializeDatabase(db *sql.DB) {
    _, err := db.Exec(`CREATE TABLE IF NOT EXISTS leaderboard (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            wpm INTEGER NOT NULL,
            date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`)

    if err != nil {
        fmt.Println("Error Creating Table")
        log.Fatal(err)
    }
}
