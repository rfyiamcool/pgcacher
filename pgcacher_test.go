package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	assert.True(t, wildcardMatch("xiaorui.cc", "*rui*"))
	assert.True(t, wildcardMatch("xiaorui.cc", "xiaorui?cc"))
	assert.True(t, wildcardMatch("xiaorui.cc", "xiaorui?cc*"))
	assert.True(t, wildcardMatch("xiaorui.cc", "*xiaorui?cc*"))
	assert.True(t, wildcardMatch("github.com/rfyiamcool", "rfy"))
}
