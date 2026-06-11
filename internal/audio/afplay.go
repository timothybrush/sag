//go:build darwin

package audio

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// StreamViaAfplay plays audio using macOS afplay.
func StreamViaAfplay(ctx context.Context, r io.Reader) error {
	tmp, err := os.CreateTemp("", "sag-*.audio")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write audio to temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	cmd := exec.CommandContext(ctx, "afplay", tmp.Name())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("afplay: %w", err)
	}
	return nil
}
