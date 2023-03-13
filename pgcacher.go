package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/rfyiamcool/pgcacher/pkg/pcstats"
	"github.com/rfyiamcool/pgcacher/pkg/psutils"
)

type emptyNull struct{}

type pgcacher struct {
	files     []string
	leastSize int64
	option    *option
}

func (pg *pgcacher) ignoreFile(file string) bool {
	if pg.option.excludeFiles != "" && wildcardMatch(file, pg.option.excludeFiles) {
		return true
	}

	if pg.option.includeFiles != "" && !wildcardMatch(file, pg.option.includeFiles) {
		return true
	}

	return false
}

func (pg *pgcacher) filterFiles() {
	sset := make(map[string]emptyNull, len(pg.files))
	for _, file := range pg.files {
		file = strings.Trim(file, " ")
		if pg.ignoreFile(file) {
			continue
		}
		sset[file] = emptyNull{}
	}

	// remove duplication.
	dups := make([]string, 0, len(sset))
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
	pcstats.SwitchMountNs(pg.option.pid)

	// get files of `/proc/{pid}/fd` and `/proc/{pid}/maps`
	processFiles := pg.getProcessFdFiles(pid)
	processMapFiles := pg.getProcessMaps(pid)

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
		if pg.ignoreFile(target) {
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
	for i := 0; i < pg.option.worker; i++ {
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

var errLessThanSize = errors.New("the file size is less than the leastSize")

func (pg *pgcacher) getPageCacheStats() PcStatusList {
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

	ignoreFunc := func(file *os.File) error {
		fs, err := file.Stat()
		if err != nil {
			return err
		}
		if pg.leastSize != 0 && fs.Size() < pg.leastSize {
			return errLessThanSize
		}
		return nil
	}

	analyse := func(fname string) {
		status, err := pcstats.GetPcStatus(fname, ignoreFunc)
		if err == errLessThanSize {
			return
		}
		if err != nil {
			log.Printf("skipping %q: %v", fname, err)
			return
		}

		// only get filename, trim full dir path of the file.
		if pg.option.bname {
			status.Name = path.Base(fname)
		}

		// append
		mu.Lock()
		stats = append(stats, status)
		mu.Unlock()
	}

	// analyse page cache stats of files concurrently.
	for i := 0; i < pg.option.worker; i++ {
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
	if pg.option.json {
		stats.FormatJson()
	} else if pg.option.terse {
		stats.FormatTerse()
	} else if pg.option.unicode {
		stats.FormatUnicode()
	} else if pg.option.plain {
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
		if proc.RSS() == 0 {
			continue
		}

		ps = append(ps, proc)
	}

	var (
		wg    = sync.WaitGroup{}
		mu    = sync.Mutex{}
		queue = make(chan psutils.Process, len(ps))
	)

	for _, process := range ps {
		queue <- process
	}
	close(queue)

	// append open fd of each process.
	for i := 0; i < pg.option.worker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for process := range queue {
				files := pg.getProcessFiles(process.Pid())

				mu.Lock()
				pg.files = append(pg.files, files...)
				mu.Unlock()
			}

		}()
	}
	wg.Wait()

	// filter files
	pg.filterFiles()

	stats := pg.getPageCacheStats()
	top = min(len(stats), top)
	pg.output(stats[:top])
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func wildcardMatch(s string, p string) bool {
	if strings.Contains(s, p) {
		return true
	}

	runeInput := []rune(s)
	runePattern := []rune(p)

	lenInput := len(runeInput)
	lenPattern := len(runePattern)

	isMatchingMatrix := make([][]bool, lenInput+1)

	for i := range isMatchingMatrix {
		isMatchingMatrix[i] = make([]bool, lenPattern+1)
	}

	isMatchingMatrix[0][0] = true
	for i := 1; i < lenInput; i++ {
		isMatchingMatrix[i][0] = false
	}

	if lenPattern > 0 {
		if runePattern[0] == '*' {
			isMatchingMatrix[0][1] = true
		}
	}

	for j := 2; j <= lenPattern; j++ {
		if runePattern[j-1] == '*' {
			isMatchingMatrix[0][j] = isMatchingMatrix[0][j-1]
		}
	}

	for i := 1; i <= lenInput; i++ {
		for j := 1; j <= lenPattern; j++ {

			if runePattern[j-1] == '*' {
				isMatchingMatrix[i][j] = isMatchingMatrix[i-1][j] || isMatchingMatrix[i][j-1]
			}

			if runePattern[j-1] == '?' || runeInput[i-1] == runePattern[j-1] {
				isMatchingMatrix[i][j] = isMatchingMatrix[i-1][j-1]
			}
		}
	}

	return isMatchingMatrix[lenInput][lenPattern]
}
