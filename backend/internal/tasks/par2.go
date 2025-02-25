package tasks

import (
	"context"
	"fmt"
	"gallery-preprocessor-go/backend/internal/utils"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

func Par2(
	ctx context.Context,
	files []string,
	poolSize int,
	updateProgressBase func(func() float64) func(),
	sendWarning func(error),
) {
	_7zFiles := []string{}
	for _, file := range files {
		fileExt := strings.ToLower(filepath.Ext(file))
		if fileExt == ".par2" {
			sendWarning(fmt.Errorf("there's .par2 file in the input"))
			return
		}
		if fileExt == ".7z" {
			_7zFiles = append(_7zFiles, file)
		}
	}

	if len(_7zFiles) == 0 {
		sendWarning(fmt.Errorf("no 7z files found"))
		return
	}

	processedFiles := 0
	var progressMutex sync.Mutex
	updateProgress := updateProgressBase(func() float64 {
		progressMutex.Lock()
		defer progressMutex.Unlock()
		processedFiles++
		return float64(processedFiles) / float64(len(_7zFiles))
	})

	pool := utils.NewWorkerPool(ctx, poolSize)

	for _, input7zFile := range _7zFiles {
		pool.Run(func() {
			defer updateProgress()

			cmd := exec.CommandContext(ctx, "par2j64.exe", "c", "/rr11", input7zFile+".par2", input7zFile)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			outputMsgBytes, err := cmd.CombinedOutput()
			outputMsgString := string(outputMsgBytes)
			switch {
			case err != nil && outputMsgString != "":
				sendWarning(fmt.Errorf("par2 error: %s", outputMsgString))
			case err != nil && outputMsgString == "":
				sendWarning(fmt.Errorf("par2 error: %w", err))
			}
		})
	}

	pool.WaitAndClose()
}
