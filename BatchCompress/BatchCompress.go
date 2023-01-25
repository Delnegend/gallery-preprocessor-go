package main

import (
	"flag"
	"fmt"
	"libs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	input_file_extension    *string
	output_file_format *string
)

func init() {
	output_file_format = flag.String("f", ".7z", "Format of output file, .7z or .zip")
	input_file_extension = flag.String("e", "*", "Extension to be included in output file, * for all, or a list of extensions including the dot")
	flag.Parse()
}

func compress(path string, format string, extension_list_ string) {
	os.Chdir(path)
	compress_cmd := []string{"7z", "a", "-bt", "-t" + format[1:], "-mx=9", "-r", filepath.Join("../", path+format)}
	extension_list := strings.Split(extension_list_, " ")
	if extension_list[0] == "*" {
		compress_cmd = append(compress_cmd, "*.*")
	} else {
		compress_cmd = append(compress_cmd, func(exts []string) []string {
			var ret []string
			for _, ext := range exts {
				ret = append(ret, "*"+ext)
			}
			return ret
		}(extension_list)...)
	}
	cmd := exec.Command(compress_cmd[0], compress_cmd[1:]...)
	if err := cmd.Run(); err != nil {
		libs.PrintErr(os.Stderr, "Error: %s\n%s\n", path, err)
	}
	fmt.Printf("==> %s\n", path)
	os.Chdir("..")
}

func main() {
	folders_to_compress := libs.ListFolders(".", false)
	for _, folder := range folders_to_compress {
		compress(folder, *output_file_format, *input_file_extension)
	}
}
