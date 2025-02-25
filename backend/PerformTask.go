package backend

import (
	"context"
	"fmt"
	"gallery-preprocessor-go/backend/internal/tasks"
	"os"
	"path/filepath"
	"sync"
)

type TaskID string

var (
	TaskArtefact     TaskID = "Artefact"
	TaskArtefactJxl  TaskID = "ArtefactJxl"
	TaskCjxlLossLess TaskID = "CjxlLossless"
	TaskCjxlLossy    TaskID = "CjxlLossy"
	TaskDjxl         TaskID = "Djxl"
	TaskPar2         TaskID = "Par2"

	AllTasks = []struct {
		Value  TaskID
		TSName string
	}{
		{TaskArtefact, string(TaskArtefact)},
		{TaskArtefactJxl, string(TaskArtefactJxl)},
		{TaskCjxlLossLess, string(TaskCjxlLossLess)},
		{TaskCjxlLossy, string(TaskCjxlLossy)},
		{TaskDjxl, string(TaskDjxl)},
		{TaskPar2, string(TaskPar2)},
	}
)

type TaskInput struct {
	TaskID TaskID
	Inputs []string
}

func PerformTask(taskCtx context.Context, taskInput TaskInput, progressChan chan<- float64, warnChan chan<- error) {
	var taskMutex sync.Mutex
	taskMutex.Lock()

	updateProgressBase := func(f func() float64) func() {
		return func() { go func() { progressChan <- f() }() }
	}
	sendWarning := func(err error) { go func() { warnChan <- err }() }

	files := []string{}
	for _, input := range taskInput.Inputs {
		info, err := os.Stat(input)
		if err != nil {
			sendWarning(fmt.Errorf("can't read input file info: %w", err))
			continue
		}
		if info.IsDir() {
			entries, err := os.ReadDir(input)
			if err != nil {
				sendWarning(fmt.Errorf("can't read input directory: %w", err))
				continue
			}
			for _, entry2 := range entries {
				if entry2.IsDir() {
					continue
				}
				files = append(files, filepath.Join(input, entry2.Name()))
			}
			continue
		}
		files = append(files, input)
	}

	switch taskInput.TaskID {
	case TaskArtefact:
		tasks.Artefact(taskCtx, files, 2, updateProgressBase, sendWarning)
	case TaskArtefactJxl:
		tasks.ArtefactJxl(taskCtx, files, 2, updateProgressBase, sendWarning)
	case TaskCjxlLossLess:
		tasks.Cjxl(taskCtx, files, 2, false, updateProgressBase, sendWarning)
	case TaskCjxlLossy:
		tasks.Cjxl(taskCtx, files, 2, true, updateProgressBase, sendWarning)
	case TaskDjxl:
		tasks.Djxl(taskCtx, files, 2, updateProgressBase, sendWarning)
	case TaskPar2:
		tasks.Par2(taskCtx, files, 2, updateProgressBase, sendWarning)
	default:
		sendWarning(fmt.Errorf("internal error: unknown task %s", taskInput.TaskID))
	}
}
