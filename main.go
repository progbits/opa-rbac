package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
)

const (
	databaseFilePathEnvVar = "OPA_RBAC_DATABASE_FILE"
)

type Server struct {
	db         *sql.DB
	httpServer *http.Server
}

func NewServer() (*Server, error) {
	server := &Server{}

	databaseFilePath := os.Getenv(databaseFilePathEnvVar)
	if databaseFilePath == "" {
		return nil, errors.New(fmt.Sprintf("%s not set", databaseFilePathEnvVar))
	}

	db, err := sql.Open("sqlite3", databaseFilePath)
	if err != nil {
		return nil, err
	}
	server.db = db

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/check", server.handle)

	httpServer := &http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}
	server.httpServer = httpServer

	return server, nil
}

func (s *Server) Start() {
	log.Fatal(s.httpServer.ListenAndServe())
}

type RbacCheckRequest struct {
	UserId     string `json:"user_id,omitempty"`
	Object     string `json:"object,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	check := RbacCheckRequest{}
	err := json.NewDecoder(r.Body).Decode(&check)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf(
		"Check request: Can '%s' '%s' '%s'?",
		check.UserId,
		check.Permission,
		check.Object)
	w.WriteHeader(http.StatusForbidden)
}

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatal(err)
	}

	server.Start()
}
