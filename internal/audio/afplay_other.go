//go:build !darwin

package audio

import (
	"context"
	"errors"
	"io"
)

// StreamViaAfplay is only available on macOS.
func StreamViaAfplay(_ context.Context, _ io.Reader) error {
	return errors.New("afplay backend is only available on macOS")
}
