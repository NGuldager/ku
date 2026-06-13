package upgrade

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func verifyFileChecksum(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening downloaded binary: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing downloaded binary: %w", err)
	}
	got := fmt.Sprintf("%x", h.Sum(nil))
	if got != want {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", want, got)
	}
	return nil
}
