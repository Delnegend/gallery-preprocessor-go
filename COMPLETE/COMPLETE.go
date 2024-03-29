package main

import (
	"flag"
	"fmt"
	"github.com/gookit/color"
	"libs"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	input          *string
	force_srgan    *bool
	source_format  *string
	target_size    *string
	format         *string
	single         *bool
	model          *string
	resize_threads *string
)

func init() {
	input = flag.String("i", ".", "Input folder")
	source_format = flag.String("sf", "png", "Source format: 'jxl', 'webp' or 'png' (default, jpg = png)")
	format = flag.String("tf", "webp", "Format of processed files: 'avif', 'webp'")
	target_size = flag.String("max", "w2000", "Max size of image in pixel, 0 to disable")
	force_srgan = flag.Bool("force", false, "Force resize using RealESRGAN even if image has dimension larger than max")
	single = flag.Bool("single", false, "Treat the input folder as a single pack")
	model = flag.String("model", "fast", "Model for RealESRGAN: 'fast' or 'details'")
	resize_threads = flag.String("t", "4", "Number of threads for BatchResize")
	flag.Parse()

	if (*input)[len(*input)-1:] == "\"" {
		*input = (*input)[:len(*input)-1]
	}

	os.Stdout.Write([]byte("\033[H\033[2J")) // clear the console

	if *source_format != "jxl" && *source_format != "webp" && *source_format != "png" {
		libs.PrintErr(os.Stderr, "Source format must be 'jxl', 'webp' or 'png'\n")
		*source_format = "png"
	}
	*input = filepath.Clean(*input)
	if *model != "fast" && *model != "details" {
		libs.PrintErr(os.Stderr, "Model must be 'fast' or 'details'\n")
		os.Exit(1)
	} else if *model == "fast" {
		*model = "realesr-animevideov3"
	} else {
		*model = "realesrgan-x4plus-anime"
	}
}

func stage_resize(pack_folder string) string {
	var resize_cmd *exec.Cmd
	resized_folder := pack_folder + "_resized"
	if _, err := os.Stat(resized_folder); os.IsNotExist(err) {
		os.Mkdir(resized_folder, os.ModePerm)
	}
	input_cmd := []string{"-input", pack_folder, "-output", resized_folder, "-threads", *resize_threads, "-target_size", *target_size, "-model", *model, "-log"}
	if *force_srgan {
		input_cmd = append(input_cmd, "-srgan")
	}
	resize_cmd = exec.Command("BatchResize", input_cmd...)
	resize_cmd.Stdout = os.Stdout
	resize_cmd.Stderr = os.Stderr
	if err := resize_cmd.Run(); err != nil {
		fmt.Println(err)
	}
	return resized_folder
}

func stage_resize_ani(pack_folder string, resized_folder string, formats []string) string {
	for _, file := range libs.ListFiles(pack_folder, formats, true, false) {
		output_file := libs.ReplaceIO(file[:len(file)-len(filepath.Ext(file))], pack_folder, resized_folder) + ".webp"
		resize_param := []string{"-i", file, "-o", output_file, "-cleanup"}
		w, h := libs.Dimension(file)
		if w > h {
			resize_param = append(resize_param, "-max", "w900")
		} else {
			resize_param = append(resize_param, "-max", "h900")
		}
		if *force_srgan {
			resize_param = append(resize_param, "-force")
		}
		resize_cmd := exec.Command("UpscaleAni", resize_param...)
		resize_cmd.Stdout = os.Stdout
		resize_cmd.Stderr = os.Stderr
		fmt.Println("==>", file)
		if err := resize_cmd.Run(); err != nil {
			libs.PrintErr(os.Stderr, "==> Error: %s\n", err)
		}
	}
	return resized_folder
}

func stage_transcode(input_folder, output_folder, format string) {
	encode_cmd := exec.Command("BatchConvert", "-input", input_folder, "-output", output_folder, "-format", format, "-log")
	encode_cmd.Stdout = os.Stdout
	encode_cmd.Stderr = os.Stderr
	if err := encode_cmd.Run(); err != nil {
		fmt.Println(err)
	}
}

