package journalparser

import (
	"database/sql"
	"fmt"
	"sync"
)

type JournalRepositoryI interface {
	ConnectToDB(outputDirectory string) error
	CheckAliveAfterError() error
	Close()
	CreateTables() error
	GetEntriesLimited(limit int, offset int) ([]JournalEntry, error)
	GetEntriesUnlimited() ([]JournalEntry, error)
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

func (r *journalRepository) CheckAliveAfterError() error {
	r.dbMutex.Lock()
	defer r.dbMutex.Unlock()
	if r.db == nil {
		return fmt.Errorf("no connection")
	}
	return r.db.Ping()
}

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
	r.dbMutex.Lock()
	defer r.dbMutex.Unlock()
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

func (r *journalRepository) GetEntriesLimited(limit int, offset int) ([]JournalEntry, error) {
	r.dbMutex.Lock()
	defer r.dbMutex.Unlock()
	query := `
		SELECT journal_file, timestamp, hostname, unit, message, priority, syslog_pid, syslog_ident, fields
		FROM journal
		ORDER BY timestamp ASC
		LIMIT ?
		OFFSET ?
	`
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []JournalEntry
	for rows.Next() {
		var entry JournalEntryFromDB
		err := rows.Scan(&entry.JournalFile, &entry.Timestamp, &entry.Hostname, &entry.Unit, &entry.Message, &entry.Priority, &entry.SyslogPID, &entry.SyslogIdent, &entry.Fields)
		if err != nil {
			return nil, err
		}
		parsedEntry, err := entry.ToJournalEntry()
		if err != nil {
			return nil, err
		}
		entries = append(entries, parsedEntry)
	}
	return entries, nil
}

func (r *journalRepository) GetEntriesUnlimited() ([]JournalEntry, error) {
	r.dbMutex.Lock()
	defer r.dbMutex.Unlock()
	query := `
		SELECT journal_file, timestamp, hostname, unit, message, priority, syslog_pid, syslog_ident, fields
		FROM journal
		ORDER BY timestamp ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []JournalEntry
	for rows.Next() {
		var entry JournalEntryFromDB
		err := rows.Scan(&entry.JournalFile, &entry.Timestamp, &entry.Hostname, &entry.Unit, &entry.Message, &entry.Priority, &entry.SyslogPID, &entry.SyslogIdent, &entry.Fields)
		if err != nil {
			return nil, err
		}
		parsedEntry, err := entry.ToJournalEntry()
		if err != nil {
			return nil, err
		}
		entries = append(entries, parsedEntry)
	}
	return entries, nil
}

func (r *journalRepository) InsertEntries(entries []JournalEntry) error {
	if len(entries) == 0 {
		return nil
	}
	for _, entry := range entries {
		journalEntryForDb, err := entry.ToJournalEntryFromDB()
		if err != nil {
			return err
		}
		r.dbMutex.Lock()
		defer r.dbMutex.Unlock()
		_, err = r.db.Exec("INSERT INTO journal (journal_file, timestamp, hostname, unit, message, priority, syslog_pid, syslog_ident, fields) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			journalEntryForDb.JournalFile, journalEntryForDb.Timestamp, journalEntryForDb.Hostname, journalEntryForDb.Unit, journalEntryForDb.Message, journalEntryForDb.Priority, journalEntryForDb.SyslogPID, journalEntryForDb.SyslogIdent, journalEntryForDb.Fields)
		if err != nil {
			return err
		}
	}
	return nil
}
