package journalparser

import (
	"database/sql"
	"sync"
)

type JournalRepositoryI interface {
	ConnectToDB(outputDirectory string) error
	CheckAliveAfterError() error
	Close()
	CreateTables() error
	GetEntries(limit int, offset int) ([]JournalEntry, error)
	InsertEntries(entries []JournalEntry) error
}

type journalRepository struct {
	db      *sql.DB
	dbMutex *sync.Mutex
}

func NewJournalRepository() JournalRepositoryI {
	return &journalRepository{
		db:      nil,
		dbMutex: new(sync.Mutex),
	}
}

func (r *journalRepository) ConnectToDB(outputDirectory string) error {
	db, err := sql.Open("sqlite3", "./"+outputDirectory+"/journal.db")
	if err != nil {
		return err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return err
	}
	r.dbMutex.Lock()
	defer r.dbMutex.Unlock()
	r.db = db
	return nil
}

func (r *journalRepository) CheckAliveAfterError() error
func (r *journalRepository) Close() {
	r.dbMutex.Lock()
	defer r.dbMutex.Unlock()
	if r.db == nil {
		return
	}
	r.db.Close()
	r.db = nil
}

func (r *journalRepository) CreateTables() error {
	query := `
		CREATE TABLE IF NOT EXISTS journal (
			id BIGINT PRIMARY KEY AUTOINCREMENT,
			journal_file TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			hostname TEXT NOT NULL,
			unit TEXT NOT NULL,
			message TEXT NOT NULL,
			priority TEXT NOT NULL,
			syslog_pid TEXT NOT NULL,
			syslog_ident TEXT NOT NULL,
			fields TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS journal_timestamp_idx ON journal (timestamp);
		CREATE INDEX IF NOT EXISTS journal_priority_idx ON journal (priority);
	`

	_, err := r.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (r *journalRepository) GetEntries(limit int, offset int) ([]JournalEntry, error)
func (r *journalRepository) InsertEntries(entries []JournalEntry) error
