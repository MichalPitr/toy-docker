package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

const envInitPid = "CONTAINER_INIT=1"
const containerName = "container-1"
const cGroupPath = "/sys/fs/cgroup/toydocker.slice/"

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

	cleanupOverlayfsMount := setupContainerFilesystem(containerDir)
	defer cleanupOverlayfsMount()

	f := setupCgroups()
	defer f.Close()
	cgroupFd := f.Fd()

	// Start container setup process.
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), envInitPid)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWCGROUP, // TODO: add cgroup ns
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      1000, // TODO: switch from default user to dedicated toy-docker user.
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
		CgroupFD:                   int(cgroupFd),
		UseCgroupFD:                true,
	}

	log.Printf("Spawning container init process: %v\n", []string{os.Args[0], strings.Join(os.Args[1:], " ")})
	if err := cmd.Run(); err != nil {
		log.Printf("Error running command: %v\n", err)
	}
}

func setupCgroups() *os.File {
	p := path.Join(cGroupPath, containerName)
	log.Printf("Creating cgroup folder: %q", p)
	if err := os.MkdirAll(p, 0755); err != nil {
		log.Fatalf("Failed to create cgroup folder %q: %v", p, err)
	}

	// Allow modifying cgroup rules for the container.
	mustWriteToFile(path.Join(cGroupPath, "cgroup.subtree_control"), "+cpu +memory")

	log.Printf("Setting cgroup rules...")
	mustWriteToFile(path.Join(p, "cpu.max"), "10000 100000")
	mustWriteToFile(path.Join(p, "memory.max"), "512M")
	mustWriteToFile(path.Join(p, "memory.swap.max"), "0")

	f, err := os.Open(p)
	if err != nil {
		log.Fatalf("failed to open cgroup folder: %v", err)
	}
	return f
}

func setupContainerFilesystem(containerDir string) func() {
	containerRoot := filepath.Join(containerDir, "root")
	upperDir := filepath.Join(containerDir, "upper")
	workDir := filepath.Join(containerDir, "work")
	lowerDir := "./alpine"

	log.Printf("Setting up folder structure...\n")
	dirs := []string{containerRoot, upperDir, workDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("Mounting overlayfs...\n")
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)
	if err := syscall.Mount("overlay", containerRoot, "overlay", 0, opts); err != nil {
		log.Fatalf("Failed to mount overlayfs: %v", err)
	}

	return func() {
		log.Printf("Unmounting %q...", containerRoot)
		if err := syscall.Unmount(containerRoot, 0); err != nil {
			log.Printf("Failed to unmount container root at %q: %v", containerRoot, err)
		}

		log.Printf("Removing %q...", containerDir)
		if err := os.RemoveAll(containerDir); err != nil {
			log.Printf("Failed to remove container's tmp directory: %v", err)
		}
	}
}

func containerInit() {
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
		log.Fatalf("Mounting proc failed: %v", err)
	}

	log.Printf("Mounting cgroups...\n")
	if err := os.MkdirAll("/sys/fs/cgroup", 0755); err != nil {
		log.Fatalf("Error creating cgroup dir: %v", err)
	}
	if err := syscall.Mount("none", "/sys/fs/cgroup", "cgroup2", 0, ""); err != nil {
		log.Fatalf("Mounting cgroup2 failed: %v", err)
	}

	log.Printf("Running user command: %v\n", []string{os.Args[2], strings.Join(os.Args[3:], " ")})

	// Exec replaces current init process with the user's desired command.
	if err := syscall.Exec(os.Args[2], os.Args[3:], os.Environ()); err != nil {
		log.Fatalf("Failed calling user command: %v", err)
	}
}

func mustWriteToFile(filename, message string) {
	err := os.WriteFile(filename, []byte(message), 0644)
	if err != nil {
		log.Fatalf("failed to write to file: %v", err)
	}
}
