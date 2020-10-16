package dockerhub

import (
	"testing"

	mclog "github.com/chryscloud/go-microkit-plugins/log"
)

var (
	zl, _ = mclog.NewZapLogger("info")
)

func TestInit(t *testing.T) {
	var option Option
	cl := NewClient(option, Log(zl))
	tags, err := cl.Tags("chryscloud/chrysedgeproxy")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) <= 0 {
		t.Fatalf("expected more than 0 repositories, got %v", len(tags))
	}
}
