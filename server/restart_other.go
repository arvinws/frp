//go:build !unix

package server

import "fmt"

func restartCurrentProcess() error {
	return fmt.Errorf("in-process restart is not supported on this platform")
}
