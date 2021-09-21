package examples

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSimpleStream(t *testing.T) {
	results := SimpleStream()
	assert.Len(t, results, 10)
}
