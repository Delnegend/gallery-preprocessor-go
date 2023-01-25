package libs

import (
	"fmt"
	"time"
	"strings"
)

func ProgressBar(current, total int, startTime time.Time, barLength int) string {
	percent := float64(current) / float64(total)
	barFill := strings.Repeat("=", int(percent * float64(barLength)))
	barEmpty := strings.Repeat(" ", barLength - len(barFill))
	bar := fmt.Sprintf("%s%s", barFill, barEmpty)
	timeTaken := time.Since(startTime).Seconds()
	eta := (timeTaken / float64(current)) * float64(total - current)
	finishAt := time.Now().Add(time.Duration(eta) * time.Second)

	humanReadableTimeTaken := HumanReadableTime(timeTaken*1000)
	humanReadableEta := HumanReadableTime(eta*1000)

	return fmt.Sprintf(
		"%d / %d [%s] %.2f%% %s / %s (%s)",
		current,
		total,
		bar,
		percent * 100,
		humanReadableTimeTaken,
		humanReadableEta,
		finishAt.Format("15:04:05"))
}