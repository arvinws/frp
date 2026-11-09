//go:build unix

package server

import (
	"os"
	"syscall"
)

func restartCurrentProcess() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// exe is this process's own binary and os.Args are the arguments it was
	// started with. The dashboard request does not choose either value.
	return syscall.Exec(exe, os.Args, os.Environ()) //nolint:gosec // G702: re-exec the current server, not a caller-supplied command
}
