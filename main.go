package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/rfyiamcool/pgcacher/pkg/psutils"
	pcstat "github.com/tobert/pcstat/pkg"
)

var (
	pidFlag, topFlag, workerFlag                int
	terseFlag, nohdrFlag, jsonFlag, unicodeFlag bool
	plainFlag, ppsFlag, bnameFlag               bool
)

func init() {
	flag.IntVar(&pidFlag, "pid", 0, "show all open maps for the given pid")
	flag.IntVar(&topFlag, "top", 0, "show top x cached files in descending order")
	flag.IntVar(&workerFlag, "worker", 2, "concurrency workers")
	flag.BoolVar(&terseFlag, "terse", false, "show terse output")
	flag.BoolVar(&nohdrFlag, "nohdr", false, "omit the header from terse & text output")
	flag.BoolVar(&jsonFlag, "json", false, "return data in JSON format")
	flag.BoolVar(&unicodeFlag, "unicode", false, "return data with unicode box characters")
	flag.BoolVar(&plainFlag, "plain", false, "return data with no box characters")
	flag.BoolVar(&ppsFlag, "pps", false, "include the per-page status in JSON output")
	flag.BoolVar(&bnameFlag, "bname", false, "convert paths to basename to narrow the output")
}

func main() {
	flag.Parse()

	files := flag.Args()
	pg := pgcacher{files: files}

	if topFlag != 0 {
		pg.handleTop(topFlag)
		os.Exit(0)
	}

	if pidFlag != 0 {
		pg.appendProcessFiles(pidFlag)
	}

	if len(files) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	pg.filterFiles()
	stats := pg.getStatsFromFiles()
	pg.output(stats)
}

type pgcacher struct {
	files []string
}

func (pg *pgcacher) filterFiles() {
	sset := make(map[string]struct{}, len(pg.files))
	for _, file := range pg.files {
		file = strings.Trim(file, " ")
		sset[file] = struct{}{}
	}

	// remove duplication.
	dups := make([]string, len(sset))
	for fname := range sset {
		dups = append(dups, fname)
	}
	pg.files = dups
}

func (pg *pgcacher) appendProcessFiles(pid int) {
	pg.files = append(pg.files, pg.getProcessFiles(pid)...)
}

func (pg *pgcacher) getProcessFiles(pid int) []string {
	// switch mount namespace for container.
	pcstat.SwitchMountNs(pidFlag)

	// get files of `/proc/{pid}/fd` and `/proc/{pid}/maps`
	processFiles := pg.getProcessFdFiles(pidFlag)
	processMapFiles := pg.getProcessFdFiles(pidFlag)

	// append
	var files []string
	files = append(files, processFiles...)
	files = append(files, processMapFiles...)
	return files
}

func (pg *pgcacher) getProcessMaps(pid int) []string {
	fname := fmt.Sprintf("/proc/%d/maps", pid)

	f, err := os.Open(fname)
	if err != nil {
		log.Printf("could not read dir %s, err: %s", fname, err.Error())
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	out := make([]string, 0, 20)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 6 && strings.HasPrefix(parts[5], "/") {
			// found something that looks like a file
			out = append(out, parts[5])
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("reading '%s' failed: %s", fname, err)
	}

	return out
}

func (pg *pgcacher) getProcessFdFiles(pid int) []string {
	dpath := fmt.Sprintf("/proc/%d/fd", pid)

	files, err := os.ReadDir(dpath)
	if err != nil {
		log.Printf("could not read dir %s, err: %s", dpath, err.Error())
		return nil
	}

	var (
		out = make([]string, 0, len(files))
		mu  = sync.Mutex{}
	)

	readlink := func(file fs.DirEntry) {
		fpath := fmt.Sprintf("%s/%s", dpath, file.Name())
		target, err := os.Readlink(fpath)
		if !strings.HasPrefix(target, "/") { // ignore socket or pipe.
			return
		}
		if strings.HasPrefix(target, "/dev") { // ignore devices
			return
		}

		if err != nil {
			log.Printf("can not read link '%s', err: %v\n", fpath, err.Error())
			return
		}

		mu.Lock()
		out = append(out, target)
		mu.Unlock()
	}

	// fill files to channel.
	queue := make(chan fs.DirEntry, len(files))
	for _, file := range files {
		queue <- file
	}
	close(queue)

	// handle files concurrently.
	wg := sync.WaitGroup{}
	for i := 0; i < workerFlag; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for file := range queue {
				readlink(file)
			}
		}()
	}
	wg.Wait()

	return out
}

func (pg *pgcacher) getStatsFromFiles() PcStatusList {
	var (
		mu = sync.Mutex{}
		wg = sync.WaitGroup{}

		stats = make(PcStatusList, 0, len(pg.files))
	)

	// fill files to queue.
	queue := make(chan string, len(pg.files))
	for _, fname := range pg.files {
		queue <- fname
	}
	close(queue)

	analyse := func(fname string) {
		status, err := pcstat.GetPcStatus(fname)
		if err != nil {
			log.Printf("skipping %q: %v", fname, err)
			return
		}

		// only get filename, trim full dir path of the file.
		if bnameFlag {
			status.Name = path.Base(fname)
		}

		mu.Lock()
		stats = append(stats, status)
		mu.Unlock()
	}

	// analyse page cache stats of files concurrently.
	for i := 0; i < workerFlag; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for fname := range queue {
				analyse(fname)
			}
		}()
	}
	wg.Wait()

	sort.Sort(PcStatusList(stats))
	return stats
}

func (pg *pgcacher) output(stats PcStatusList) {
	if jsonFlag {
		stats.FormatJson(!ppsFlag)
	} else if terseFlag {
		stats.FormatTerse()
	} else if unicodeFlag {
		stats.FormatUnicode()
	} else if plainFlag {
		stats.FormatPlain()
	} else {
		stats.FormatText()
	}
}

func (pg *pgcacher) handleTop(top int) {
	// get all active process.
	procs, err := psutils.Processes()
	if err != nil || len(procs) == 0 {
		log.Fatalf("failed to get processes, err: %v", err)
	}

	ps := make([]psutils.Process, 0, 50)
	for _, proc := range procs {
		if proc.RSS() != 0 {
			ps = append(ps, proc)
		}
	}

	// append open fd of each process.
	for _, process := range ps {
		pg.appendProcessFiles(process.Pid())
	}

	pg.filterFiles()
	stats := pg.getStatsFromFiles()

	top = min(len(stats), top)
	pg.output(stats[:top])
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
