package benchrunner

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const sqlStateLockNotAvailable = "55P03"

type countedOperationError struct {
	err error
}

func (e countedOperationError) Error() string {
	return e.err.Error()
}

func (e countedOperationError) Unwrap() error {
	return e.err
}

func countFailedOperation(err error) error {
	if err == nil {
		return nil
	}
	return countedOperationError{err: err}
}

func isCountedOperationError(err error) bool {
	var counted countedOperationError
	return errors.As(err, &counted)
}

func isLockNotAvailableError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == sqlStateLockNotAvailable {
		return true
	}
	return strings.Contains(err.Error(), "SQLSTATE "+sqlStateLockNotAvailable)
}
