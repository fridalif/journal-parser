package journalparser

import (
	"encoding/json"
	"time"
)

type JournalEntry struct {
	JournalFile string
	Timestamp   time.Time
	Hostname    string
	Unit        string
	Message     string
	Priority    string
	SyslogPID   string
	SyslogIdent string
	Fields      map[string]string
}

func (je JournalEntry) ToJournalEntryFromDB() (JournalEntryFromDB, error) {
	fieldsBytes, err := json.Marshal(je.Fields)
	if err != nil {
		return JournalEntryFromDB{}, err
	}
	return JournalEntryFromDB{
		JournalFile: je.JournalFile,
		Timestamp:   je.Timestamp,
		Hostname:    je.Hostname,
		Unit:        je.Unit,
		Message:     je.Message,
		Priority:    je.Priority,
		SyslogPID:   je.SyslogPID,
		SyslogIdent: je.SyslogIdent,
		Fields:      string(fieldsBytes),
	}, nil
}

type JournalEntryFromDB struct {
	ID          int
	JournalFile string
	Timestamp   time.Time
	Hostname    string
	Unit        string
	Message     string
	Priority    string
	SyslogPID   string
	SyslogIdent string
	Fields      string
}

func (jfd *JournalEntryFromDB) ToJournalEntry() (JournalEntry, error) {
	var fields map[string]string
	err := json.Unmarshal([]byte(jfd.Fields), &fields)
	if err != nil {
		return JournalEntry{}, err
	}
	return JournalEntry{
		JournalFile: jfd.JournalFile,
		Timestamp:   jfd.Timestamp,
		Hostname:    jfd.Hostname,
		Unit:        jfd.Unit,
		Message:     jfd.Message,
		Priority:    jfd.Priority,
		SyslogPID:   jfd.SyslogPID,
		SyslogIdent: jfd.SyslogIdent,
		Fields:      fields,
	}, nil
}
