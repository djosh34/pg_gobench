package benchrunner

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"pg_gobench/internal/benchmark"
)

func TestSetupStatementsTargetOnlyBenchmarkSchemaByDefault(t *testing.T) {
	statements := setupStatements(benchmark.StartOptions{Scale: 2}, benchmark.ResolveScale(2))
	joined := strings.Join(statements, "\n")

	for _, want := range []string{
		"CREATE SCHEMA IF NOT EXISTS pg_gobench",
		"CREATE TABLE IF NOT EXISTS pg_gobench.branches",
		"CREATE TABLE IF NOT EXISTS pg_gobench.tellers",
		"CREATE TABLE IF NOT EXISTS pg_gobench.accounts",
		"CREATE TABLE IF NOT EXISTS pg_gobench.history",
		"INSERT INTO pg_gobench.branches",
		"INSERT INTO pg_gobench.tellers",
		"INSERT INTO pg_gobench.accounts",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("setup SQL missing %q in:\n%s", want, joined)
		}
	}

	for _, forbidden := range []string{
		"DROP SCHEMA",
		" public.",
		"CREATE TABLE IF NOT EXISTS accounts",
		"CREATE TABLE IF NOT EXISTS branches",
		"CREATE TABLE IF NOT EXISTS tellers",
		"CREATE TABLE IF NOT EXISTS history",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("setup SQL unexpectedly contains %q in:\n%s", forbidden, joined)
		}
	}
}

func TestRunnerStartExecutesSchemaSetupThroughDatabaseSQL(t *testing.T) {
	db, recorder := openRecordingDB(t)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	run, err := New(db).Start(ctx, benchmark.StartOptions{
		Scale:           2,
		Clients:         1,
		DurationSeconds: 60,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileRead,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if run == nil {
		t.Fatal("Run = nil, want handle")
	}

	executed := recorder.statements()
	if len(executed) == 0 {
		t.Fatal("executed statements = 0, want setup SQL")
	}
	if !containsStatement(executed, "CREATE SCHEMA IF NOT EXISTS pg_gobench") {
		t.Fatalf("executed statements missing CREATE SCHEMA:\n%s", strings.Join(executed, "\n---\n"))
	}
	if !containsStatement(executed, "INSERT INTO pg_gobench.accounts") {
		t.Fatalf("executed statements missing account seed:\n%s", strings.Join(executed, "\n---\n"))
	}
}

func TestRunnerStartLimitsResetToBenchmarkSchema(t *testing.T) {
	t.Run("reset disabled does not drop schema", func(t *testing.T) {
		db, recorder := openRecordingDB(t)
		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Fatalf("Close db: %v", err)
			}
		})

		_, err := New(db).Start(context.Background(), benchmark.StartOptions{
			Scale:           1,
			Clients:         1,
			DurationSeconds: 60,
			WarmupSeconds:   10,
			Profile:         benchmark.ProfileRead,
		})
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}

		executed := recorder.statements()
		if containsStatement(executed, "DROP SCHEMA") {
			t.Fatalf("executed statements unexpectedly dropped a schema:\n%s", strings.Join(executed, "\n---\n"))
		}
	})

	t.Run("reset enabled drops only pg_gobench", func(t *testing.T) {
		db, recorder := openRecordingDB(t)
		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Fatalf("Close db: %v", err)
			}
		})

		_, err := New(db).Start(context.Background(), benchmark.StartOptions{
			Scale:           1,
			Clients:         1,
			DurationSeconds: 60,
			WarmupSeconds:   10,
			Reset:           true,
			Profile:         benchmark.ProfileRead,
		})
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}

		executed := recorder.statements()
		if !containsStatement(executed, "DROP SCHEMA IF EXISTS pg_gobench CASCADE") {
			t.Fatalf("executed statements missing benchmark schema drop:\n%s", strings.Join(executed, "\n---\n"))
		}
		for _, statement := range executed {
			if strings.Contains(statement, "DROP SCHEMA") && statement != "DROP SCHEMA IF EXISTS pg_gobench CASCADE" {
				t.Fatalf("executed unexpected destructive statement %q", statement)
			}
		}
	})
}

type recordingDriver struct {
	recorder *statementRecorder
}

func (d *recordingDriver) Open(string) (driver.Conn, error) {
	return &recordingConn{recorder: d.recorder}, nil
}

type recordingConn struct {
	recorder *statementRecorder
}

func (c *recordingConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("Prepare should not be called")
}

func (c *recordingConn) Close() error {
	return nil
}

func (c *recordingConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("Begin should not be called")
}

func (c *recordingConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.recorder.record(query)
	return driver.RowsAffected(0), nil
}

type statementRecorder struct {
	mu   sync.Mutex
	list []string
}

func (r *statementRecorder) record(statement string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.list = append(r.list, statement)
}

func (r *statementRecorder) statements() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := make([]string, len(r.list))
	copy(cloned, r.list)
	return cloned
}

var recordingDriverID atomic.Uint64

func openRecordingDB(t *testing.T) (*sql.DB, *statementRecorder) {
	t.Helper()

	recorder := &statementRecorder{}
	driverName := fmt.Sprintf("recording-driver-%d", recordingDriverID.Add(1))
	sql.Register(driverName, &recordingDriver{recorder: recorder})
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db, recorder
}

func containsStatement(statements []string, want string) bool {
	for _, statement := range statements {
		if strings.Contains(statement, want) {
			return true
		}
	}

	return false
}
