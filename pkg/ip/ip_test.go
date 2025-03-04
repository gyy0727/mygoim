package ip

import (
	"testing"
)

func TestIP(t *testing.T) {
	ip := InternalIP()
	t.Log(ip)
	if ip == "" {
		t.FailNow()
	}
}
