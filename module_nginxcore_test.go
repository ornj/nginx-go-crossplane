package crossplane

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlags(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0x00000001, ConfNoArgs)
	assert.Equal(t, 0x00000002, ConfTake1)
	assert.Equal(t, 0x00000004, ConfTake2)
	assert.Equal(t, 0x00000008, ConfTake3)
	assert.Equal(t, 0x00000010, ConfTake4)
	assert.Equal(t, 0x00000020, ConfTake5)
	assert.Equal(t, 0x00000040, ConfTake6)
	assert.Equal(t, 0x00000080, ConfTake7)
	assert.Equal(t, 0x00000100, ConfBlock)
	assert.Equal(t, 0x00000200, ConfFlag)
	assert.Equal(t, 0x00000400, ConfAny)
	assert.Equal(t, 0x00000800, Conf1More)
	assert.Equal(t, 0x00001000, Conf2More)

	assert.Equal(t, 8, ConfMaxArgs)
}
