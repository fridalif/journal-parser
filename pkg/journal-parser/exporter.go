package journalparser

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type ExporterI interface {
	Export(entries []JournalEntry) error
}

type exporter struct {
	CSV             bool
	JSON            bool
	CLI             bool
	OutputDirectory string
	wg              *sync.WaitGroup
}

func NewExporter(csv bool, json bool, cli bool, output string) ExporterI {
	return &exporter{
		CSV:             csv,
		JSON:            json,
		CLI:             cli,
		OutputDirectory: output,
		wg:              new(sync.WaitGroup),
	}
}

func (e *exporter) Export(entries []JournalEntry) error {
	errorAccured := false
	maxTimestamp := time.Time{}
	minTimestamp := time.Time{}
	if len(entries) > 0 {
		maxTimestamp = entries[len(entries)-1].Timestamp
		minTimestamp = entries[0].Timestamp
	}
	maxTimestampStr := maxTimestamp.Format(time.RFC3339)
	minTimestampStr := minTimestamp.Format(time.RFC3339)
	if e.CSV {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			err := e.ExportToCSV(entries, maxTimestampStr, minTimestampStr)
			if err != nil {
				fmt.Println("Error exporting in CSV: ", err)
				errorAccured = true
			}
		}()
	}

	if e.JSON {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			err := e.ExportToJSON(entries, maxTimestampStr, minTimestampStr)
			if err != nil {
				fmt.Println("Error exporting in JSON: ", err)
				errorAccured = true
			}
		}()
	}

	if e.CLI {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			err := e.ExportToCLI(entries)
			if err != nil {
				fmt.Println("Error exporting in CLI: ", err)
				errorAccured = true
			}
		}()
	}

	e.wg.Wait()
	if errorAccured {
		return fmt.Errorf("error exporting")
	}
	return nil
}

func (e *exporter) ExportToCSV(entries []JournalEntry, maxTimestamp string, minTimestamp string) error {
	filename := path.Join(e.OutputDirectory, fmt.Sprintf("%s_%s.%d.csv",
		strings.ReplaceAll(minTimestamp, ":", "-"),
		strings.ReplaceAll(maxTimestamp, ":", "-"),
		time.Now().UnixNano()),
	)

	fd, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fd.Close()

	writer := csv.NewWriter(fd)
	defer writer.Flush()

	headers := []string{
		"JournalFile",
		"Timestamp",
		"Hostname",
		"Unit",
		"Message",
		"Priority",
		"SyslogPID",
		"SyslogIdent",
		"FullInfo",
	}

	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, entry := range entries {
		record := make([]string, 0, len(headers))
		fieldsBytes, err := json.Marshal(entry.Fields)
		if err != nil {
			fieldsBytes = []byte(fmt.Sprintf("%v", entry.Fields))
			continue
		}
		record = append(record,
			entry.JournalFile,
			entry.Timestamp.Format(time.RFC3339),
			entry.Hostname,
			entry.Unit,
			e.escapeCSVField(entry.Message),
			entry.Priority,
			entry.SyslogPID,
			entry.SyslogIdent,
			e.escapeCSVField(string(fieldsBytes)),
		)

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func (e *exporter) escapeCSVField(value string) string {
	if strings.ContainsAny(value, ",\"\n\r") {
		escaped := strings.ReplaceAll(value, "\"", "\"\"")
		return "\"" + escaped + "\""
	}
	return value
}

func (e *exporter) ExportToJSON(entries []JournalEntry, maxTimestamp string, minTimestamp string) error {
	filename := path.Join(e.OutputDirectory, fmt.Sprintf("%s_%s.%d.json",
		strings.ReplaceAll(minTimestamp, ":", "-"),
		strings.ReplaceAll(maxTimestamp, ":", "-"),
		time.Now().UnixNano()),
	)
	fd, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fd.Close()
	encoder := json.NewEncoder(fd)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entries)
}

func (e *exporter) ExportToCLI(entries []JournalEntry) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	for _, entry := range entries {
		err := encoder.Encode(entry)
		if err != nil {
			continue
		}
	}
	return nil
}
