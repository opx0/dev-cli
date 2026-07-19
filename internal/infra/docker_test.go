package infra

import "testing"

func TestParseBytePair(t *testing.T) {
	left, right := parseBytePair("1.5MiB / 2GB")
	if left != 1572864 || right != 2000000000 {
		t.Fatalf("unexpected bytes: %d / %d", left, right)
	}
}
