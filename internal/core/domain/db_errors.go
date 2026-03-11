package domain

import "errors"

var (
	ErrDBConnection = errors.New("database connection error")
	ErrDBExtension  = errors.New("database extension error")
	ErrDBMigration  = errors.New("database migration error")
)
