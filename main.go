package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

const (
	databaseFilePathEnvVar = "OPA_RBAC_DATABASE_FILE"
)

type Server struct {
	db *sql.DB
}

func NewServer() (*Server, error) {
	databaseFilePath := os.Getenv(databaseFilePathEnvVar)
	if databaseFilePath == "" {
		return nil, errors.New(fmt.Sprintf("%s not set", databaseFilePathEnvVar))
	}

	db, err := sql.Open("sqlite3", databaseFilePath)
	if err != nil {
		return nil, err
	}

	return &Server{db: db}, nil
}

func main() {
	_, err := NewServer()
	if err != nil {
		log.Fatal(err)
	}
}
