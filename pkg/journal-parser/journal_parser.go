package journalparser

import (
	"fmt"
	"os"
	"sync"
)

type JournalParser struct {
	Target           string
	Partition        int
	DirectoryQueue   []string
	DirectoryQueueMu *sync.Mutex
	FileQueueMu      *sync.Mutex
	FileQueue        []string
	WG               *sync.WaitGroup
	OutputDirectory  string
}

func NewJournalParser(target string, partition int, output string) *JournalParser {
	return &JournalParser{
		Target:           target,
		Partition:        partition,
		DirectoryQueue:   []string{},
		FileQueue:        []string{},
		OutputDirectory:  output,
		DirectoryQueueMu: new(sync.Mutex),
		FileQueueMu:      new(sync.Mutex),
		WG:               new(sync.WaitGroup),
	}
}

func (jp *JournalParser) createOutputDirectoryAndDatabaseFile() error {
	err := os.Mkdir("./"+jp.OutputDirectory, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	err = os.WriteFile("./"+jp.OutputDirectory+"/database.db", []byte(""), 0644)
	if err != nil {
		return fmt.Errorf("failed to create database file: %v", err)
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
		jp.DirectoryQueueMu.Lock()
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
		jp.DirectoryQueueMu.Unlock()
		jp.WG.Add(1)
		go func() {
			defer jp.WG.Done()

			err := jp.ParseDirectory(dirName)
			if err != nil {
				fmt.Println("Error: ", err)
			}
		}()
	}
	jp.WG.Wait()

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
	return nil
}

func (jp *JournalParser) ParseFile(filename string) error {
	return nil
}
