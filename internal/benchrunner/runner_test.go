package benchrunner

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pg_gobench/internal/benchmark"
	"pg_gobench/internal/benchmarkrun"
)

func TestRunnerJoinProfileExecutesJoinAndAggregationQueriesThroughDatabaseSQL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		db       *sql.DB
		recorder *callRecorder
	)
	db, recorder = openDriverDB(t, testDriverConfig{
		onQuery: func(call recordedCall) driver.Rows {
			switch {
			case strings.Contains(call.query, "JOIN pg_gobench.branches"):
				recorder.markObserved("join")
				return rowsWithColumns([]string{"account_id", "account_name", "branch_name", "teller_name", "balance"}, [][]driver.Value{{int64(1), "account-1", "branch-1", "teller-1", int64(0)}})
			case strings.Contains(call.query, "GROUP BY a.branch_id"):
				recorder.markObserved("aggregation")
				cancel()
				return rowsWithColumns([]string{"branch_id", "account_count", "total_balance"}, [][]driver.Value{{int64(1), int64(1), int64(0)}})
			default:
				return rowsWithColumns([]string{"ignored"}, nil)
			}
		},
	})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	run, err := New(db).Start(ctx, benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileJoin,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	waitErr := run.Wait()
	if !errors.Is(waitErr, context.Canceled) {
		t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
	}

	if !recorder.saw("join") {
		t.Fatalf("observed calls = %#v, want join workload query", recorder.snapshot())
	}
	if !recorder.saw("aggregation") {
		t.Fatalf("observed calls = %#v, want aggregation workload query", recorder.snapshot())
	}
}

func TestRunnerLockProfileExecutesLockAndHotUpdateQueriesThroughDatabaseSQL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		db       *sql.DB
		recorder *callRecorder
	)
	db, recorder = openDriverDB(t, testDriverConfig{
		onExec: func(call recordedCall) error {
			switch {
			case strings.Contains(call.query, "SET LOCAL lock_timeout"):
				recorder.markObserved("set-lock-timeout")
			case strings.Contains(call.query, "SET balance = balance + $1"):
				recorder.markObserved("hot-update")
			}
			return nil
		},
		onCommit: func() error {
			if recorder.saw("hot-update") {
				cancel()
			}
			return nil
		},
		onQuery: func(call recordedCall) driver.Rows {
			if strings.Contains(call.query, "FOR UPDATE NOWAIT") {
				recorder.markObserved("lock-contention")
			}
			return rowsWithColumns([]string{"id"}, [][]driver.Value{{int64(1)}})
		},
	})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	run, err := New(db).Start(ctx, benchmark.StartOptions{
		Scale:           1,
		Clients:         2,
		DurationSeconds: 600,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileLock,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	waitErr := run.Wait()
	if !errors.Is(waitErr, context.Canceled) {
		t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
	}

	if !recorder.saw("set-lock-timeout") {
		t.Fatalf("observed calls = %#v, want lock-timeout setup", recorder.snapshot())
	}
	if !recorder.saw("lock-contention") {
		t.Fatalf("observed calls = %#v, want explicit lock-contention query", recorder.snapshot())
	}
	if !recorder.saw("hot-update") {
		t.Fatalf("observed calls = %#v, want hot-update query", recorder.snapshot())
	}
	if !containsCall(recorder.snapshot(), "begin", true) {
		t.Fatalf("observed calls = %#v, want transaction begin", recorder.snapshot())
	}
	if !containsCall(recorder.snapshot(), "commit", true) {
		t.Fatalf("observed calls = %#v, want transaction commit", recorder.snapshot())
	}
}

func TestRunnerReadProfileExecutesPointAndRangeQueriesThroughDatabaseSQL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		db       *sql.DB
		recorder *callRecorder
	)
	db, recorder = openDriverDB(t, testDriverConfig{
		onQuery: func(call recordedCall) driver.Rows {
			switch {
			case strings.Contains(call.query, "WHERE id ="):
				recorder.markObserved("point-read")
				return rowsWithColumns([]string{"balance", "name"}, [][]driver.Value{{int64(0), "account-1"}})
			case strings.Contains(call.query, "WHERE branch_id ="):
				recorder.markObserved("range-read")
				cancel()
				return rowsWithColumns([]string{"id", "balance"}, [][]driver.Value{{int64(1), int64(0)}, {int64(2), int64(0)}})
			default:
				return rowsWithColumns([]string{"ignored"}, nil)
			}
		},
	})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	run, err := New(db).Start(ctx, benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileRead,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	waitErr := run.Wait()
	if !errors.Is(waitErr, context.Canceled) {
		t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
	}

	if !recorder.saw("point-read") {
		t.Fatalf("observed calls = %#v, want point read", recorder.snapshot())
	}
	if !recorder.saw("range-read") {
		t.Fatalf("observed calls = %#v, want range read", recorder.snapshot())
	}
}

