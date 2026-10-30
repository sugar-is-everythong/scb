package tools

import (
	"testing"
)

func TestFile(t *testing.T) {
	for _, pan := range GetPanName() {
		println(pan)
	}
}
