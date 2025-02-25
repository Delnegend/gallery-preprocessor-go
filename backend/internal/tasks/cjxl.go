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

func Cjxl(
	ctx context.Context,
	files []string,
	poolSize int,
	outputLossy bool,
	updateProgressBase func(func() float64) func(),
	sendWarning func(error),
) {
	jpgPngFiles := []string{}
	for _, file := range files {
		fileExt := strings.ToLower(filepath.Ext(file))
		if fileExt == ".jpg" || fileExt == ".png" {
			jpgPngFiles = append(jpgPngFiles, file)
		}
	}

	if len(jpgPngFiles) == 0 {
		sendWarning(fmt.Errorf("no jpg or png files found"))
		return
	}

	fileNamesWithoutExt := []string{}

	// check if output files already exist, or 2 files jpg and png
	// with the same name might result in the same output jxl file
	for _, inputFile := range jpgPngFiles {
		outputFile := utils.ReplaceExt(inputFile, ".jxl")
		if _, err := os.Stat(outputFile); err == nil {
			sendWarning(fmt.Errorf("possible output file '%s' already exists", outputFile))
			return
		}

		withoutExt := utils.ReplaceExt(inputFile, "")
		if utils.Contains(fileNamesWithoutExt, withoutExt) {
			sendWarning(fmt.Errorf("duplicate possible output file for '%s'", inputFile))
			return
		}
		fileNamesWithoutExt = append(fileNamesWithoutExt, withoutExt)
	}

	processedFiles := 0
	var progressMutex sync.Mutex
	updateProgress := updateProgressBase(func() float64 {
		progressMutex.Lock()
		defer progressMutex.Unlock()
		processedFiles++
		return float64(processedFiles) / float64(len(jpgPngFiles))
	})

	pool := utils.NewWorkerPool(ctx, poolSize)

	distance := "0"
	if outputLossy {
		distance = "1"
	}

	for _, inputFile := range jpgPngFiles {
		pool.Run(func() {
			defer updateProgress()
			outputFile := utils.ReplaceExt(inputFile, ".jxl")

			// convert jpg/png to jxl
			cmd := exec.CommandContext(ctx, "cjxl", inputFile, outputFile, "-d", distance, "-e", "9")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			outputMsgBytes, err := cmd.CombinedOutput()
			outputMsgString := string(outputMsgBytes)
			switch {
			case err != nil && outputMsgString != "":
				sendWarning(fmt.Errorf("cjxl error: %s", outputMsgString))
				return
			case err != nil && outputMsgString == "":
				sendWarning(fmt.Errorf("cjxl error: %w", err))
				return
			}

			// check output file exists
			_, err = os.Stat(outputFile)
			if errors.Is(err, os.ErrNotExist) {
				sendWarning(fmt.Errorf("output file '%s' not created", outputFile))
			} else if err != nil {
				sendWarning(fmt.Errorf("can't check if output file exists: %w", err))
			}
		})
	}

	pool.WaitAndClose()
}
