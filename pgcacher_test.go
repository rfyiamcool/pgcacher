package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	ppc "github.com/tobert/pcstat/pkg"
)

func TestMatch(t *testing.T) {
	assert.True(t, wildcardMatch("xiaorui.cc", "*rui*"))
	assert.True(t, wildcardMatch("xiaorui.cc", "xiaorui?cc"))
	assert.True(t, wildcardMatch("xiaorui.cc", "xiaorui?cc*"))
	assert.True(t, wildcardMatch("xiaorui.cc", "*xiaorui?cc*"))
	assert.True(t, wildcardMatch("github.com/rfyiamcool", "rfy"))
}

func TestNull(t *testing.T) {
	ppc.SwitchMountNs(os.Getegid())
	stat, err := ppc.GetPcStatus(os.Args[0])
	assert.Nil(t, err)

	t.Logf("%s %v", stat.Name, stat.Cached)
}
