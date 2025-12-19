package journalparser

import (
	"fmt"
	"sync"
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

	if e.CSV {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			err := e.ExportToCSV(entries)
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
			err := e.ExportToJSON(entries)
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

func (e *exporter) ExportToCSV(entries []JournalEntry) error {
	return nil
}

func (e *exporter) ExportToJSON(entries []JournalEntry) error {
	return nil
}

func (e *exporter) ExportToCLI(entries []JournalEntry) error {
	return nil
}
