package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	if os.Args[1] != "run" {
		log.Fatalf("usage: ccrun run <cmd>")
	}

	args := os.Args[2:]

	cmd := exec.Command("sh", "-c")

	commands := []string{
		"hostname container",
		"hostname", // Verify the change
		"echo 'Current hostname:'",
		"hostname",
		"rm -rf /tmp/busybox-tmp",
		"mkdir -p /tmp/busybox-tmp/bin/",
		"mkdir -p /tmp/busybox-tmp/dev/",
		"mkdir -p /tmp/busybox-tmp/proc/",
		"mkdir -p /tmp/busybox-tmp/sys/",
		"cp /bin/busybox /tmp/busybox-tmp/bin/",
		"cd /tmp/busybox-tmp/bin",
		"ls -R | tail -5",
		"./busybox --install .",
		"ls -R | tail -5",
		"pwd",
		"cd /tmp/busybox-tmp/",
		"chroot . ./bin/sh",
		strings.Join(args, " "),
	}

	cmd.Args = append(cmd.Args, strings.Join(commands, " && "))

	// Set namespace creation flags
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS,
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	// No error handling? sure.

	// Verify the hostname
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}
