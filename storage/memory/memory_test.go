package memory

import (
	"testing"

	"github.com/coreos/dex/storage/conformance"
)

func TestStorage(t *testing.T) {
	conformance.RunTestSuite(t, New)
}
