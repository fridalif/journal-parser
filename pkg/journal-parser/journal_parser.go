package journalparser

type JournalParser struct {
	Target    string
	Partition int
}

func NewJournalParser(target string, partition int) *JournalParser {
	return &JournalParser{
		Target:    target,
		Partition: partition,
	}
}
