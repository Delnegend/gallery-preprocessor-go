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

func ArtefactJxl(
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

	for _, inputJpgFile := range jpgFiles {
		fmt.Println("processing", inputJpgFile)
	}

	if len(jpgFiles) == 0 {
		sendWarning(fmt.Errorf("no jpg files found"))
		return
	}

	// output file already exists
	for _, inputJpgFile := range jpgFiles {
		outputJxlFile := utils.ReplaceExt(inputJpgFile, ".jxl")
		if _, err := os.Stat(outputJxlFile); err == nil {
			sendWarning(fmt.Errorf("possible output file '%s' already exists", outputJxlFile))
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

	tmpDir, err := os.MkdirTemp("", "artefact-jxl-tmp")
	if err != nil {
		sendWarning(fmt.Errorf("can't create temp dir: %w", err))
		return
	}

	pool := utils.NewWorkerPool(ctx, poolSize)

	for i, inputJpgFile := range jpgFiles {
		pool.Run(func() {
			defer updateProgress()

			outputPngFile := filepath.Join(tmpDir, fmt.Sprintf("%d.png", i))
			outputJxlFile := utils.ReplaceExt(inputJpgFile, ".jxl")

			// jpg --artefact--> png
			cmd := exec.CommandContext(ctx, "artefact-cli", inputJpgFile, "-o", outputPngFile, "-i", "50")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			outputMsgBytes, err := cmd.CombinedOutput()
			outputMsgString := string(outputMsgBytes)
			switch {
			case err != nil && outputMsgString != "":
				sendWarning(fmt.Errorf("artefact error: %s", outputMsgString))
				return
			case err != nil && outputMsgString == "":
				sendWarning(fmt.Errorf("artefact error: %w", err))
				return
			}

			// png -> jxl
			cmd = exec.CommandContext(ctx, "cjxl", outputPngFile, outputJxlFile, "-d", "1", "-e", "9")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			outputMsgBytes, err = cmd.CombinedOutput()
			outputMsgString = string(outputMsgBytes)
			switch {
			case err != nil && outputMsgString != "":
				sendWarning(fmt.Errorf("cjxl error: %s", outputMsgString))
				return
			case err != nil && outputMsgString == "":
				sendWarning(fmt.Errorf("cjxl error: %w", err))
				return
			}

			// check if output exists
			if _, err := os.Stat(outputJxlFile); err != nil {
				// sendWarning(fmt.Errorf("output file '%s' does not exist", outputJxlFile))
				// return
				if errors.Is(err, os.ErrNotExist) {
					sendWarning(fmt.Errorf("output file '%s' does not exist", outputJxlFile))
					return
				}
				sendWarning(fmt.Errorf("can't check output file '%s': %w", outputJxlFile, err))
			}
		})
	}

	pool.WaitAndClose()
	if err := os.RemoveAll(tmpDir); err != nil {
		sendWarning(fmt.Errorf("can't remove temp dir: %w", err))
	}
}
