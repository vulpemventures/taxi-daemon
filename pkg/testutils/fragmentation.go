package testutils

import (
	"os/exec"
	"strings"
)

// Fragment launch the fragment.py script (target a nigiri instance i.e with default rpc credentials and endpoint)
func Fragment(addresses []string) error {
	addressesArgs := strings.Join(addresses, " ")
	cmd := exec.Command(
		"/bin/bash", "-c",
		"python", "../../scripts/fragment.py",
		"--unsafe",
		"--fragment_size", "0.00001",
		"--addresses", addressesArgs,
	)

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
