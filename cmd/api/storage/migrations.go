package storage

import (
	"database/sql"
	"log"
)

type Migrations struct {
	DB      *sql.DB
	infoLog *log.Logger
}

func NewMigrations(db *sql.DB, logger *log.Logger) *Migrations {
	return &Migrations{DB: db, infoLog: logger}
}

func (m *Migrations) Run() error {
	return nil
}
