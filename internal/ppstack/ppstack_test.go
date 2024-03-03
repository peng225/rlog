package ppstack

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPPStack(t *testing.T) {
	b := new(bytes.Buffer)
	pps := NewParenPrintStack(b)
	require.NotNil(t, pps)

	assert.True(t, pps.Empty())

	assert.NoError(t, pps.Push())
	assert.Equal(t, "(", b.String())
	assert.False(t, pps.Empty())

	assert.NoError(t, pps.Push())
	assert.Equal(t, "((", b.String())
	assert.False(t, pps.Empty())

	assert.NoError(t, pps.Pop())
	assert.Equal(t, "(()", b.String())
	assert.False(t, pps.Empty())

	assert.NoError(t, pps.Push())
	assert.Equal(t, "(()(", b.String())
	assert.False(t, pps.Empty())

	for !pps.Empty() {
		assert.NoError(t, pps.Pop())
	}
}
