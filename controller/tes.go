package controller

import (
    "encoding/json"
    "database/sql"
    "net/http"
)

type Tes struct {
    Id   int    `json:"id"`
    Name string `json:"name"`
}

func HandleTes(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    stmt, err := db.Prepare("INSERT INTO tes(name) VALUES(?)")
    if err != nil {
        http.Error(w, "Error preparing insert statement", http.StatusInternalServerError)
        return
    }
    defer stmt.Close()

    _, err = stmt.Exec("fuji")
    if err != nil {
        http.Error(w, "Error inserting data", http.StatusInternalServerError)
        return
    }

    rows, err := db.Query("SELECT id, name FROM tes")
    if err != nil {
        http.Error(w, "Error querying data", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var results []Tes
    for rows.Next() {
        var tes Tes
        if err := rows.Scan(&tes.Id, &tes.Name); err != nil {
            http.Error(w, "Error scanning row", http.StatusInternalServerError)
            return
        }
        results = append(results, tes)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(results)
}