func TestRunnerWriteProfileExecutesInsertAndUpdateThroughDatabaseSQL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		db       *sql.DB
		recorder *callRecorder
	)
	db, recorder = openDriverDB(t, testDriverConfig{
		onExec: func(call recordedCall) error {
			switch {
			case strings.Contains(call.query, "INSERT INTO pg_gobench.history"):
				recorder.markObserved("history-insert")
			case strings.Contains(call.query, "UPDATE pg_gobench.accounts"):
				recorder.markObserved("account-update")
				cancel()
			}
			return nil
		},
	})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	run, err := New(db).Start(ctx, benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileWrite,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	waitErr := run.Wait()
	if !errors.Is(waitErr, context.Canceled) {
		t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
	}

	if !recorder.saw("history-insert") {
		t.Fatalf("observed calls = %#v, want history insert", recorder.snapshot())
	}
	if !recorder.saw("account-update") {
		t.Fatalf("observed calls = %#v, want account update", recorder.snapshot())
	}
}

func TestRunnerMixedProfileUsesReadPercentToChooseFamily(t *testing.T) {
	t.Run("100 percent read never executes write workload", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var (
			db       *sql.DB
			recorder *callRecorder
		)
		db, recorder = openDriverDB(t, testDriverConfig{
			onExec: func(call recordedCall) error {
				if isWorkloadWrite(call.query) {
					return fmt.Errorf("unexpected write workload query: %s", call.query)
				}
				return nil
			},
			onQuery: func(call recordedCall) driver.Rows {
				if isWorkloadRead(call.query) {
					recorder.markObserved("mixed-read")
					cancel()
				}
				if strings.Contains(call.query, "SELECT balance, name") {
					return rowsWithColumns([]string{"balance", "name"}, [][]driver.Value{{int64(0), "account-1"}})
				}
				return rowsWithColumns([]string{"id", "balance"}, [][]driver.Value{{int64(1), int64(0)}})
			},
		})
		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Fatalf("Close db: %v", err)
			}
		})

		run, err := New(db).Start(ctx, benchmark.StartOptions{
			Scale:           1,
			Clients:         1,
			DurationSeconds: 600,
			WarmupSeconds:   10,
			Profile:         benchmark.ProfileMixed,
			ReadPercent:     intPtr(100),
		})
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}

		if waitErr := run.Wait(); !errors.Is(waitErr, context.Canceled) {
			t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
		}
		if !recorder.saw("mixed-read") {
			t.Fatalf("observed calls = %#v, want read workload", recorder.snapshot())
		}
	})

	t.Run("0 percent read never executes read workload", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var (
			db       *sql.DB
			recorder *callRecorder
		)
		db, recorder = openDriverDB(t, testDriverConfig{
			onExec: func(call recordedCall) error {
				if strings.Contains(call.query, "INSERT INTO pg_gobench.history") || strings.Contains(call.query, "UPDATE pg_gobench.accounts") {
					recorder.markObserved("mixed-write")
					cancel()
				}
				return nil
			},
			onQuery: func(call recordedCall) driver.Rows {
				if isWorkloadRead(call.query) {
					panic(fmt.Sprintf("unexpected read workload query: %s", call.query))
				}
				return rowsWithColumns([]string{"ignored"}, nil)
			},
		})
		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Fatalf("Close db: %v", err)
			}
		})

		run, err := New(db).Start(ctx, benchmark.StartOptions{
			Scale:           1,
			Clients:         1,
			DurationSeconds: 600,
			WarmupSeconds:   10,
			Profile:         benchmark.ProfileMixed,
			ReadPercent:     intPtr(0),
		})
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}

		if waitErr := run.Wait(); !errors.Is(waitErr, context.Canceled) {
			t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
		}
		if !recorder.saw("mixed-write") {
			t.Fatalf("observed calls = %#v, want write workload", recorder.snapshot())
		}
	})
}

