package querylog

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/st3v3nmw/beacon/internal/config"
	"github.com/st3v3nmw/beacon/internal/types"
)

var (
	DataDir string
	DB      *sql.DB
	QL      *QueryLogger
)

const schema = `
CREATE TABLE IF NOT EXISTS queries (
	id INTEGER PRIMARY KEY,
	hostname VARCHAR(255) NOT NULL,
	ip VARCHAR(50) NOT NULL,
	domain VARCHAR(255) NOT NULL,
	query_type VARCHAR(20) NOT NULL,
	cached BOOLEAN NOT NULL,
	blocked BOOLEAN NOT NULL,
	block_reason VARCHAR(50) NULL,
	upstream VARCHAR(50) NULL,
	response_code VARCHAR(255) NOT NULL,
    response_time_ms INTEGER NOT NULL,
	prefetched BOOLEAN NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS query_patterns (
	domain VARCHAR(255) NOT NULL,
	occurrences INTEGER NOT NULL,
	prefetch TEXT NOT NULL
);
`

func NewDB() (err error) {
	DB, err = sql.Open("sqlite3", DataDir+"/querylog.db")
	if err != nil {
		return err
	}

	// Run migrations
	_, err = DB.Exec(schema)
	return
}

type QueryLog struct {
	Hostname       string    `json:"hostname"`
	IP             string    `json:"ip"`
	Domain         string    `json:"domain"`
	QueryType      string    `json:"query_type"`
	Cached         bool      `json:"cached"`
	Blocked        bool      `json:"blocked"`
	BlockReason    *string   `json:"block_reason"`
	Upstream       *string   `json:"upstream"`
	ResponseCode   string    `json:"response_code"`
	ResponseTimeMs int       `json:"response_time_ms"`
	Prefetched     bool      `json:"prefetched"`
	Timestamp      time.Time `json:"timestamp"`
}

type QueryLogger struct {
	queryChan chan *QueryLog
	queue     types.ThreadSafeSlice[*QueryLog]
	wg        sync.WaitGroup
	shutdown  chan struct{}
}

func Collect() {
	QL = &QueryLogger{
		queryChan: make(chan *QueryLog, 1_000),
		shutdown:  make(chan struct{}),
	}

	Broadcaster = &QueryBroadcaster{
		clients: make(map[chan *QueryLog]bool),
	}

	QL.wg.Add(1)
	go QL.worker()
}

func (ql *QueryLogger) Log(q *QueryLog) {
	select {
	case ql.queryChan <- q:
	default:
		slog.Warn("QueryLogger channel full - dropping query")
	}
}

func (ql *QueryLogger) worker() {
	defer ql.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case query := <-ql.queryChan:
			ql.queue.Append(query)

			Broadcaster.broadcast(query)
		case <-ticker.C:
			if ql.queue.Len() > 0 {
				ql.flush()
			}

		case <-ql.shutdown:
			if ql.queue.Len() > 0 {
				ql.flush()
			}
			return
		}
	}
}

func (ql *QueryLogger) flush() {
	tx, err := DB.Begin()
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		return
	}

	stmt, err := tx.Prepare(`
		INSERT INTO queries (
			hostname, ip, domain, query_type,
			cached, blocked, block_reason, upstream,
			response_code, response_time_ms, prefetched, timestamp
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`)
	if err != nil {
		slog.Error("Failed to prepare statement", "error", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	for q := range ql.queue.Iterator() {
		_, err := stmt.Exec(
			q.Hostname, q.IP, q.Domain, q.QueryType,
			q.Cached, q.Blocked, q.BlockReason, q.Upstream,
			q.ResponseCode, q.ResponseTimeMs, q.Prefetched, q.Timestamp,
		)
		if err != nil {
			slog.Error("Failed to insert query", "error", err)
			tx.Rollback()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		tx.Rollback()
		return
	}

	ql.queue.Clear()
}

func (ql *QueryLogger) Shutdown() {
	close(ql.shutdown)
	ql.wg.Wait()
}

func DeleteOldQueries() error {
	const query = `
		DELETE FROM queries
		WHERE timestamp < datetime('now', ?)
	`

	stmt, err := DB.Prepare(query)
	if err != nil {
		slog.Error("failed to prepare query:", "error", err)
		return err
	}
	defer stmt.Close()

	retention := config.All.QueryLog.Retention
	offset := fmt.Sprintf("-%d minutes", int(retention.Minutes()))
	_, err = stmt.Exec(offset)
	if err != nil {
		slog.Error("failed to execute query:", "error", err)
		return err
	}

	return nil
}
