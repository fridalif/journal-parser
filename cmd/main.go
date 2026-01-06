package main

import (
	"fmt"
	journalparser "journal-parser/pkg/journal-parser"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func bannerOutput() {
	fmt.Println(`------------------------------------------------------------------------------------------------`)
	fmt.Println(`      @                                              @@@@@@                                     `)
	fmt.Println(`      @  @@@@  @    @ @@@@@  @    @   @@   @         @     @   @@   @@@@@   @@@@  @@@@@@ @@@@@  `)
	fmt.Println(`      @ @    @ @    @ @    @ @@   @  @  @  @         @     @  @  @  @    @ @      @      @    @ `)
	fmt.Println(`      @ @    @ @    @ @    @ @ @  @ @    @ @         @@@@@@  @    @ @    @  @@@@  @@@@@  @    @ `)
	fmt.Println(`@     @ @    @ @    @ @@@@@  @  @ @ @@@@@@ @         @       @@@@@@ @@@@@       @ @      @@@@@  `)
	fmt.Println(`@     @ @    @ @    @ @   @  @   @@ @    @ @         @       @    @ @   @  @    @ @      @   @  `)
	fmt.Println(` @@@@@   @@@@   @@@@  @    @ @    @ @    @ @@@@@@    @       @    @ @    @  @@@@  @@@@@@ @    @ `)
	fmt.Println(`------------------------------------------------------------------------------------------------`)
}

func printHelpMessage() {
	fmt.Println("Usage: journalparser [FLAGS]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  	-h, --help    Display this help message")
	fmt.Println("  	-t, --target <Directory or filename>  The target to parse (this argument can be repeated)")
	fmt.Println("  	-p, --partition <Number>  Max strings per file (min 0, no partition by default)")
	fmt.Println("	-mb, --max-batch <Number> Max number of butch for SQL insert (max 1000, min 1, default 1000)")
	fmt.Println("	-o, --output <Directory> Directory for output")
	fmt.Println("  	--csv Export output to csv")
	fmt.Println("	--json Export output to json")
	fmt.Println("	--cli Export output to cli")
	fmt.Println("   -fo, --files-offset <Number> Use this for skip files which you already parsed")
	fmt.Println("   -fl, --files-limit <Number> Use this for limit files which you want parse")
	fmt.Println("   -dbn, --db-name <Path> Name of existing database if needs append data in old DB")
	fmt.Println("   -sp, --skip-parsing - Flag for skipping parsing (only export)")
}

func main() {
	bannerOutput()
	argsLen := len(os.Args)
	targets := []string{}
	partition := 0
	outputDir := ""
	oldDatabase := ""
	skipParsing := false
	exportSettings := journalparser.ExportSettings{
		CSV:  false,
		JSON: false,
		CLI:  false,
	}
	filesOffset := 0
	filesLimit := 0
	maxBatch := 1000

	/*
		Parsing args
	*/
	for i := 0; i < argsLen; i++ {
		if os.Args[i] == "--help" || os.Args[i] == "-h" {
			printHelpMessage()
			return
		}
		if os.Args[i] == "--target" || os.Args[i] == "-t" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --target or -t")
				printHelpMessage()
				return
			}
			if os.Args[i+1] == "" {
				fmt.Println("Error: Missing argument for --target or -t")
				printHelpMessage()
				return
			}
			targets = append(targets, os.Args[i+1])
			i += 1
			continue
		}
		if os.Args[i] == "--skip-parsing" || os.Args[i] == "-sp" {
			skipParsing = true
		}
		if os.Args[i] == "--output" || os.Args[i] == "-o" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --output or -o")
				printHelpMessage()
				return
			}
			if os.Args[i+1] == "" {
				fmt.Println("Error: Missing argument for --output or -o")
				printHelpMessage()
				return
			}
			outputDir = os.Args[i+1]
			i += 1
			continue
		}
		if os.Args[i] == "--db-name" || os.Args[i] == "-dbn" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --db-name or -dbn")
				printHelpMessage()
				return
			}
			oldDatabase = os.Args[i+1]
			i += 1
			continue
		}
		if os.Args[i] == "--max-batch" || os.Args[i] == "-mb" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --max-batch or -mb")
				printHelpMessage()
				return
			}
			flagMaxBatch, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				fmt.Println("Error: Invalid argument for --max-batch or -mb")
				printHelpMessage()
				return
			}
			if flagMaxBatch < 1 || flagMaxBatch > 1000 {
				fmt.Println("Error: Invalid argument for --max-batch or -mb")
				printHelpMessage()
				return
			}
			maxBatch = flagMaxBatch
			i += 1
			continue
		}
		if os.Args[i] == "--files-offset" || os.Args[i] == "-fo" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --files-offset or -fo")
				printHelpMessage()
				return
			}
			flagFilesOffset, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				fmt.Println("Error: Invalid argument for --files-offset or -fo")
				printHelpMessage()
				return
			}
			if filesOffset < 0 {
				fmt.Println("Error: Invalid argument for --files-offset or -fo")
				printHelpMessage()
				return
			}
			filesOffset = flagFilesOffset
			i += 1
			continue
		}
		if os.Args[i] == "--files-limit" || os.Args[i] == "-fl" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --files-limit or -fl")
				printHelpMessage()
				return
			}
			flagFilesLimit, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				fmt.Println("Error: Invalid argument for --files-limit or -fl")
				printHelpMessage()
				return
			}
			if filesOffset <= 0 {
				fmt.Println("Error: Invalid argument for --files-limit or -fl")
				printHelpMessage()
				return
			}
			filesLimit = flagFilesLimit
			i += 1
			continue
		}
		if os.Args[i] == "--partition" || os.Args[i] == "-p" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --partition or -p")
				printHelpMessage()
				return
			}
			flagPartition, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				fmt.Println("Error: Invalid argument for --partition or -p")
				printHelpMessage()
				return
			}
			if flagPartition < 0 {
				fmt.Println("Error: Invalid argument for --partition or -p")
				printHelpMessage()
				return
			}
			partition = flagPartition
			i += 1
			continue
		}
		if os.Args[i] == "--csv" {
			exportSettings.CSV = true
			continue
		}
		if os.Args[i] == "--json" {
			exportSettings.JSON = true
			continue
		}
		if os.Args[i] == "--cli" {
			exportSettings.CLI = true
			continue
		}
	}

	if len(targets) == 0 && !skipParsing {
		fmt.Println("Error: Missing argument for --target or -t")
		printHelpMessage()
		return
	}

	/*
		Show Launch Mode
	*/
	output := time.Now().Format("output_2006-01-02_15-04-05")
	exportSettings.OutputDirectory = path.Join(outputDir, output)
	fmt.Println("Mode")
	fmt.Println("Targets: ", strings.Join(targets, ", "))
	fmt.Println("Partition: ", partition)
	if oldDatabase != "" {
		fmt.Println("OldDatabase: ", oldDatabase)
	}
	fmt.Println("Output: ", exportSettings.OutputDirectory)
	fmt.Println("Files Offset: ", filesOffset)
	fmt.Println("Files Limit: ")
	if filesLimit != 0 {
		fmt.Println(filesLimit)
	} else {
		fmt.Println("unlimited")
	}
	if skipParsing {
		fmt.Println("Skip parsing: true")
	} else {
		fmt.Println("Skip parsing: false")
	}
	fmt.Println("")

	/*
		Start Functionality
	*/
	fmt.Println("Creating Database...")

	repository := journalparser.NewJournalRepository(maxBatch)

	err := os.Mkdir(exportSettings.OutputDirectory, 0755)
	if err != nil {
		fmt.Println("Failed to create output directory: ", err)
		return
	}

	err = repository.ConnectToDB(exportSettings.OutputDirectory, oldDatabase)
	if err != nil {
		fmt.Println("Error connecting to database: ", err)
		return
	}
	defer repository.Close()
	fmt.Println("Database created")
	fmt.Println("Creating SQL Tables...")
	err = repository.CreateTables()
	if err != nil {
		fmt.Println("Error creating SQL tables: ", err)
		return
	}
	fmt.Println("SQL Tables created")
	exporter := journalparser.NewExporter(
		exportSettings.CSV,
		exportSettings.JSON,
		exportSettings.CLI,
		exportSettings.OutputDirectory,
	)
	fmt.Println("Starting parsing...")
	jp := journalparser.NewJournalParser(
		targets,
		partition,
		exportSettings.OutputDirectory,
		exporter,
		repository,
		skipParsing,
		oldDatabase,
		filesOffset,
		filesLimit,
	)
	jp.Parse()
}
