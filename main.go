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

var (
	pidFlag, topFlag, workerFlag                      int
	terseFlag, nohdrFlag, jsonFlag, unicodeFlag       bool
	plainFlag, ppsFlag, bnameFlag                     bool
	leastSizeFlag, excludeFilesFlag, includeFilesFlag string
)

func init() {
	// basic params
	flag.IntVar(&pidFlag, "pid", 0, "show all open maps for the given pid")
	flag.IntVar(&topFlag, "top", 0, "show top x cached files in descending order")
	flag.IntVar(&workerFlag, "worker", 2, "concurrency workers")
	flag.StringVar(&leastSizeFlag, "least-size", "0mb", "ignore files smaller than the lastSize, such as 10MB and 15GB")
	flag.StringVar(&excludeFilesFlag, "exclude-files", "", "exclude the specified files by wildcard, such as 'a*c?d' and '*xiaorui*,rfyiamcool'")
	flag.StringVar(&includeFilesFlag, "include-files", "", "only include the specified files by wildcard, such as 'a*c?d' and '*xiaorui?cc,rfyiamcool'")

	// show params
	flag.BoolVar(&terseFlag, "terse", false, "show terse output")
	flag.BoolVar(&nohdrFlag, "nohdr", false, "omit the header from terse & text output")
	flag.BoolVar(&jsonFlag, "json", false, "return data in JSON format")
	flag.BoolVar(&unicodeFlag, "unicode", false, "return data with unicode box characters")
	flag.BoolVar(&plainFlag, "plain", false, "return data with no box characters")
	flag.BoolVar(&ppsFlag, "pps", false, "include the per-page status in JSON output")
	flag.BoolVar(&bnameFlag, "bname", false, "convert paths to basename to narrow the output")
}

func main() {
	// prepare phase
	flag.Parse()
	if runtime.GOOS != "linux" {
		log.Fatalf("pgcacher only support running on Linux !!!")
	}
	leastSize, _ := humanize.ParseBytes(leastSizeFlag)

	// running phase
	files := flag.Args()
	pg := pgcacher{files: files, leastSize: int64(leastSize)}

	if topFlag != 0 {
		pg.handleTop(topFlag)
		os.Exit(0)
	}

	if pidFlag != 0 {
		pg.appendProcessFiles(pidFlag)
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
	defer func() {
		pcstat.SwitchMountNs(os.Getegid())
	}()
}
