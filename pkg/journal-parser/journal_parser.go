package journalparser

import (
	"context"
	"fmt"
	parser "journal-parser/pkg/velocidex_parser"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	ntfs_parser "www.velocidex.com/golang/go-ntfs/parser"

	"github.com/schollz/progressbar/v3"
)

type ExportSettings struct {
	CSV  bool
	JSON bool
	CLI  bool
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
}

func NewJournalParser(targets []string, partition int, output string, exporter ExporterI, repo JournalRepositoryI) *JournalParser {
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
	fd, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fd.Close()
	reader, _ := ntfs_parser.NewPagedReader(fd, 1024, 10000)
	journal, err := parser.OpenFile(reader)
	if err != nil {
		panic(err)
	}
	for log := range journal.GetLogs(context.Background()) {
		var entry JournalEntry
		entry.JournalFile = filename
	}
	return nil
}
