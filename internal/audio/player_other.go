//go:build !darwin

package audio

import (
	"context"
	"io"
)

// StreamToSpeakers plays MP3 audio via the oto backend by default.
func StreamToSpeakers(ctx context.Context, r io.Reader) error {
	return StreamViaOto(ctx, r)
}
