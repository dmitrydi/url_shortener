package storage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicStorage(t *testing.T) {
	prefix := "prefix/"
	initURL := "some_url"
	stor := NewBasicStorage(prefix)
	shortURL, err := stor.Put(initURL)
	require.NoError(t, err)
	assert.Equal(t, len(strings.TrimPrefix(shortURL, prefix)), shortURLLen)
	restoredURL, err := stor.Get(stor.RemovePrefix(shortURL))
	require.NoError(t, err)
	assert.Equal(t, restoredURL, initURL)
}
