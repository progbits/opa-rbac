package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/plugins"
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
)

type Server struct {
	db            *sql.DB
	httpServer    *http.Server
	pluginManager *plugins.Manager
	compiler      *ast.Compiler
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

	// Create a new plugin manager without any configuration.
	pluginManager, err := plugins.New([]byte{}, xid.New().String(), inmem.New())
	if err != nil {
		return nil, err
	}
	server.pluginManager = pluginManager

	// Compile the RBAC module.
	module := `
		package rbac

		default allow = false

		allow {
			# Look up the list of projects the user has access too.
			project_roles := data.roles[input.user_id]

			# For each of the roles held by the user for the named project.
			project_role := project_roles[input.project]
			pr := project_role[_]

			# Lookup the permissions for the roles.
			permissions := data.permissions[pr]

			# For each role permission, check if there is a match.
			p := permissions[_]
			p == concat("", [input.permission, ":", input.object])
		}
	`
	compiler, err := ast.CompileModules(map[string]string{"rbac": module})
	if err != nil {
		return nil, err
	}
	server.compiler = compiler

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
		r := rego.New(
			rego.Query("data.rbac.allow"),
			rego.Input(input),
			rego.Compiler(s.compiler),
			rego.Store(s.pluginManager.Store),
			rego.Transaction(txn))

		result, err := r.Eval(context.Background())
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
func (s *Server) loadRbacData() (map[string]interface{}, error) {
	row := s.db.QueryRow("SELECT * FROM rbac_data;")
	raw := ""
	err := row.Scan(&raw)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	err = json.NewDecoder(bytes.NewReader([]byte(raw))).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// writeRbacData writes the specified RBAC data to the OPA store, where is
// accessible on policy evaluation. RBAC data is written to the location
// specified by `policyDataPath`.
func (s *Server) writeRbacData(data map[string]interface{}) error {
	store := s.pluginManager.Store
	path := make([]string, 0)

	txn := storage.NewTransactionOrDie(context.Background(), store, storage.WriteParams)
	err := store.Write(context.Background(), txn, storage.AddOp, path, data)
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
