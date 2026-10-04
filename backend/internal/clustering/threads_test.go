package clustering

import (
	"runtime"
	"testing"
)

func TestOnnxThreads(t *testing.T) {
	if got := onnxThreads(3); got != 3 {
		t.Fatalf("configured threads: got %d, want 3", got)
	}
	if got, want := onnxThreads(0), runtime.GOMAXPROCS(0); got != want || got < 1 {
		t.Fatalf("default threads: got %d, want GOMAXPROCS=%d", got, want)
	}
}
