// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSanitizeUrl_Empty(t *testing.T) {
	// arrange / act
	url := SanitizeUrl("", true)

	// assert
	assert.Empty(t, url)
}

func TestSanitizeUrl_JustSlash(t *testing.T) {
	// arrange
	url := "/"

	// act
	url = SanitizeUrl(url, true)

	// assert
	assert.Equal(t, "/", url)
}

func TestSanitizeUrl_MissingTrailingForwardSlash(t *testing.T) {
	// arrange
	url := "http://dumb.net"

	// act
	url = SanitizeUrl(url, true)

	// assert
	assert.Equal(t, "http://dumb.net/", url)
}

func TestSanitizeUrl_ExcessTrailingSlashes(t *testing.T) {
	// arrange
	url := "http://dumb.net///"

	// act
	url = SanitizeUrl(url, true)

	// assert
	assert.Equal(t, "http://dumb.net/", url)
}

func TestSanitizeUrl_ExcessSlashesInBetween(t *testing.T) {
	// arrange
	url := "https://why//would///you///do///this"

	// act
	url = SanitizeUrl(url, true)

	// assert
	assert.Equal(t, "https://why/would/you/do/this/", url)
}

func TestSanitizeUrl_RemoveTrailingSlash(t *testing.T) {
	// arrange
	url := "http://dumb.net/"

	// act
	url = SanitizeUrl(url, false)

	// assert
	assert.Equal(t, "http://dumb.net", url)
}

func TestSanitizeUrl_RemoveTrailingSlash_JustSlash(t *testing.T) {
	// arrange
	url := "/"

	// act
	url = SanitizeUrl(url, false)

	// assert
	assert.Equal(t, "", url)
}
