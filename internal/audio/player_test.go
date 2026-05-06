package audio

import (
	"context"
	"strings"
	"testing"
)

func TestStreamViaOtoBadMP3(t *testing.T) {
	err := StreamViaOto(context.Background(), strings.NewReader("not-mp3"))
	if err == nil {
		t.Fatalf("expected decode error")
	}
}
