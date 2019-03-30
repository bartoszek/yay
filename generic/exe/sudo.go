package exe

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// SudoLoopBackground leaves a constantly revalidating sudo instance running throughout the program execution
func SudoLoopBackground() {
	updateSudo()
	go sudoLoop()
}

func sudoLoop() {
	for {
		updateSudo()
		time.Sleep(298 * time.Second)
	}
}

func updateSudo() {
	for {
		err := Show(exec.Command("sudo", "-v"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			break
		}
	}
}
