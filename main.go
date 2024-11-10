package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const envInitPid = "CONTAINER_INIT=1"
const containerName = "container-1"

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s run <command>\n", os.Args[0])
		os.Exit(1)
	}

	if os.Getenv("CONTAINER_INIT") == "1" {
		containerInit()
		return
	}

	if os.Args[1] == "run" {
		containerSetup()
		return
	}

	fmt.Printf("Unknown command: %s\n", os.Args[1])
	os.Exit(1)
}

func containerSetup() {
	if os.Geteuid() != 0 {
		log.Fatal("This program must be run as root")
	}

	// Create a single parent directory for all container-related dirs
	containerDir := filepath.Join("/tmp/", containerName)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		log.Fatal(err)
	}

	log.Printf("Creating folder structure...\n")

	// Set up directory structure
	containerRoot := filepath.Join(containerDir, "root")
	upperDir := filepath.Join(containerDir, "upper")
	workDir := filepath.Join(containerDir, "work")
	lowerDir := "./alpine" // this remains outside as it's our base image

	// Create the subdirectories
	dirs := []string{containerRoot, upperDir, workDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}
	}

	exec.Command("ls", containerDir).Run()

	log.Printf("Mounting overlayfs...\n")

	// Mount overlayfs
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)
	if err := syscall.Mount("overlay", containerRoot, "overlay", 0, opts); err != nil {
		log.Fatalf("Failed to mount overlayfs: %v", err)
	}

	log.Printf("Running self in new uts namespace, exact command: %v\n", []string{os.Args[0], strings.Join(os.Args[1:], " ")})

	// Start container setup process.
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      1000,
				Size:        65536,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      1000,
				Size:        65536,
			},
		},
		GidMappingsEnableSetgroups: false,
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), envInitPid)

	// Host cleanup.
	defer func() {
		log.Println("Unmounting container root...")
		if err := syscall.Unmount(containerRoot, 0); err != nil {
			log.Printf("Failed to unmount container root at %q: %v", containerRoot, err)
		}
	}()

	log.Println("Starting container...")
	if err := cmd.Run(); err != nil {
		log.Printf("Error running command: %v\n", err)
	}
}

func containerInit() {
	log.Printf("User id %v", os.Getuid())

	hostname := "container-1"
	log.Printf("Setting up hostname %q", hostname)
	if err := syscall.Sethostname([]byte(hostname)); err != nil {
		log.Fatalf("Setting new hostname failed: %v", err)
	}

	containerRoot := filepath.Join("/tmp/", containerName, "root")
	log.Printf("Changing root to %q\n", containerRoot)
	if err := syscall.Chroot(containerRoot); err != nil {
		log.Fatalf("Chroot failed: %v", err)
	}

	log.Printf("Changing dir to /\n")
	if err := os.Chdir("/"); err != nil {
		log.Fatalf("Chdir failed: %v", err)
	}

	if err := syscall.Mount("none", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		log.Fatalf("Failed to make mount namespace private")
	}

	log.Printf("Mounting proc...\n")
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		log.Fatalf("Chdir failed: %v", err)
	}

	log.Printf("Running user command: %v\n", []string{os.Args[2], strings.Join(os.Args[3:], " ")})
	// Exec replaces current init process with the user's desired command.
	if err := syscall.Exec(os.Args[2], os.Args[3:], os.Environ()); err != nil {
		log.Fatalf("Failed calling user command: %v", err)
	}
}
