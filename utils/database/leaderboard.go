package database

import (
    "database/sql"
    "log"
)

type LeaderboardEntry struct {
    Name string
    WPM int
    Accuracy float64
}

// save score to leaderboard in database
func SaveToLeaderboard(db *sql.DB, name string, wpm int, accuracy float64) {
    _, err := db.Exec(`INSERT INTO leaderboard (name, wpm, accuracy) VALUES (?, ?, ?)`, name, wpm, accuracy)
    if err != nil {
        log.Fatal(err)
    }
}

// convert leaderboard from sql to array of entries
func FetchLeaderboard(db *sql.DB) []LeaderboardEntry {
    rows, err := db.Query(`SELECT name, wpm, accuracy
        FROM leaderboard
        ORDER BY wpm DESC, accuracy DESC
        LIMIT 5`)

    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    var leaderboard []LeaderboardEntry
    for rows.Next() {
        var entry LeaderboardEntry
        err := rows.Scan(&entry.Name, &entry.WPM, &entry.Accuracy)
        if err != nil {
            log.Fatal(err)
        }
        leaderboard = append(leaderboard, entry)
    }

    return leaderboard
}
