package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	pcstat "github.com/tobert/pcstat/pkg"
)

type PcStatusList []pcstat.PcStatus

func (a PcStatusList) Len() int {
	return len(a)
}
func (a PcStatusList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a PcStatusList) Less(i, j int) bool {
	return a[j].Cached < a[i].Cached
}

func (stats PcStatusList) FormatUnicode() {
	maxName := stats.maxNameLen()

	// create horizontal grid line
	pad := strings.Repeat("─", maxName+2)
	top := fmt.Sprintf("┌%s┬────────────────┬─────────────┬────────────────┬─────────────┬─────────┐", pad)
	hr := fmt.Sprintf("├%s┼────────────────┼─────────────┼────────────────┼─────────────┼─────────┤", pad)
	bot := fmt.Sprintf("└%s┴────────────────┴─────────────┴────────────────┴─────────────┴─────────┘", pad)

	var size_sum, page_sum, cached_page_sum, cached_size, cached_size_sum int64

	fmt.Println(top)

	// -nohdr may be chosen to save 2 lines of precious vertical space
	if !nohdrFlag {
		pad = strings.Repeat(" ", maxName-4)
		fmt.Printf("│ Name%s │ Size           │ Pages       │ Cached Size    │ Cached Pages│ Percent │\n", pad)
		fmt.Println(hr)
	}

	for _, pcs := range stats {
		pad = strings.Repeat(" ", maxName-len(pcs.Name))

		// The cache is counted through the page，can't count accurately.
		// Here cached file size is calculated by pcs.Size and pcs.Percent,
		// so it is not completely accurate, but it has reference value.
		cached_size = int64(float64(pcs.Size) * pcs.Percent / 100)

		// %-7.3f was chosen to make it easy to scan the percentages vertically
		// I tried a few different formats only this one kept the decimals aligned
		fmt.Printf("│ %s%s │ %-15s│ %-12d│ %-15s│ %-12d│ %-7.3f │\n",
			pcs.Name, pad, ConvertUnit(pcs.Size), pcs.Pages, ConvertUnit(cached_size), pcs.Cached, pcs.Percent)

		size_sum += pcs.Size
		page_sum += int64(pcs.Pages)
		cached_page_sum += int64(pcs.Cached)
		cached_size_sum += cached_size
	}

	fmt.Println(hr)
	pad = strings.Repeat(" ", maxName-len("Sum"))
	fmt.Printf("│ %s%s │ %-15s│ %-12d│ %-15s│ %-12d│ %-7.3f │\n",
		"Sum", pad, ConvertUnit(size_sum), page_sum, ConvertUnit(cached_size_sum), cached_page_sum, (float64(cached_page_sum)/float64(page_sum))*100.00)
	fmt.Println(bot)
}

func (stats PcStatusList) FormatText() {
	maxName := stats.maxNameLen()

	// create horizontal grid line
	pad := strings.Repeat("-", maxName+2)
	top := fmt.Sprintf("+%s+----------------+-------------+----------------+-------------+---------+", pad)
	hr := fmt.Sprintf("|%s+----------------+-------------+----------------+-------------+---------|", pad)
	bot := fmt.Sprintf("+%s+----------------+-------------+----------------+-------------+---------+", pad)
	var size_sum, page_sum, cached_page_sum, cached_size, cached_size_sum int64

	fmt.Println(top)

	// -nohdr may be chosen to save 2 lines of precious vertical space
	if !nohdrFlag {
		pad = strings.Repeat(" ", maxName-4)
		fmt.Printf("| Name%s | Size           │ Pages       │ Cached Size    │ Cached Pages│ Percent │\n", pad)
		fmt.Println(hr)
	}

	for _, pcs := range stats {
		pad = strings.Repeat(" ", maxName-len(pcs.Name))
		cached_size = int64(float64(pcs.Size) * pcs.Percent / 100)

		// %-7.3f was chosen to make it easy to scan the percentages vertically
		// I tried a few different formats only this one kept the decimals aligned
		fmt.Printf("| %s%s | %-15s| %-12d| %-15s| %-12d| %-7.3f |\n",
			pcs.Name, pad, ConvertUnit(pcs.Size), pcs.Pages, ConvertUnit(cached_size), pcs.Cached, pcs.Percent)

		size_sum += pcs.Size
		page_sum += int64(pcs.Pages)
		cached_page_sum += int64(pcs.Cached)
		cached_size_sum += cached_size
	}

	fmt.Println(hr)
	pad = strings.Repeat(" ", maxName-len("Sum"))
	fmt.Printf("│ %s%s │ %-15s│ %-12d│ %-15s│ %-12d│ %-7.3f │\n",
		"Sum", pad, ConvertUnit(size_sum), page_sum, ConvertUnit(cached_size_sum), cached_page_sum, (float64(cached_page_sum)/float64(page_sum))*100.00)
	fmt.Println(bot)
}

