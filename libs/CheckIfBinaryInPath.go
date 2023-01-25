package libs

import (
	"os/exec"
)

func CheckIfBinaryInPath(binary string) bool {
	_, err := exec.LookPath(binary)
	return err == nil
}
