package journalparser

import (
	"context"
	"fmt"
	parser "journal-parser/pkg/velocidex_parser"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
	ntfs_parser "www.velocidex.com/golang/go-ntfs/parser"

	"github.com/Velocidex/ordereddict"
	"github.com/schollz/progressbar/v3"
)

type ExportSettings struct {
	CSV             bool
	JSON            bool
	CLI             bool
	OutputDirectory string
}

type JournalParser struct {
	Targets         []string
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
	skipParsing     bool
	oldDatabase     string
	filesOffset     int
	filesLimit      int
}

func NewJournalParser(targets []string, partition int, output string, exporter ExporterI, repo JournalRepositoryI, skipParsing bool, oldDatabase string, filesOffset int, filesLimit int) *JournalParser {
	return &JournalParser{
		Targets:         targets,
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
		skipParsing:     skipParsing,
		oldDatabase:     oldDatabase,
		filesOffset:     filesOffset,
		filesLimit:      filesLimit,
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

	for _, target := range jp.Targets {
		if jp.skipParsing {
			break
		}
		isDir, err := jp.isDirectory(target)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		fmt.Println("Successfully create output directory: ", jp.OutputDirectory)
		if isDir {
			jp.DirectoryQueue = append(jp.DirectoryQueue, target)
		} else {
			jp.FileQueue = append(jp.FileQueue, target)
		}
	}

	// Parse directories
	for !jp.skipParsing {
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

	// Skipping files before offset
	for i := 0; i < jp.filesOffset; i++ {
		jp.FileQueueMutex.Lock()
		fileQueueLen := len(jp.FileQueue)
		if fileQueueLen == 0 {
			jp.FileQueueMutex.Unlock()
			break
		}
		jp.FileQueue = jp.FileQueue[1:]
		jp.FileQueueMutex.Unlock()
		parsingBar.Add(1)
	}

	if jp.filesLimit != 0 && len(jp.FileQueue) > jp.filesLimit {
		jp.FileQueue = jp.FileQueue[:jp.filesLimit]
	}

	// Parse files
	for !jp.skipParsing {
		jp.FileQueueMutex.Lock()
		fileQueueLen := len(jp.FileQueue)
		if fileQueueLen == 0 {
			jp.FileQueueMutex.Unlock()
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
					err := jp.repo.ConnectToDB(jp.OutputDirectory, jp.oldDatabase)
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

func (jp *JournalParser) parseLogItems(items []ordereddict.Item, journalFile string) *JournalEntry {
	entry := &JournalEntry{
		JournalFile: journalFile,
		Fields:      make(map[string]string),
	}
	for _, item := range items {
		switch item.Value.(type) {
		case string:
			entry.Fields[item.Key] = item.Value.(string)
		case *ordereddict.Dict:
			newEntry := jp.parseLogItems(item.Value.(*ordereddict.Dict).Items(), journalFile)
			for key, value := range newEntry.Fields {
				entry.Fields[key] = value
			}
		case int64:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case uint64:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case uint32:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case uint16:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case uint8:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case int32:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case int16:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case int8:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case int:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case uint:
			entry.Fields[item.Key] = fmt.Sprintf("%d", item.Value)
		case time.Time:
			entry.Fields[item.Key] = item.Value.(time.Time).Format(time.RFC3339)
		default:
			entry.Fields[item.Key] = fmt.Sprintf("%v", item.Value)
		}
	}
	for k, v := range entry.Fields {
		switch k {
		case "Timestamp":
			entry.Timestamp, _ = time.Parse(time.RFC3339, v)
		case "_HOSTNAME":
			entry.Hostname = v
		case "_SYSTEMD_UNIT":
			entry.Unit = v
		case "_PID":
			if entry.SyslogPID == "" {
				entry.SyslogPID = v
			}
		case "MESSAGE":
			entry.Message = v
		case "PRIORITY":
			entry.Priority = v
		case "SYSLOG_PID":
			entry.SyslogPID = v
		case "SYSLOG_IDENTIFIER":
			entry.SyslogIdent = v
		}
	}
	return entry
}

func (jp *JournalParser) ParseFile(filename string) error {
	fd, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fd.Close()
	reader, _ := ntfs_parser.NewPagedReader(fd, 1024, 10000)
	journal, err := parser.OpenFile(reader)
	if err != nil {
		return nil
	}
	for log := range journal.GetLogs(context.Background()) {
		entry := jp.parseLogItems(log.Items(), filename)
		jp.EntriesChan <- *entry
		jp.entriesCounter.Add(1)
	}
	return nil
}
