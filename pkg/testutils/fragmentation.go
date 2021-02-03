package testutils

import (
	"os/exec"
)

// Fragment launch the fragment.py script (target a nigiri instance i.e with default rpc credentials and endpoint)
func Fragment(addresses []string) error {
	args := []string{
		"../../resources/fragment/fragment.py",
		"--unsafe",
		"--fragment_size", "0.00001",
		"--addresses",
	}

	args = append(args, addresses...)

	cmd := exec.Command("python3", args...)

	_, err := cmd.Output()
	if err != nil {
		return err
	}

	return nil
}
