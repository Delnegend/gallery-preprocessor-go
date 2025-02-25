package tasks

import (
	"context"
	"errors"
	"fmt"
	"gallery-preprocessor-go/backend/internal/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

func Artefact(
	ctx context.Context,
	files []string,
	poolSize int,
	updateProgressBase func(func() float64) func(),
	sendWarning func(error),
) {
	jpgFiles := []string{}
	for _, entry := range files {
		if strings.ToLower(filepath.Ext(entry)) == ".jpg" {
			jpgFiles = append(jpgFiles, entry)
		}
	}

	if len(jpgFiles) == 0 {
		sendWarning(fmt.Errorf("no jpg files found"))
		return
	}

	// output file already exists
	for _, inputJpgFile := range jpgFiles {
		outputPngFile := utils.ReplaceExt(inputJpgFile, ".png")
		if _, err := os.Stat(outputPngFile); err == nil {
			sendWarning(fmt.Errorf("possible output file '%s' already exists", outputPngFile))
			return
		}
	}

	processedFiles := 0
	var progressMutex sync.Mutex
	updateProgress := updateProgressBase(func() float64 {
		progressMutex.Lock()
		defer progressMutex.Unlock()
		processedFiles++
		return float64(processedFiles) / float64(len(jpgFiles))
	})

	pool := utils.NewWorkerPool(ctx, poolSize)

	for _, inputJpgFile := range jpgFiles {
		pool.Run(func() {
			defer updateProgress()

			outputPngFile := utils.ReplaceExt(inputJpgFile, ".png")

			cmd := exec.CommandContext(ctx, "artefact-cli", inputJpgFile, "-o", outputPngFile, "-i", "50")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			outputMsgBytes, err := cmd.CombinedOutput()
			outputMsgString := string(outputMsgBytes)
			switch {
			case err != nil && outputMsgString != "":
				sendWarning(fmt.Errorf("artefact error: %s", outputMsgString))
				return
			case err != nil && outputMsgString == "":
				sendWarning(fmt.Errorf("artefact error: %s", err))
				return
			}

			// check output file exists
			_, err = os.Stat(outputPngFile)
			if errors.Is(err, os.ErrNotExist) {
				sendWarning(fmt.Errorf("output file '%s' not created", outputPngFile))
			} else if err != nil {
				sendWarning(fmt.Errorf("can't check if output file exists: %w", err))
			}
		})
	}

	pool.WaitAndClose()
}
