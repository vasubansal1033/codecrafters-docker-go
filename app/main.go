package main

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	// change root filesystem for the child process using chroot
	// this is necessary to make the child process believe it is running in a different root filesystem
	EnterNewJail()

	cmd := exec.Command(command, args...)

	// Process isolation
	cmd.SysProcAttr = &syscall.SysProcAttr {
		Cloneflags: syscall.CLONE_NEWPID,
	}

	// bind the standard input, output and error to the parent process
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	// exit with the same exit code as the child process
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			os.Exit(exitError.ExitCode())
		}
	}
}

func EnterNewJail() {
	rootFsPath, err := os.MkdirTemp("", "root_fs_*")
	if err != nil {
		logAndThrowError(err, "Failed to create temporary directory")
	}
	// rwxr-xr-x
	err = os.Chmod(rootFsPath, 0755)
	if err != nil {
		logAndThrowError(err, "Failed to change permissions of temporary directory")
	}

	defer os.Remove(rootFsPath)

	binPath := "/usr/local/bin"

	err = os.MkdirAll(path.Join(rootFsPath, binPath), 0755)
	if err != nil {
		logAndThrowError(err, "Failed to create bin directory")
	}

	os.MkdirAll(path.Join(rootFsPath, "/usr/local/bin"), 0755)

	// Linking the two paths instead of copying
	// TODO: do actual copying of files
	os.Link("/usr/local/bin/docker-explorer", path.Join(rootFsPath, "/usr/local/bin/docker-explorer"))
	if err != nil {
		logAndThrowError(err, "Failed to copy binaries to root file system")
	}

	err = syscall.Chroot(rootFsPath)
	if err != nil {
		logAndThrowError(err, "Failed to change root filesystem")
	}
}

func logAndThrowError(err error, errorMessage string) {
	log.Fatalf("%s: %v", errorMessage, err)
	os.Exit(1)
}
