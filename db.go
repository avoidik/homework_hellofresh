package main

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseStore interface {
	IsConnected() bool
	InsertConfig(cfg *Config) (int, error)
	GetConfigById(id int) (*Config, error)
	GetConfigByName(name string) (*Config, error)
	GetConfigs() (*[]Config, error)
	DeleteConfigByName(name string) error
	UpdateConfigByName(name string, cfg *Config) error
}

type Database struct {
	*sqlx.DB
}

type Metadata map[string]interface{}

type Config struct {
	ID       int       `db:"id" json:"id"`
	Name     string    `db:"name" json:"name"`
	Metadata *Metadata `db:"metadata" json:"metadata"`
	Created  time.Time `db:"created_at" json:"-"`
}

type Monitoring struct {
	Enabled bool `db:"enabled" json:"enabled"`
}

type Cpu struct {
	Enabled bool   `db:"enabled" json:"enabled"`
	Value   string `db:"value" json:"value"`
}

type Limits struct {
	Cpu Cpu `db:"cpu" json:"cpu"`
}

const databaseFile = "state.db"

// Scan performs custom-type conversion, deserialize stream of bytes into struct
func (m *Metadata) Scan(src interface{}) error {

	if src == nil {
		return errors.New("empty input data")
	}

	// try to translate whether string or stream of bytes
	switch t := src.(type) {
	case []byte:
		if err := json.Unmarshal(t, &m); err != nil {
			return err
		}
	case string:
		buf := []byte(t)
		if err := json.Unmarshal(buf, &m); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unexpected data type %t", t)
	}

	return nil
}

// Value performs custom-type conversion, serialize struct into stream of bytes
func (m *Metadata) Value() (driver.Value, error) {
	return json.Marshal(&m)
}

// NewDatabaseStore prepares connection to database
func NewDatabaseStore() (*Database, func(), error) {

	// store database somewhere else if defined
	path := getStringOrDefault("SERVE_DATA", databaseFile)

	// check if database file exists
	initDB := !existDir(path)

	// open or create database file
	openFileDB, err := sqlx.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_loc=auto", path))
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
	if initDB {
		if err := initializeDb(openFileDB); err != nil {
			return nil, nil, err
		}
	}

	db := &Database{DB: openFileDB}

	// cleaner
	closeFunc := func() {
		openFileDB.Close()
	}

	return db, closeFunc, nil
}

// initializeDb populates database file with structure and data
func initializeDb(db *sqlx.DB) error {

	// use transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// rollback or commit
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// create required table
	stmt := `
	CREATE TABLE configs (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(255) NOT NULL,
		metadata TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);
	CREATE INDEX idx_configs_created ON configs(created_at);
	`

	// execute DDL statement
	_, err = tx.Exec(stmt)
	if err != nil {
		return err
	}

	// insert initial data
	stmt = `INSERT INTO configs (name, metadata, created_at) VALUES (?, ?, datetime('now', ?))`

	metadataDC1 := `{"monitoring":{"enabled":"true"},"limits":{"cpu":{"enabled":"false","value":"300m"}}}`
	metadataDC2 := `{"monitoring":{"enabled":"true"},"limits":{"cpu":{"enabled":"true","value":"250m"}}}`

	dc := map[int][]interface{}{
		0: {"datacenter-1", metadataDC1, "-60 days"},
		1: {"datacenter-2", metadataDC2, "-14 days"},
	}

	// execute DML statement
	for _, v := range dc {
		_, err = tx.Exec(stmt, v...)
		if err != nil {
			return err
		}
	}

	return nil
}

// InsertConfig inserts Config struct into database file
func (db *Database) InsertConfig(cfg *Config) (int, error) {

	// insert statement
	stmt := `INSERT INTO configs (name, metadata, created_at) VALUES (?, ?, datetime('now'))`

	// execute DML statement
	result, err := db.Exec(stmt, cfg.Name, cfg.Metadata)
	if err != nil {
		return 0, err
	}

	// get newly created record id
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetConfigById retrieves Config by its id
func (db *Database) GetConfigById(id int) (*Config, error) {
	stmt := `SELECT id, name, metadata, created_at FROM configs	WHERE id = ?`

	row := db.QueryRow(stmt, id)

	cfg := &Config{}

	err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Metadata, &cfg.Created)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetConfigByName retrieves Config by its name
func (db *Database) GetConfigByName(name string) (*Config, error) {
	stmt := `SELECT id, name, metadata, created_at FROM configs	WHERE name = ?`

	row := db.QueryRow(stmt, name)

	cfg := &Config{}

	err := row.Scan(&cfg.ID, &cfg.Name, &cfg.Metadata, &cfg.Created)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetConfigs retrieves all Configs
func (db *Database) GetConfigs() (*[]Config, error) {
	stmt := `SELECT id, name, metadata, created_at FROM configs ORDER BY created_at ASC`

	cfgs := []Config{}
	if err := db.Select(&cfgs, stmt); err != nil {
		return nil, err
	}

	return &cfgs, nil
}

// DeleteConfigByName removes Config by its name
func (db *Database) DeleteConfigByName(name string) error {
	stmt := `DELETE FROM configs WHERE name = ?`

	query, err := db.Prepare(stmt)
	if err != nil {
		return err
	}

	_, err = query.Exec(name)
	return err
}

func (db *Database) UpdateConfigByName(name string, cfg *Config) error {
	stmt := `UPDATE configs SET metadata = ? WHERE name = ?`

	// execute DML statement
	_, err := db.Exec(stmt, cfg.Metadata, name)
	if err != nil {
		return err
	}

	return nil
}

// IsConnected verifies connection to database
func (db *Database) IsConnected() bool {
	if err := db.DB.Ping(); err != nil {
		return false
	}
	return true
}