func TestRunnerTransactionProfileUsesDatabaseSQLTransactions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		db       *sql.DB
		recorder *callRecorder
	)
	db, recorder = openDriverDB(t, testDriverConfig{
		onQuery: func(call recordedCall) driver.Rows {
			if strings.Contains(call.query, "WHERE id =") {
				recorder.markObserved("tx-point-read")
				return rowsWithColumns([]string{"balance", "name"}, [][]driver.Value{{int64(0), "account-1"}})
			}
			recorder.markObserved("tx-range-read")
			return rowsWithColumns([]string{"id", "balance"}, [][]driver.Value{{int64(1), int64(0)}})
		},
		onExec: func(call recordedCall) error {
			if call.inTx && strings.Contains(call.query, "INSERT INTO pg_gobench.history") {
				recorder.markObserved("tx-insert")
			}
			return nil
		},
		onCommit: func() error {
			cancel()
			return nil
		},
	})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	run, err := New(db).Start(ctx, benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileTransaction,
		TransactionMix:  benchmark.TransactionMixReadHeavy,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	waitErr := run.Wait()
	if !errors.Is(waitErr, context.Canceled) {
		t.Fatalf("Wait error = %v, want %v", waitErr, context.Canceled)
	}

	if !recorder.saw("tx-point-read") {
		t.Fatalf("observed calls = %#v, want point read in transaction", recorder.snapshot())
	}
	if !recorder.saw("tx-range-read") {
		t.Fatalf("observed calls = %#v, want range read in transaction", recorder.snapshot())
	}
	if !recorder.saw("tx-insert") {
		t.Fatalf("observed calls = %#v, want insert in transaction", recorder.snapshot())
	}
	if !containsCall(recorder.snapshot(), "begin", true) {
		t.Fatalf("observed calls = %#v, want transaction begin", recorder.snapshot())
	}
	if !containsCall(recorder.snapshot(), "commit", true) {
		t.Fatalf("observed calls = %#v, want transaction commit", recorder.snapshot())
	}
}

func TestCoordinatorMarksBenchrunnerWorkerSQLFailureAsFailed(t *testing.T) {
	db, _ := openDriverDB(t, testDriverConfig{
		onExec: func(call recordedCall) error {
			if strings.Contains(call.query, "UPDATE pg_gobench.accounts") {
				return fmt.Errorf("synthetic workload failure")
			}
			return nil
		},
	})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	coordinator := benchmarkrun.New(New(db))
	state, err := coordinator.Start(context.Background(), benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   10,
		Profile:         benchmark.ProfileWrite,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if state.Status != benchmarkrun.StatusRunning {
		t.Fatalf("Start state = %q, want %q", state.Status, benchmarkrun.StatusRunning)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		return coordinator.State().Status == benchmarkrun.StatusFailed
	})

	failedState := coordinator.State()
	if failedState.Status != benchmarkrun.StatusFailed {
		t.Fatalf("final state = %#v, want failed", failedState)
	}
	if !strings.Contains(failedState.Error, "update account: synthetic workload failure") {
		t.Fatalf("failed state error = %q, want workload failure context", failedState.Error)
	}
}

func TestCoordinatorCountsAndSurfacesLockContentionFailures(t *testing.T) {
	db, _ := openDriverDB(t, testDriverConfig{
		onExec: func(call recordedCall) error {
			if call.inTx && strings.Contains(call.query, "SET LOCAL lock_timeout") {
				return fmt.Errorf("could not obtain lock on row")
			}
			return nil
		},
	})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close db: %v", err)
		}
	})

	coordinator := benchmarkrun.New(New(db))
	state, err := coordinator.Start(context.Background(), benchmark.StartOptions{
		Scale:           1,
		Clients:         1,
		DurationSeconds: 600,
		WarmupSeconds:   0,
		Profile:         benchmark.ProfileLock,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if state.Status != benchmarkrun.StatusRunning {
		t.Fatalf("Start state = %q, want %q", state.Status, benchmarkrun.StatusRunning)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		return coordinator.State().Status == benchmarkrun.StatusFailed
	})

	failedState := coordinator.State()
	if failedState.Status != benchmarkrun.StatusFailed {
		t.Fatalf("final state = %#v, want failed", failedState)
	}
	if !strings.Contains(failedState.Error, "lock contention: could not obtain lock on row") {
		t.Fatalf("failed state error = %q, want lock contention context", failedState.Error)
	}

	results := coordinator.Results()
	if results.Stats.TotalOperations != 1 {
		t.Fatalf("TotalOperations = %d, want %d", results.Stats.TotalOperations, 1)
	}
	if results.Stats.SuccessfulOperations != 0 {
		t.Fatalf("SuccessfulOperations = %d, want %d", results.Stats.SuccessfulOperations, 0)
	}
	if results.Stats.FailedOperations != 1 {
		t.Fatalf("FailedOperations = %d, want %d", results.Stats.FailedOperations, 1)
	}
	if results.Stats.LatestError != "lock contention: could not obtain lock on row" {
		t.Fatalf("LatestError = %q, want compact lock contention error", results.Stats.LatestError)
	}
}

