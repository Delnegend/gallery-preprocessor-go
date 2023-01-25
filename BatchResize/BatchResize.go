package main

import (
	"flag"
	"fmt"
	"libs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var (
	input       *string
	output      *string
	target_size *string
	threads     *int
	force_srgan *bool
	model       *string
	export_log  *bool
)

func init() {
	input = flag.String("input", ".", "Input folder")
	output = flag.String("output", "resize_output", "Output folder")
	threads = flag.Int("threads", 4, "Number of threads")
	target_size = flag.String("target_size", "w2500", "Destination size (README.md for more info)")
	force_srgan = flag.Bool("srgan", false, "Force resize using RealESRGAN for image has dimension larger than max")
	model = flag.String("model", "realesr-animevideov3", "Model for RealESRGAN")
	export_log = flag.Bool("log", false, "Export log of subprocesses to BatchResize_<date>.log")
	flag.Parse()
	*input = filepath.Clean(*input)
	*output = filepath.Clean(*output)
}

func resize(input_file, output_file string, log *os.File) error {
	config := *target_size
	mode := config[0:1]
	target_size := libs.StrToInt(config[1:])

	// Manually set ratio
	if mode == "r" {
		cmd := exec.Command("realesrgan-ncnn-vulkan", "-n", *model, "-i", input_file, "-o", output_file, "-s", fmt.Sprintf("%d", target_size))
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	}

	// Calculate ratio from given target size and image dimension (w200 -> scale to 200px width, h200 -> scale to 200px height)
	w, h := libs.Dimension(input_file)
	if h == 0 || w == 0 {
		return fmt.Errorf("%s is not a valid image", input_file)
	}

	var source_size int
	if (mode == "w") || (libs.InArr(mode, []string{"w", "h"}) == "" && w >= h) {
		source_size = w
		mode = "w" // Enforce mode to be w or h
	} else {
		source_size = h
		mode = "h"
	}

	ratio := 2
	if source_size*3 >= target_size {
		ratio = 3
	} else if source_size*4 >= target_size {
		ratio = 4
	}
	if *model == "realesrgan-x4plus-anime" {
		ratio = 4
	}

	// Upscale
	upscaled_file := input_file + ".upscaled.png"
	if *force_srgan || source_size < target_size {
		srgan_cmd := exec.Command("realesrgan-ncnn-vulkan", "-i", input_file, "-o", upscaled_file, "-s", fmt.Sprintf("%d", ratio), "-n", *model)
		srgan_cmd.Stdout = log
		srgan_cmd.Stderr = log
		if err := srgan_cmd.Run(); err != nil {
			return err
		}
	} else {
		if err := libs.Copy(input_file, upscaled_file); err != nil {
			return err
		}
	}

	// Resize down
	if source_size*ratio > target_size && target_size != 0 {
		var cmd_resize *exec.Cmd
		if mode == "w" {
			cmd_resize = exec.Command("ffmpeg", "-i", upscaled_file, "-q:v", "2", "-vf", fmt.Sprintf(`scale='min(%d,iw)':-1`, target_size), output_file)
		} else {
			cmd_resize = exec.Command("ffmpeg", "-i", upscaled_file, "-q:v", "2", "-vf", fmt.Sprintf(`scale='-1:min(%d,ih)'`, target_size), output_file)
		}
		cmd_resize.Stdout = log
		cmd_resize.Stderr = log
		if err := cmd_resize.Run(); err != nil {
			return err
		}
		os.Remove(upscaled_file)
	} else {
		if err := os.Rename(upscaled_file, output_file); err != nil {
			return err
		}
	}
	return nil
}

func startResize(input_file_list []string, log *os.File) {
	wg := new(sync.WaitGroup)
	files_queue := make(chan string)
	wg.Add(*threads)
	for i := 1; i <= *threads; i++ {
		go func() {
			for input_file := range files_queue {
				file_name := libs.ReplaceIO(input_file, *input, *output)

				output_file := file_name[:len(file_name)-len(filepath.Ext(file_name))] + ".png"
				if _, err := os.Stat(output_file); err == nil {
					libs.PrintErr(os.Stderr, "==> Already existed: %s\n", input_file)
					continue
				}

				output_foler := filepath.Dir(output_file)
				if _, err := os.Stat(output_foler); os.IsNotExist(err) {
					os.MkdirAll(output_foler, 0755)
				}

				if err := resize(input_file, output_file, log); err != nil {
					libs.PrintErr(os.Stderr, "==> Error: %s - %s\n", input_file, err)
				}
			}
			wg.Done()
		}()
	}
	for _, file := range input_file_list {
		files_queue <- file
	}
	close(files_queue)
	wg.Wait()
}

func main() {
	if libs.Rel(*input) == libs.Rel(*output) {
		libs.PrintErr(os.Stderr, "Input and output folder are the same, please use -o to specify output folder")
		os.Exit(1)
	}
	if !libs.IsDir(*input) {
		libs.PrintErr(os.Stderr, "Input folder must be a directory\n")
	}

	var log *os.File
	log = nil
	if *export_log {
		log, _ = os.Create(fmt.Sprintf("BatchResize_%s.log", time.Now().Format("2006-01-02-15-04-05")))
	}
	defer log.Close()

	startResize(libs.ListFiles(*input, []string{".png", ".jpg", ".jpeg", ".webp"}, true, false), log)
}
