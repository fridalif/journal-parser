package journalparser

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
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
	FileQueueMutex  *sync.Mutex
	wg              *sync.WaitGroup
	EntriesChan     chan JournalEntry
	OutputDirectory string
	repo            JournalRepositoryI
	Exporter        ExporterI
	entriesCounter  atomic.Int32
	writerWG        *sync.WaitGroup
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
		entriesCounter:  atomic.Int32{},
		EntriesChan:     make(chan JournalEntry, repo.GetMaxBatchSize()),
		FileQueueMutex:  new(sync.Mutex),
		wg:              new(sync.WaitGroup),
		writerWG:        new(sync.WaitGroup),
	}
}

func (jp *JournalParser) WriterToDB() {
	defer jp.writerWG.Done()
	entries := make([]JournalEntry, 0)
	if jp.Partition > 0 {
		entries = make([]JournalEntry, 0, jp.Partition)
	}
	for {
		entry, ok := <-jp.EntriesChan
		if !ok {
			break
		}

		entries = append(entries, entry)
		if len(entries) >= jp.Partition && jp.Partition > 0 {
			err := jp.repo.InsertEntries(entries)
			if err != nil {
				fmt.Println("Error: ", err)
			}
			entries = entries[:0]
		}
	}
	if len(entries) > 0 {
		err := jp.repo.InsertEntries(entries)
		if err != nil {
			fmt.Println("Error: ", err)
		}
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

	jp.writerWG.Add(1)
	go jp.WriterToDB()
	fileQueueStartLen := len(jp.FileQueue)
	parsingBar := progressbar.Default(int64(fileQueueStartLen), "Parsing Files...")

	// Parse files
	for {
		jp.FileQueueMutex.Lock()
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
		jp.FileQueueMutex.Unlock()
		jp.wg.Add(1)
		go func() {
			defer jp.wg.Done()
			err := jp.ParseFile(fileName)
			parsingBar.Add(1)
			if err != nil {
				fmt.Println("Error: ", err)
				err := jp.repo.CheckAliveAfterError()
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
		}()
	}

	jp.wg.Wait()
	fmt.Println("Parsing Done. Waiting for Inserting to Database...")
	close(jp.EntriesChan)
	jp.writerWG.Wait()

	fmt.Println("Starting Export...")

	if jp.Partition == 0 {
		entries, err := jp.repo.GetEntriesUnlimited()
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		err = jp.Exporter.Export(entries)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		fmt.Println("Export Done!")
		return
	}

	exportBar := progressbar.Default(int64(jp.entriesCounter.Load()), "Exporting...")

	for i := 0; i < int(jp.entriesCounter.Load()); i += jp.Partition {
		entries, err := jp.repo.GetEntriesLimited(jp.Partition, i)
		if err != nil {
			fmt.Println("Error: ", err)
			err = jp.repo.CheckAliveAfterError()
			if err != nil {
				fmt.Println("Fatal Error Lost connection with Database: ", err)
				return
			}
			continue
		}

		err = jp.Exporter.Export(entries)
		exportBar.Add(jp.Partition)
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
	}
	fmt.Println("Export Done!")
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
	journal, err := sdjournal.NewJournalFromFiles(filename)
	if err != nil {
		return fmt.Errorf("failed to open journal: %v", err)
	}
	defer journal.Close()

	err = journal.SeekHead()
	if err != nil {
		return fmt.Errorf("failed to seek head: %v", err)
	}

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
		jp.entriesCounter.Add(1)
		jp.EntriesChan <- journalEntry

	}
	return nil
}
