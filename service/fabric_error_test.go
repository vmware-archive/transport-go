package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetFabricError(t *testing.T) {

	fe := GetFabricError("test", 500, "something is wrong")
	assert.Equal(t, "test", fe.Title)
	assert.Equal(t, "something is wrong", fe.Detail)
	assert.Equal(t, 500, fe.Status)

}
