package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/shmel1k/qumomf/internal/vshard/orchestrator"

	"github.com/shmel1k/qumomf/internal/vshard"

	"github.com/shmel1k/qumomf/internal/storage"
)

const (
	querySaveSnapshot = `INSERT INTO snapshots(cluster_name, created_at, data) 
							VALUES(?, ?, ?)`
	querySaveRecoveries = `INSERT INTO recoveries(cluster_name, created_at, data) 
							VALUES(?, ?, ?)`
	initDatabaseQueries = `CREATE TABLE IF NOT EXISTS snapshots (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"cluster_name" TEXT,
		"created_at" INTEGER,
		"data" BLOB
	  );
	CREATE TABLE IF NOT EXISTS recoveries (
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

type sqlite struct {
	db     *sql.DB
	config Config
}

type Config struct {
	FileName       string
	ConnectTimeout time.Duration
	QueryTimeout   time.Duration
}

func NewSQLiteStorage(cfg Config) (storage.Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.QueryTimeout)
	defer cancel()

	db, err := sql.Open("sqlite3", cfg.FileName)
	if err != nil {
		return &sqlite{}, err
	}

	db.SetMaxOpenConns(1)

	err = createTables(ctx, db)
	if err != nil {
		return nil, err
	}

	return &sqlite{
		db:     db,
		config: cfg,
	}, nil
}

func (s *sqlite) SaveSnapshot(ctx context.Context, clusterName string, snapshot vshard.Snapshot) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, querySaveSnapshot, clusterName, snapshot.Created, data)

	return err
}

func (s *sqlite) SaveRecovery(ctx context.Context, recovery orchestrator.Recovery) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	data, err := json.Marshal(recovery)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, querySaveRecoveries, recovery.ClusterName, recovery.EndTimestamp, data)

	return err
}

func (s *sqlite) GetClusterLastSnapshot(ctx context.Context, clusterName string) (vshard.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	data := make([]byte, 0)
	row := s.db.QueryRowContext(ctx, queryGetLastSnapshot, clusterName)

	var ns vshard.Snapshot
	err := row.Scan(&data)
	if err == sql.ErrNoRows {
		return ns, ErrEmptyResult
	}
	err = json.Unmarshal(data, &ns)

	return ns, err
}

func (s *sqlite) GetRecoveries(ctx context.Context, clusterName string) ([]orchestrator.Recovery, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()

	data := make([]byte, 0)
	resp := make([]orchestrator.Recovery, 0)
	rows, err := s.db.QueryContext(ctx, queryGetRecoveries, clusterName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&data)
		if err != nil {
			return nil, err
		}

		var recovery orchestrator.Recovery
		err = json.Unmarshal(data, &recovery)
		if err != nil {
			return nil, err
		}

		resp = append(resp, recovery)
	}

	return resp, err
}

func createTables(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, initDatabaseQueries)

	return err
}
