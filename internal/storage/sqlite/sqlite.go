package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mattn/go-sqlite3"

	"url-shortener/internal/storage"
)

type Storage struct {
	db         *sql.DB
	stmtInsert *sql.Stmt
	stmtSelect *sql.Stmt
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS url(
		id INTEGER PRIMARY KEY,
		alias TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL);
	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	`); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmtInsert, err := db.Prepare("INSERT INTO url(url, alias) VALUES(?, ?)")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmtSelect, err := db.Prepare("SELECT url FROM url WHERE alias = ?")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db, stmtInsert, stmtSelect}, nil
}

// SaveURL реализует интерфейс URLSaver
func (s *Storage) SaveURL(urlToSave, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	res, err := s.stmtInsert.Exec(urlToSave, alias)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	var resURL string

	err := s.stmtSelect.QueryRow(alias).Scan(&resURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}

		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return resURL, nil
}

func (s *Storage) Close() error {
	const op = "storage.sqlite.Close"
	errs := make([]string, 0, 3)
	if err := s.stmtInsert.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := s.stmtSelect.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := s.db.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) != 0 {
		return fmt.Errorf("%s: %s", op, strings.Join(errs, ", "))
	}
	return nil
}

// TODO: implement method
// func (s *Storage) DeleteURL(alias string) error
