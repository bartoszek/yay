package exe

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Show executes and prints input command
func Show(cmd *exec.Cmd) error {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("")
	}
	return nil
}

// Capture executes input command and captures the output as a return
func Capture(cmd *exec.Cmd) (string, string, error) {
	var outbuf, errbuf bytes.Buffer

	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	stdout := strings.TrimSpace(outbuf.String())
	stderr := strings.TrimSpace(errbuf.String())

	return stdout, stderr, err
}
