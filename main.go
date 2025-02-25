package main

import (
	"context"
	"embed"
	"fmt"
	"gallery-preprocessor-go/backend"
	"sync"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

type App struct {
	ctx context.Context
}

type OtherEmitID string

const (
	ProgressEmitID   OtherEmitID = "Progress"
	WarningEmitID    OtherEmitID = "Warning"
	CancelTaskEmitID OtherEmitID = "CancelTask"
	TaskDoneEmitID   OtherEmitID = "TaskDone"
	TaskStartEmitID  OtherEmitID = "TaskStart"
)

func main() {
	app := App{}

	// Create application with options
	err := wails.Run(&options.App{
		Title: "gallery-preprocessor-go",

		Width:         320,
		Height:        400,
		DisableResize: true,

		AlwaysOnTop:      true,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			app.ctx = ctx
		},
		Bind: []interface{}{&app},
		EnumBind: []interface{}{
			backend.AllTasks,
			[]struct {
				Value  OtherEmitID
				TSName string
			}{
				{ProgressEmitID, string(ProgressEmitID)},
				{WarningEmitID, string(WarningEmitID)},
				{CancelTaskEmitID, string(CancelTaskEmitID)},
				{TaskDoneEmitID, string(TaskDoneEmitID)},
				{TaskStartEmitID, string(TaskStartEmitID)},
			},
		},
		DragAndDrop: &options.DragAndDrop{EnableFileDrop: true},
		OnDomReady: func(ctx context.Context) {
			progressChan := make(chan float64)
			warnChan := make(chan error)
			var taskMutex sync.Mutex

			go func() {
				for progress := range progressChan {
					runtime.EventsEmit(ctx, string(ProgressEmitID), progress)
				}
			}()
			go func() {
				for warn := range warnChan {
					runtime.EventsEmit(ctx, string(WarningEmitID), warn.Error())
				}
			}()

			var taskCancel context.CancelFunc
			for _, taskType := range backend.AllTasks {
				taskID := taskType.TSName
				runtime.EventsOn(ctx, taskID, func(data ...interface{}) {
					if len(data) != 1 {
						warnChan <- fmt.Errorf("expect 1 argument from frontend, got %v", data)
						return
					}

					taskMutex.Lock()
					defer taskMutex.Unlock()

					var taskCtx context.Context
					taskCtx, taskCancel = context.WithCancel(ctx)
					defer func() { taskCancel = nil }()

					taskInput := backend.TaskInput{Inputs: []string{}, TaskID: taskType.Value}
					for _, input := range data[0].([]interface{}) {
						inputString, ok := input.(string)
						if !ok {
							warnChan <- fmt.Errorf("expect string from frontend, got %v", input)
							return
						}
						taskInput.Inputs = append(taskInput.Inputs, inputString)
					}

					runtime.EventsEmit(ctx, string(TaskStartEmitID), taskID)
					defer runtime.EventsEmit(ctx, string(TaskDoneEmitID), taskID)
					backend.PerformTask(taskCtx, taskInput, progressChan, warnChan)
				})
			}
			runtime.EventsOn(ctx, string(CancelTaskEmitID), func(data ...interface{}) {
				if taskCancel != nil {
					taskCancel()
				}
				runtime.EventsEmit(ctx, string(TaskDoneEmitID))
			})
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