type testDriverConfig struct {
	onExec   func(call recordedCall) error
	onQuery  func(call recordedCall) driver.Rows
	onBegin  func() error
	onCommit func() error
}

type recordedCall struct {
	kind  string
	query string
	args  []driver.NamedValue
	inTx  bool
}

type callRecorder struct {
	mu       sync.Mutex
	calls    []recordedCall
	observed map[string]bool
}

func (r *callRecorder) record(call recordedCall) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call)
}

func (r *callRecorder) markObserved(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.observed == nil {
		r.observed = map[string]bool{}
	}
	r.observed[name] = true
}

func (r *callRecorder) saw(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.observed[name]
}

func (r *callRecorder) snapshot() []recordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := make([]recordedCall, len(r.calls))
	copy(cloned, r.calls)
	return cloned
}

type testDriver struct {
	config   testDriverConfig
	recorder *callRecorder
}

func (d *testDriver) Open(string) (driver.Conn, error) {
	return &testConn{
		config:   d.config,
		recorder: d.recorder,
	}, nil
}

type testConn struct {
	config   testDriverConfig
	recorder *callRecorder
	mu       sync.Mutex
	inTx     bool
}

func (c *testConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("Prepare should not be called")
}

func (c *testConn) Close() error {
	return nil
}

func (c *testConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *testConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	if c.config.onBegin != nil {
		if err := c.config.onBegin(); err != nil {
			return nil, err
		}
	}

	c.mu.Lock()
	c.inTx = true
	c.mu.Unlock()

	c.recorder.record(recordedCall{kind: "begin", inTx: true})

	return &testTx{
		conn: c,
	}, nil
}

func (c *testConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	call := recordedCall{kind: "exec", query: query, args: cloneNamedValues(args), inTx: c.txActive()}
	c.recorder.record(call)

	if c.config.onExec != nil {
		if err := c.config.onExec(call); err != nil {
			return nil, err
		}
	}

	return driver.RowsAffected(1), nil
}

func (c *testConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	call := recordedCall{kind: "query", query: query, args: cloneNamedValues(args), inTx: c.txActive()}
	c.recorder.record(call)

	if c.config.onQuery != nil {
		return c.config.onQuery(call), nil
	}

	return rowsWithColumns([]string{"ignored"}, nil), nil
}

type testTx struct {
	conn *testConn
}

func (t *testTx) Commit() error {
	if t.conn.config.onCommit != nil {
		if err := t.conn.config.onCommit(); err != nil {
			return err
		}
	}

	t.conn.mu.Lock()
	t.conn.inTx = false
	t.conn.mu.Unlock()
	t.conn.recorder.record(recordedCall{kind: "commit", inTx: true})
	return nil
}

func (t *testTx) Rollback() error {
	t.conn.mu.Lock()
	t.conn.inTx = false
	t.conn.mu.Unlock()
	t.conn.recorder.record(recordedCall{kind: "rollback", inTx: true})
	return nil
}

type testRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func rowsWithColumns(columns []string, values [][]driver.Value) driver.Rows {
	return &testRows{columns: columns, values: values}
}

func (r *testRows) Columns() []string {
	return r.columns
}

func (r *testRows) Close() error {
	return nil
}

func (r *testRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}

	copy(dest, r.values[r.index])
	r.index++
	return nil
}

var testDriverID atomic.Uint64

func openDriverDB(t *testing.T, config testDriverConfig) (*sql.DB, *callRecorder) {
	t.Helper()

	recorder := &callRecorder{}
	driverName := fmt.Sprintf("benchrunner-test-driver-%d", testDriverID.Add(1))
	sql.Register(driverName, &testDriver{
		config:   config,
		recorder: recorder,
	})

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	return db, recorder
}

func cloneNamedValues(args []driver.NamedValue) []driver.NamedValue {
	if len(args) == 0 {
		return nil
	}

	cloned := make([]driver.NamedValue, len(args))
	copy(cloned, args)
	return cloned
}

func (c *testConn) txActive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inTx
}

func containsCall(calls []recordedCall, kind string, inTx bool) bool {
	for _, call := range calls {
		if call.kind == kind && call.inTx == inTx {
			return true
		}
	}
	return false
}

func isWorkloadRead(query string) bool {
	return strings.Contains(query, "SELECT balance, name") || strings.Contains(query, "SELECT id, balance")
}

func isWorkloadWrite(query string) bool {
	return strings.Contains(query, "INSERT INTO pg_gobench.history") || strings.Contains(query, "UPDATE pg_gobench.accounts")
}

func intPtr(value int) *int {
	return &value
}

func waitForCondition(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition not reached before timeout")
}
