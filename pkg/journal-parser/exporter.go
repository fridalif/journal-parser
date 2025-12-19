package journalparser

import (
	"encoding/json"
	"fmt"
	"os"
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

	return nil
}

func (e *exporter) ExportToJSON(entries []JournalEntry, maxTimestamp string, minTimestamp string) error {
	filename := fmt.Sprintf("%s/%s-%s.%d.json", e.OutputDirectory, minTimestamp, maxTimestamp, time.Now().UnixNano())
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
	return nil
}
