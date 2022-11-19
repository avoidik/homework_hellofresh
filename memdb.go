package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type InitializerFunc func(*Database) error

// NewMemDatabaseStore prepares connection to database
func NewMemDatabaseStore(initFunction InitializerFunc) (*Database, func(), error) {

	// open or create database file
	openFileDB, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, nil, err
	}

	// close database on initialization error
	defer func() {
		if err != nil {
			openFileDB.Close()
		}
	}()

	// check connection
	if err = openFileDB.Ping(); err != nil {
		return nil, nil, err
	}

	// preapre db
	db := &Database{DB: openFileDB}

	if initFunction != nil {
		err = initFunction(db)
		if err != nil {
			return nil, nil, err
		}
	}

	// cleaner
	closeFunc := func() {
		openFileDB.Close()
	}

	return db, closeFunc, nil
}
