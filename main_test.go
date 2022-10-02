package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

// populatedDatabaseFromFile populates the specified database from a named file
func populatedDatabaseFromFile(db *sql.DB, file string) error {
	schemaFile, err := os.Open(file)
	if err != nil {
		return err
	}
	schemaBytes, err := ioutil.ReadAll(schemaFile)
	if err != nil {
		return err
	}

	_, err = db.Exec(string(schemaBytes))
	if err != nil {
		return err
	}
	return nil
}

func TestServer(t *testing.T) {
	databaseFile, err := os.CreateTemp(os.TempDir(), "opa-rbac")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Remove(databaseFile.Name())
	}()

	db, err := sql.Open("sqlite3", databaseFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	err = populatedDatabaseFromFile(db, "./database/migration.sql")
	if err != nil {
		t.Fatal(err)
	}
	err = populatedDatabaseFromFile(db, "./database/bootstrap-simple.sql")
	if err != nil {
		t.Fatal(err)
	}

	err = os.Setenv(databaseFilePathEnvVar, databaseFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer()

	if err != nil {
		t.Fatal(err)
	}

	go server.Start()
	for {
		_, err := http.Get("http://localhost:8080")
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}

	testCases := []struct {
		body               RbacCheckRequest
		expectedStatusCode int
	}{
		{
			body: RbacCheckRequest{
				UserId:     "1",
				Project:    "Foo",
				Object:     "account",
				Permission: "create",
			},
			expectedStatusCode: 200,
		},
		{
			body: RbacCheckRequest{
				UserId:     "1",
				Project:    "Foo",
				Object:     "account",
				Permission: "close",
			},
			expectedStatusCode: 200,
		},
		{
			body: RbacCheckRequest{
				UserId:     "1",
				Project:    "Bar",
				Object:     "account",
				Permission: "close",
			},
			expectedStatusCode: 403,
		},
		{
			body: RbacCheckRequest{
				UserId:     "2",
				Project:    "Bar",
				Object:     "payment",
				Permission: "create",
			},
			expectedStatusCode: 200,
		},
	}

	for _, c := range testCases {
		bodyBuffer := bytes.NewBuffer(nil)
		err = json.NewEncoder(bodyBuffer).Encode(c.body)
		if err != nil {
			t.Fatal(err)
		}

		res, err := http.Post("http://localhost:8080/v1/check", "application/json", bodyBuffer)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != c.expectedStatusCode {
			t.Fatal("unexpected status")
		}
	}
}
