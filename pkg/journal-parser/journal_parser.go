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
}

func NewJournalParser(target string, partition int) *JournalParser {
	return &JournalParser{
		Target:           target,
		Partition:        partition,
		DirectoryQueue:   []string{},
		FileQueue:        []string{},
		DirectoryQueueMu: new(sync.Mutex),
		FileQueueMu:      new(sync.Mutex),
		WG:               new(sync.WaitGroup),
	}
}

func (jp *JournalParser) Parse() {
	isDir, err := jp.isDirectory(jp.Target)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	if isDir {
		jp.DirectoryQueue = append(jp.DirectoryQueue, jp.Target)
	} else {
		jp.FileQueue = append(jp.FileQueue, jp.Target)
	}

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
