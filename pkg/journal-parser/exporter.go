package journalparser

type ExporterI interface {
	Export(entries []JournalEntry) error
}

type exporter struct {
	CSV             bool
	JSON            bool
	CLI             bool
	OutputDirectory string
}

func NewExporter(csv bool, json bool, cli bool, output string) ExporterI {
	return &exporter{
		CSV:             csv,
		JSON:            json,
		CLI:             cli,
		OutputDirectory: output,
	}
}

func (e *exporter) Export(entries []JournalEntry) error {
	return nil
}
