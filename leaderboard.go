package main

import (
    "database/sql"
    "log"
)

type LeaderboardEntry struct {
    Name string
    WPM int
}

func saveToLeaderboard(db *sql.DB, name string, wpm int) {
    _, err := db.Exec(`INSERT INTO leaderboard (name, wpm) VALUES (?, ?)`, name, wpm)
    if err != nil {
        log.Fatal(err)
    }
}

func fetchLeaderboard(db *sql.DB) []LeaderboardEntry {
    rows, err := db.Query(`SELECT name, wpm
        FROM leaderboard
        ORDER BY wpm
        DESC LIMIT 5`)

    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    var leaderboard []LeaderboardEntry
    for rows.Next() {
        var entry LeaderboardEntry
        err := rows.Scan(&entry.Name, &entry.WPM)
        if err != nil {
            log.Fatal(err)
        }
        leaderboard = append(leaderboard, entry)
    }

    return leaderboard
}
