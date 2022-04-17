package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/open-policy-agent/opa/plugins"
	"github.com/open-policy-agent/opa/plugins/discovery"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
	"github.com/rs/xid"
	"log"
	"net/http"
	"os"
)

const (
	databaseFilePathEnvVar = "OPA_RBAC_DATABASE_FILE"
	opaConfigEnvVar        = "OPA_CONFIG"
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

	// Try and load the OPA configuration.
	opaConfigFilePath := os.Getenv(opaConfigEnvVar)
	if opaConfigFilePath == "" {
		log.Fatalf("%s not set.", opaConfigEnvVar)
	}
	opaConfigBuf, err := os.ReadFile(opaConfigFilePath)
	if err != nil {
		return nil, err
	}

	// Create a new plugin manager so we can register the `Discovery` plugin.
	pluginManager, err := plugins.New(opaConfigBuf, xid.New().String(), inmem.New())
	if err != nil {
		return nil, err
	}

	// Register the `Discovery` plugin to periodically download new bundles.
	disc, err := discovery.New(pluginManager)
	if err != nil {
		return nil, err
	}
	pluginManager.Register("discovery", disc)

	server.pluginManager = pluginManager

	// Start the plugin engine
	err = pluginManager.Init(context.Background())
	if err != nil {
		return nil, err
	}
	err = pluginManager.Start(context.Background())
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Server) Start() {
	log.Fatal(s.httpServer.ListenAndServe())
}

type RbacCheckRequest struct {
	UserId     string `json:"user_id,omitempty"`
	Project    string `json:"project,omitempty"`
	Object     string `json:"object,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	input := RbacCheckRequest{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Load the most recent RBAC data from the database.
	rbacData, err := s.loadRbacData()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Write the RBAC data to be used by the query.
	err = s.writeRbacData(rbacData)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Build the query.
	allow := false
	query := func(txn storage.Transaction) error {
		result, err := rego.New(
			rego.Query("data.example.rbac.allow"),
			rego.Input(input),
			rego.Compiler(s.pluginManager.GetCompiler()),
			rego.Store(s.pluginManager.Store),
			rego.Dump(os.Stdout),
			rego.Trace(true),
			rego.Transaction(txn)).Eval(context.Background())
		if err != nil {
			return err
		} else if len(result) == 0 {
			return errors.New("Undefined query.")
		} else if len(result) > 1 {
			return errors.New("Attempt to evaluate non-boolean decision.")
		} else if value, ok := result[0].Expressions[0].Value.(bool); !ok {
			return errors.New("Attempt to evaluate non-boolean decision.")
		} else {
			allow = value
		}
		return nil
	}

	// Execute the query.
	err = storage.Txn(
		context.Background(),
		s.pluginManager.Store,
		storage.TransactionParams{},
		query)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf(
		"Check request: Can '%s' '%s' '%s' in project '%s'? %v",
		input.UserId,
		input.Permission,
		input.Object,
		input.Project,
		allow)
	if allow {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusForbidden)
}

// loadRbacData loads the most recent RBAC data from the database in JSON
// format, ready to be consumed by dependant policies.
func (s *Server) loadRbacData() ([]byte, error) {
	row := s.db.QueryRow("SELECT * FROM rbac_data;")
	data := make([]byte, 0)
	err := row.Scan(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// writeRbacData writes the specified RBAC data to the OPA store, where is
// accessible on policy evaluation. RBAC data is written to the location
// specified by `policyDataPath`.
func (s *Server) writeRbacData(data []byte) error {
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
