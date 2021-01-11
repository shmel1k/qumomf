package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	// sqlite3 driver
	_ "github.com/mattn/go-sqlite3"
)

const (
	querySaveSnapshot = `INSERT INTO snapshots(cluster_name, created_at, data) 
							VALUES(?, ?, ?)`
	querySaveRecoveries = `INSERT INTO recoveries(cluster_name, created_at, data) 
							VALUES(?, ?, ?)`
	queryCreateTableSnapshots = `CREATE TABLE IF NOT EXISTS snapshots (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"cluster_name" TEXT,
		"created_at" INTEGER,
		"data" BLOB
	  )`
	queryCreateTableRecoveries = `CREATE TABLE IF NOT EXISTS recoveries (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"cluster_name" TEXT,
		"created_at" INTEGER,
		"data" BLOB
	  )`
	queryGetLastSnapshot = `SELECT data
		FROM snapshots
		WHERE cluster_name = ?
		ORDER BY id DESC limit 1`
	queryGetRecoveries = `SELECT data
		FROM recoveries
		WHERE cluster_name = ?`
)

var (
	ErrEmptyResult = errors.New("empty result")
)

type SaveRequest struct {
	ClusterName string
	CreatedAt   int64
	Data        []byte
}

type Storage interface {
	SaveSnapshot(context.Context, SaveRequest) error
	SaveRecovery(context.Context, SaveRequest) error
	GetClusterLastSnapshot(context.Context, string) ([]byte, error)
	GetRecoveries(context.Context, string) ([][]byte, error)
}

type storage struct {
	db     *sql.DB
	config Config
}

type Config struct {
	FileName       string
	ConnectTimeout time.Duration
	QueryTimeout   time.Duration
}

func NewStorage(cfg Config) (Storage, error) {
	cfg = validateConfig(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	_, err := createFileIfNotExists(cfg.FileName)
	if err != nil {
		return &storage{}, nil
	}

	db, err := sql.Open("sqlite3", cfg.FileName)
	if err != nil {
		return &storage{}, err
	}

	db.SetMaxOpenConns(1)

	err = createTables(ctx, db)
	if err != nil {
		return &storage{}, err
	}

	return &storage{
		db:     db,
		config: cfg,
	}, nil
}

func validateConfig(cfg Config) Config {
	if cfg.FileName == "" {
		cfg.FileName = "qumomf.db"
	}

	if cfg.QueryTimeout == 0 {
		cfg.QueryTimeout = time.Second
	}

	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = time.Second
	}

	return cfg
}

func (s *storage) SaveSnapshot(ctx context.Context, sr SaveRequest) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	stmt, err := s.db.PrepareContext(ctx, querySaveSnapshot)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, sr.ClusterName, sr.CreatedAt, sr.Data)

	return err
}

func (s *storage) SaveRecovery(ctx context.Context, sr SaveRequest) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	stmt, err := s.db.PrepareContext(ctx, querySaveRecoveries)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, sr.ClusterName, sr.CreatedAt, sr.Data)

	return err
}

func (s *storage) GetClusterLastSnapshot(ctx context.Context, clusterName string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	stmt, err := s.db.PrepareContext(ctx, queryGetLastSnapshot)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 0)
	row := stmt.QueryRowContext(ctx, clusterName)
	err = row.Scan(&data)
	if err == sql.ErrNoRows {
		return nil, ErrEmptyResult
	}

	return data, err
}

func (s *storage) GetRecoveries(ctx context.Context, clusterName string) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	stmt, err := s.db.PrepareContext(ctx, queryGetRecoveries)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 0)
	resp := make([][]byte, 0)
	rows, err := stmt.QueryContext(ctx, clusterName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&data)
		if err != nil {
			return nil, err
		}

		resp = append(resp, data)
	}

	return resp, err
}

func createTables(ctx context.Context, db *sql.DB) error {
	for _, q := range []string{queryCreateTableRecoveries, queryCreateTableSnapshots} {
		statement, err := db.Prepare(q)
		if err != nil {
			return err
		}

		_, err = statement.ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func createFileIfNotExists(name string) (*os.File, error) {
	file, err := os.OpenFile(name, os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	err = file.Close()

	return file, err
}
