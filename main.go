package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/open-policy-agent/opa/plugins"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
	"github.com/rs/xid"
	"log"
	"net/http"
	"os"
)

const (
	databaseFilePathEnvVar = "OPA_RBAC_DATABASE_FILE"
	policyDataPath         = "/rbac/data"
)

type Server struct {
	db            *sql.DB
	httpServer    *http.Server
	pluginManager *plugins.Manager
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

	pluginManager, err := plugins.New([]byte{}, xid.New().String(), inmem.New())
	if err != nil {
		return nil, err
	}
	server.pluginManager = pluginManager

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
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.writePolicyData([]byte("hello"))
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	log.Printf(
		"Check request: Can '%s' '%s' '%s'?",
		check.UserId,
		check.Permission,
		check.Object)
	w.WriteHeader(http.StatusForbidden)
}

func (s *Server) writePolicyData(data []byte) error {
	store := s.pluginManager.Store
	path, ok := storage.ParsePath(policyDataPath)
	if !ok {
		return errors.New("Failed to parse path.")
	}

	txn := storage.NewTransactionOrDie(context.Background(), store, storage.WriteParams)
	_, err := store.Read(context.Background(), txn, path)
	if err != nil {
		// Not found is fine, we'll just create the directory.
		if !storage.IsNotFound(err) {
			store.Abort(context.Background(), txn)
			return err
		}
		err = storage.MakeDir(context.Background(), store, txn, path)
		if err != nil {
			store.Abort(context.Background(), txn)
			return err
		}
	}

	// Directory now exists, so we are safe to write.
	err = store.Write(context.Background(), txn, storage.ReplaceOp, path, data)
	if err != nil {
		store.Abort(context.Background(), txn)
		return err
	}

	err = store.Commit(context.Background(), txn)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatal(err)
	}

	server.Start()
}
