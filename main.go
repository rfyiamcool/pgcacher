package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/dustin/go-humanize"
	pcstat "github.com/tobert/pcstat/pkg"
)

type option struct {
	pid, top, worker                      int
	terse, json, unicode                  bool
	plain, bname                          bool
	leastSize, excludeFiles, includeFiles string
}

var globalOption = new(option)

func init() {
	// basic params
	flag.IntVar(&globalOption.pid, "pid", 0, "show all open maps for the given pid")
	flag.IntVar(&globalOption.top, "top", 0, "scan the open files of all processes, show the top few files that occupy the most memory space in the page cache.")
	flag.IntVar(&globalOption.worker, "worker", 2, "concurrency workers")
	flag.StringVar(&globalOption.leastSize, "least-size", "0mb", "ignore files smaller than the lastSize, such as 10MB and 15GB")
	flag.StringVar(&globalOption.excludeFiles, "exclude-files", "", "exclude the specified files by wildcard, such as 'a*c?d' and '*xiaorui*,rfyiamcool'")
	flag.StringVar(&globalOption.includeFiles, "include-files", "", "only include the specified files by wildcard, such as 'a*c?d' and '*xiaorui?cc,rfyiamcool'")

	// show params
	flag.BoolVar(&globalOption.terse, "terse", false, "show terse output")
	flag.BoolVar(&globalOption.json, "json", false, "return data in JSON format")
	flag.BoolVar(&globalOption.unicode, "unicode", false, "return data with unicode box characters")
	flag.BoolVar(&globalOption.plain, "plain", false, "return data with no box characters")
	flag.BoolVar(&globalOption.bname, "bname", false, "convert paths to basename to narrow the output")
}

func main() {
	// prepare phase
	flag.Parse()
	if runtime.GOOS != "linux" {
		log.Fatalf("pgcacher only support running on Linux !!!")
	}
	leastSize, _ := humanize.ParseBytes(globalOption.leastSize)

	// running phase
	files := flag.Args()
	pg := pgcacher{files: files, leastSize: int64(leastSize), option: globalOption}

	if globalOption.top != 0 {
		pg.handleTop(globalOption.top)
		os.Exit(0)
	}

	if globalOption.pid != 0 {
		pg.appendProcessFiles(globalOption.pid)
	}

	if len(pg.files) == 0 {
		fmt.Println("files is null ?")
		flag.Usage()
		os.Exit(1)
	}

	pg.filterFiles()
	stats := pg.getPageCacheStats()
	pg.output(stats)

	// invalid function, just make a reference relationship with a
	pcstat.SwitchMountNs(os.Getegid())
	pcstat.GetPcStatus(os.Args[0])
}
