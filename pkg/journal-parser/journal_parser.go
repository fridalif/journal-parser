package journalparser

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type JournalParser struct {
	Target          string
	Partition       int
	DirectoryQueue  []string
	FileQueueMu     *sync.Mutex
	FileQueue       []string
	WG              *sync.WaitGroup
	OutputDirectory string
	DBConn          *sql.DB
	DBMutex         *sync.Mutex
}

func NewJournalParser(target string, partition int, output string) *JournalParser {
	return &JournalParser{
		Target:          target,
		Partition:       partition,
		DirectoryQueue:  []string{},
		FileQueue:       []string{},
		OutputDirectory: output,
		FileQueueMu:     new(sync.Mutex),
		WG:              new(sync.WaitGroup),
		DBConn:          nil,
		DBMutex:         new(sync.Mutex),
	}
}

func (jp *JournalParser) createOutputDirectoryAndDatabaseFile() error {
	err := os.Mkdir("./"+jp.OutputDirectory, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	return nil
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
	err = jp.createOutputDirectoryAndDatabaseFile()
	if err != nil {
		fmt.Println("Error: ", err)
		return
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

	// Creating database
	db, err := sql.Open("sqlite3", "./"+jp.OutputDirectory+"/journal.db")
	if err != nil {
		fmt.Println("Error: error while opening database: ", err)
		return
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Println("Error: error while checking connection to database: ", err)
		return
	}
	jp.DBConn = db

	// Parse files
	for {
		jp.FileQueueMu.Lock()
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
		jp.FileQueueMu.Unlock()
		jp.WG.Add(1)
		go func() {
			defer jp.WG.Done()
			err := jp.ParseFile(fileName)
			if err != nil {
				fmt.Println("Error: ", err)
			}
		}()
	}
	jp.WG.Wait()

	fmt.Println("Done")
}

func (jp *JournalParser) isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("file '%s' does not exist.\n", path)
		}
		return false, fmt.Errorf("cant check filetype of '%s': %v\n", path, err)
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
	return nil
}
