package main

import (
	"errors"
	"flag"
	"fmt"
	"libs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	input_folder  *string
	output_folder *string
	threads       *int
	output_format *string
	export_log    *bool

	failed_files   []string
	skipped_files  []string
	original_size  int
	converted_size int

	all_input_extensions   []string
	allow_input_extensions map[string][]string
	output_extensions      map[string]string
)

func init() {
	threads = flag.Int("threads", 4, "Threads")
	input_folder = flag.String("input", ".", "Input folder")
	output_folder = flag.String("output", "", "Output folder")
	output_format = flag.String("format", "avif", "Output format: avif, cjxl, djxl, webp")
	export_log = flag.Bool("log", false, "Export log")
	flag.Parse()

	*input_folder = libs.Rel(*input_folder)

	if libs.InArr(*output_format, []string{"avif", "cjxl", "djxl", "webp"}) == "" {
		libs.PrintErr(os.Stderr, "Error: Invalid output format %s\n", *output_format)
		os.Exit(1)
	}

	if libs.Rel(*output_folder) == "." || libs.Rel(*output_folder) == "" {
		*output_folder = "output_" + *output_format
	} else {
		*output_folder = libs.Rel(*output_folder)
	}

	if *output_format == "avif" && !libs.CheckIfBinaryInPath("avifenc") && libs.CheckIfBinaryInPath("ffmpeg") {
		libs.PrintErr(os.Stderr, "Error: avifenc or ffmpeg not found in path\n")
		os.Exit(1)
	}
	if *output_format == "cjxl" && !libs.CheckIfBinaryInPath("cjxl") {
		libs.PrintErr(os.Stderr, "Error: cjxl not found in path\n")
		os.Exit(1)
	}
	if *output_format == "djxl" && !libs.CheckIfBinaryInPath("djxl") {
		libs.PrintErr(os.Stderr, "Error: djxl not found in path\n")
		os.Exit(1)
	}
	if *output_format == "webp" && !libs.CheckIfBinaryInPath("ffmpeg") {
		libs.PrintErr(os.Stderr, "Error: ffmpeg not found in path\n")
		os.Exit(1)
	}

	all_input_extensions = []string{".jpg", ".jpeg", ".png", ".bmp", ".tif", ".tiff", ".webp", ".gif", ".mp4", ".webm"}
	allow_input_extensions = map[string][]string{
		"avif": all_input_extensions,
		"cjxl": {".png", ".apng", ".gif", ".jpeg", ".jpg", ".ppm", ".pfm", ".pgx"},
		"djxl": {".jxl"},
		"webp": all_input_extensions,
	}
	output_extensions = map[string]string{
		"avif": ".avif",
		"cjxl": ".jxl",
		"djxl": ".png",
		"webp": ".webp",
	}

	original_size = 0
	converted_size = 0
}

func convertImage(file_in, file_out, output_format string, log *os.File) error {
	var cmd *exec.Cmd
	file_type := filepath.Ext(file_in)
	switch output_format {
	case "avif":
		if libs.InArr(file_type, []string{".jpg", ".jpeg", ".png", ".y4m"}) != "" {
			cmd = exec.Command("avifenc", file_in, file_out, "-y", "444", "-d", "8", "-c", "aom", "--min", "0", "--max", "63", "--minalpha", "0", "--maxalpha", "63", "-a", "aq-mode=1", "-a", "cq-level=30", "-a", "enable-chroma-deltaq=1", "-a", "tune=ssim")
		} else {
			cmd = exec.Command("ffmpeg", "-i", file_in, "-c:v", "libaom-av1", "-b:v", "0", "-qmin", "0", "-qmax", "63", "-crf", "30", "-cpu-used", "6", "-aq-mode", "1", "-pix_fmt", "yuv444p8le", "-aom-params", "enable-chroma-deltaq=1", file_out)
		}
	case "cjxl":
		cmd = exec.Command("cjxl", file_in, file_out, "-e", "8", "-q", "100", "--num_threads", "4")
	case "djxl":
		cmd = exec.Command("djxl", file_in, file_out)
	case "webp":
		cmd = exec.Command("ffmpeg", "-i", file_in, "-compression_level", "6", "-quality", "80", file_out)
	}
	cmd.Stderr = log
	cmd.Stdout = log
	if err := cmd.Run(); err != nil {
		return errors.New("failed to convert")
	}
	return nil
}

func startConvert(file_list []string, log *os.File) {
	wg := new(sync.WaitGroup)
	queue_list := make(chan string)
	wg.Add(*threads)
	for i := 1; i <= *threads; i++ {
		go func() {
			for input_file := range queue_list {
				file_name := strings.TrimSuffix(input_file, filepath.Ext(input_file))
				file_extension := output_extensions[*output_format]

				output_dir := filepath.Dir(libs.ReplaceIO(input_file, *input_folder, *output_folder))
				if _, err := os.Stat(output_dir); os.IsNotExist(err) {
					os.MkdirAll(output_dir, 0755)
				}

				output_file := filepath.Join(output_dir, filepath.Base(file_name)+file_extension)
				if _, err := os.Stat(output_file); err == nil {
					libs.PrintErr(os.Stderr, "==> Already existed: %s\n", input_file)
					skipped_files = append(skipped_files, input_file)
					continue
				}

				if err := convertImage(input_file, output_file, *output_format, log); err == nil {
					original_size = original_size + libs.FileSize(input_file)
					converted_size = converted_size + libs.FileSize(output_file)
				} else {
					failed_files = append(failed_files, input_file)
					libs.PrintErr(os.Stderr, "==> Error: %s - %s\n", input_file, err.Error())
				}
			}
			wg.Done()
		}()
	}
	for _, file := range file_list {
		queue_list <- file
	}
	close(queue_list)
	wg.Wait()
}

func main() {

	media := libs.ListFiles(*input_folder, allow_input_extensions[*output_format], true, false)

	var log *os.File
	log = nil
	if *export_log {
		log, _ = os.Create(fmt.Sprintf("BatchConvert_%s.log", time.Now().Format("2006-01-02-15-04-05")))
	}

	duplicated_files := func(file_list []string) []string {
		array := make([]string, len(file_list))
		copy(array, file_list)
		var dupls []string
		for i := 0; i < len(array); i++ {
			for j := i; j < len(array); j++ {
				file_name_A := strings.TrimSuffix(array[i], filepath.Ext(array[i]))
				file_name_B := strings.TrimSuffix(array[j], filepath.Ext(array[j]))
				if file_name_A == file_name_B && (i != j) {
					if libs.InArr(array[i], dupls) != "" {
						dupls = append(dupls, array[i])
					}
					if libs.InArr(array[j], dupls) != "" {
						dupls = append(dupls, array[j])
					}
				}
			}
		}
		return dupls
	}(media)
	if len(duplicated_files) > 0 {
		fmt.Println("Error: Found duplicated file names:")
		for _, file := range duplicated_files {
			fmt.Println("\t", file)
		}
		os.Exit(1)
	}

	start_time := time.Now().UnixNano() / 1000000
	startConvert(media, log)

	if log != nil {
		log.Close()
	}

	if len(failed_files) > 0 {
		fmt.Fprintln(os.Stderr, "")
		libs.PrintErr(os.Stderr, "%d files were failed to convert to %s.\n", len(failed_files), *output_format)
	}

	fmt.Println(libs.ReportResult(len(media), original_size, converted_size, start_time, time.Now().UnixNano()/1000000))
}
