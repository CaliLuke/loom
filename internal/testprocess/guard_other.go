//go:build !unix

package testprocess

import "os/exec"

type processGuard struct{}

func startGuard(*exec.Cmd) (*processGuard, error) {
	return &processGuard{}, nil
}

func (*processGuard) finish() error {
	return nil
}
