package journalparser

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
	_ "github.com/mattn/go-sqlite3"
)

type ExportSettings struct {
	CSV  bool
	JSON bool
	CLI  bool
}

type JournalParser struct {
	Target          string
	Partition       int
	DirectoryQueue  []string
	FileQueue       []string
	OutputDirectory string
	repo            JournalRepositoryI
	Exporter        ExporterI
}

func NewJournalParser(target string, partition int, output string, exporter ExporterI, repo JournalRepositoryI) *JournalParser {
	return &JournalParser{
		Target:          target,
		Partition:       partition,
		DirectoryQueue:  []string{},
		FileQueue:       []string{},
		OutputDirectory: output,
		repo:            repo,
		Exporter:        exporter,
	}
}

func (jp *JournalParser) Parse() {
	isDir, err := jp.isDirectory(jp.Target)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	fmt.Println("Successfully create output directory: ", jp.OutputDirectory)
	if isDir {
		jp.DirectoryQueue = append(jp.DirectoryQueue, jp.Target)
	} else {
		jp.FileQueue = append(jp.FileQueue, jp.Target)
	}

	// Parse directories
	for {
		dirQueueLen := len(jp.DirectoryQueue)
		if dirQueueLen == 0 {
			break
		}
		dirName := jp.DirectoryQueue[0]
		if dirQueueLen > 1 {
			jp.DirectoryQueue = jp.DirectoryQueue[1:]
		} else {
			jp.DirectoryQueue = []string{}
		}

		err := jp.ParseDirectory(dirName)
		if err != nil {
			fmt.Println("Error: ", err)
		}
	}

	// Parse files
	for {
		fileQueueLen := len(jp.FileQueue)
		if fileQueueLen == 0 {
			break
		}
		fileName := jp.FileQueue[0]
		if fileQueueLen > 1 {
			jp.FileQueue = jp.FileQueue[1:]
		} else {
			jp.FileQueue = []string{}
		}
		err := jp.ParseFile(fileName)
		if err != nil {
			fmt.Println("Error: ", err)
			err := jp.repo.ConnectToDB(jp.OutputDirectory)
			if err != nil {
				fmt.Println("Error: ", err)
				jp.repo.Close()
				err := jp.repo.ConnectToDB(jp.OutputDirectory)
				if err != nil {
					fmt.Println("Fatal Error Lost connection with Database: ", err)
					return
				}
			}
		}
	}

	fmt.Println("Parsing Done!")
}

func (jp *JournalParser) isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("file '%s' does not exist", path)
		}
		return false, fmt.Errorf("cant check filetype of '%s': %v", path, err)
	}

	if fileInfo.IsDir() {
		return true, nil
	}
	return false, nil
}

func (jp *JournalParser) ParseDirectory(directory string) error {
	fmt.Println("Parsing Directory:", directory)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("failed to parse directory: %v", err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(directory, entry.Name())
		isDir, err := jp.isDirectory(fullPath)
		if err != nil {
			return err
		}

		if isDir {
			jp.DirectoryQueue = append(jp.DirectoryQueue, fullPath)
		} else {
			jp.FileQueue = append(jp.FileQueue, fullPath)
		}
	}
	return nil
}

func (jp *JournalParser) ParseFile(filename string) error {
	fmt.Println("File:", filename)
	journal, err := sdjournal.NewJournalFromFiles(filename)
	if err != nil {
		return fmt.Errorf("failed to open journal: %v", err)
	}
	defer journal.Close()

	err = journal.SeekHead()
	if err != nil {
		return fmt.Errorf("failed to seek head: %v", err)
	}

	entries := []JournalEntry{}

	for {
		n, err := journal.Next()
		if err != nil {
			return fmt.Errorf("failed to read next entry: %v", err)
		}

		if n == 0 {
			break
		}

		entry, err := journal.GetEntry()
		if err != nil {
			return fmt.Errorf("failed to get entry: %v", err)
		}

		journalEntry := JournalEntry{
			Fields:      make(map[string]string),
			JournalFile: filename,
		}

		for k, v := range entry.Fields {
			journalEntry.Fields[k] = v

			switch k {
			case "__REALTIME_TIMESTAMP":
				var usec int64
				fmt.Sscanf(v, "%d", &usec)
				journalEntry.Timestamp = time.Unix(usec/1000000, (usec%1000000)*1000)
			case "_SOURCE_REALTIME_TIMESTAMP":
				var usec int64
				fmt.Sscanf(v, "%d", &usec)
				journalEntry.Timestamp = time.Unix(usec/1000000, (usec%1000000)*1000)
			case "_HOSTNAME":
				journalEntry.Hostname = v
			case "_SYSTEMD_UNIT":
				journalEntry.Unit = v
			case "MESSAGE":
				journalEntry.Message = v
			case "PRIORITY":
				journalEntry.Priority = v
			case "SYSLOG_PID":
				journalEntry.SyslogPID = v
			case "SYSLOG_IDENTIFIER":
				journalEntry.SyslogIdent = v
			}
		}

		entries = append(entries, journalEntry)
		if jp.Partition > 0 && len(entries) >= jp.Partition {
			err = jp.repo.InsertEntries(entries)
			if err != nil {
				fmt.Println("Failed insert entry to database: ", err.Error())
			}
			entries = []JournalEntry{}
		}
	}
	err = jp.repo.InsertEntries(entries)
	if err != nil {
		fmt.Println("Failed insert entry to database: ", err.Error())
		return err
	}
	return nil
}