func stage_compress(pack string, name string, compress_type string, included []string) {
	cwd, _ := os.Getwd()
	os.Chdir(pack)
	archive_cmd := exec.Command("7z", append([]string{"a", "-bt", "-t" + compress_type, "-mx=9", "-r", filepath.Join("../", name+"."+compress_type)}, included...)...)
	if err := archive_cmd.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("==> %s\n", pack)
	os.Chdir(cwd)
}

func process(pack string) {
	libs.PrintSign(pack, "main")

	// ====== Stage 1: Transcode to/from JXL ======
	if *source_format == "jxl" {
		color.Greenf("\nStage 1: Decode JXL to PNG\n")
		stage_transcode(pack, pack, "djxl")
	} else if *source_format == "png" {
		color.Greenf("\nStage 1: Encode to JXL\n")
		stage_transcode(pack, pack, "cjxl")
	} else {
		color.Greenf("\nStage 1: Source file lossly webp, converting to JXL brings no size benefit.\n")
		stage_transcode(pack, pack, "dwebp")
	}

	// ====== Stage 2: Resize ======
	color.Greenf("\nStage 2: Resizing images\n")
	resize_folder := stage_resize(pack)

	color.Greenf("\nStage 2.5: Resizing animations\n")
	transcoded_folder := pack + "_transcoded" // animations get resize and encode directly into webp without going through a transcode step so we can just throw the output file into the transcoded folder
	if _, err := os.Stat(transcoded_folder); os.IsNotExist(err) {
		os.Mkdir(transcoded_folder, os.ModePerm)
	}
	stage_resize_ani(pack, transcoded_folder, []string{".mp4", ".mkv", ".webm", ".gif"})

	// ====== Stage 3: Transcode ======
	color.Greenf("\nStage 3: Transcode\n")
	stage_transcode(resize_folder, transcoded_folder, *format)

	// ====== Stage 4: Compress ======
	if !*single {
		color.Greenf("\nStage 4: Compress\n")
		if *source_format != "jxl" {
			stage_compress(pack, pack, "7z", []string{"*.mp4", "*.webp", "*.gif", "*.jxl"})
		}
		stage_compress(transcoded_folder, pack, "zip", []string{"*.avif", "*.webp"})
	}

	// ====== Stage 5: Cleanup ======
	if *single {
		color.Greenf("\nStage 5: Re-organizing files\n")
		// Create a jxl folder for the transcoded jxl files
		if *source_format != "jxl" {
			jxl_folder := pack + "_jxl"
			os.Mkdir(jxl_folder, os.ModePerm)
			for _, file := range libs.ListFiles(pack, []string{".jxl"}, true, false) {
				os.Rename(file, libs.ReplaceIO(file, pack, jxl_folder))
			}
		}
	} else {
		color.Greenf("\nStage 5: Cleanup\n")
		os.RemoveAll(transcoded_folder)
	}
	os.RemoveAll(resize_folder)
}

func main() {

	BatchResize_in_path := libs.CheckIfBinaryInPath("BatchResize")
	BatchConvert_in_path := libs.CheckIfBinaryInPath("BatchConvert")
	if !BatchResize_in_path || !BatchConvert_in_path {
		libs.PrintErr(os.Stderr, "Error: BatchResize and BatchConvert not found in PATH\n")
		os.Exit(1)
	}


	if !libs.IsDir(*input) {
		libs.PrintErr(os.Stderr, "Input must be a folder\n")
		os.Exit(1)
	}

	fmt.Printf("Source format: %s | Single mode: %t | Target format: %s | Model: %s\n", *source_format, *single, *format, *model)

	if *single {
		process(*input)
	} else {
		os.Chdir(*input)
		for _, pack := range libs.ListFolders(".", false) {
			process(pack)
		}
	}

	exec.Command("wscript", "D:\\sound.vbs").Run()
}
