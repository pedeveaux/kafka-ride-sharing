package rides_db

import (
    "database/sql"
    _ "github.com/lib/pq"
    "log"
)

var DB *sql.DB

func Init(connStr string) error {
    var err error
    DB, err = sql.Open("postgres", connStr)
    if err != nil {
        return err
    }

    if err = DB.Ping(); err != nil {
        return err
    }

    log.Println("✅ Connected to PostgreSQL")
    return nil
}