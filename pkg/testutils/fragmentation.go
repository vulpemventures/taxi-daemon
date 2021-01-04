package testutils

import (
	"log"
	"os/exec"
)

// Fragment launch the fragment.py script (target a nigiri instance i.e with default rpc credentials and endpoint)
func Fragment(addresses []string) error {
	args := []string{
		"../../scripts/fragment.py",
		"--unsafe",
		"--fragment_size", "0.00001",
		"--addresses",
	}

	args = append(args, addresses...)

	cmd := exec.Command("python3", args...)

	// TODO remove log
	output, err := cmd.Output()
	log.Print(string(output))
	if err != nil {
		return err
	}

	return nil
}