func (stats PcStatusList) FormatPlain() {
	maxName := stats.maxNameLen()

	var size_sum, page_sum, cached_page_sum, cached_size, cached_size_sum int64

	// -nohdr may be chosen to save 2 lines of precious vertical space
	if !nohdrFlag {
		pad := strings.Repeat(" ", maxName-4)
		fmt.Printf("Name%s  Size            Pages        Cached Size     Cached Pages Percent\n", pad)
	}

	for _, pcs := range stats {
		pad := strings.Repeat(" ", maxName-len(pcs.Name))
		cached_size = int64(float64(pcs.Size) * pcs.Percent / 100)

		// %-7.3f was chosen to make it easy to scan the percentages vertically
		// I tried a few different formats only this one kept the decimals aligned
		fmt.Printf("%s%s  %-15s %-12d %-15s %-12d %-7.3f\n",
			pcs.Name, pad, ConvertUnit(pcs.Size), pcs.Pages, ConvertUnit(cached_size), pcs.Cached, pcs.Percent)

		size_sum += pcs.Size
		page_sum += int64(pcs.Pages)
		cached_page_sum += int64(pcs.Cached)
		cached_size_sum += cached_size
	}

	pad := strings.Repeat(" ", maxName-len("Sum"))
	fmt.Printf("%s%s  %-15s %-12d %-15s %-12d %-7.3f\n",
		"Sum", pad, ConvertUnit(size_sum), page_sum, ConvertUnit(cached_size_sum), cached_page_sum, (float64(cached_page_sum)/float64(page_sum))*100.00)
}

func (stats PcStatusList) FormatTerse() {

	if !nohdrFlag {
		fmt.Println("name,size,timestamp,mtime,pages,cached,percent")
	}
	for _, pcs := range stats {
		time := pcs.Timestamp.Unix()
		mtime := pcs.Mtime.Unix()
		fmt.Printf("%s,%d,%d,%d,%d,%d,%g\n",
			pcs.Name, pcs.Size, time, mtime, pcs.Pages, pcs.Cached, pcs.Percent)
	}
}

func (stats PcStatusList) FormatJson(clearpps bool) {
	// clear the per-page status when requested
	// emits an empty "status": [] field in the JSON when disabled, but NBD.
	if clearpps {
		for i := range stats {
			stats[i].PPStat = nil
		}
	}

	b, err := json.Marshal(stats)
	if err != nil {
		log.Fatalf("JSON formatting failed: %s\n", err)
	}
	os.Stdout.Write(b)
	fmt.Println("")
}

// maxNameLen returns the len of longest filename in the stat list
// if the bnameFlag is set, this will return the max basename len
func (stats PcStatusList) maxNameLen() int {
	var maxName int
	for _, pcs := range stats {
		if len(pcs.Name) > maxName {
			maxName = len(pcs.Name)
		}
	}

	if maxName < 5 {
		maxName = 5
	}
	return maxName
}

// define some const unit
// convert origin size data to a friendly readable string.
func ConvertUnit(byteSize int64) string {
	const KB int64 = 1024
	const MB int64 = 1024 * KB
	const GB int64 = 1024 * MB
	const TB int64 = 1024 * GB
	const PB int64 = 1024 * TB

	switch {
	case byteSize >= PB:
		return fmt.Sprintf("%.3fP", (float64(byteSize) / float64(PB)))
	case byteSize >= TB:
		return fmt.Sprintf("%.3fT", (float64(byteSize) / float64(TB)))
	case byteSize >= GB:
		return fmt.Sprintf("%.3fG", (float64(byteSize) / float64(GB)))
	case byteSize >= MB:
		return fmt.Sprintf("%.3fM", (float64(byteSize) / float64(MB)))
	case byteSize >= KB:
		return fmt.Sprintf("%.3fK", (float64(byteSize) / float64(KB)))
	default:
		return fmt.Sprintf("%dB", byteSize)
	}
}
