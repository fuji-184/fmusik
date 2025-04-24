package utils

import (
    "database/sql"
    "log"
)

func MigrateSQLite(db *sql.DB) {
    _, err := db.Exec(`

        CREATE TABLE IF NOT EXISTS tes (id INTEGER NOT NULL PRIMARY KEY, name TEXT);

    `)
    if err != nil {
        log.Println("Error creating table:", err)
        return
    }
}
