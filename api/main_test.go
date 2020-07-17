package api

import (
	"testing"

	"github.com/clear-ness/qa-discussion/testlib"
)

func TestMain(m *testing.M) {
	var options = testlib.HelperOptions{
		EnableStore: true,
	}

	mainHelper = testlib.NewMainHelperWithOptions(&options)
	defer mainHelper.Close()

	mainHelper.Main(m)
}
