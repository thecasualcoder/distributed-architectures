package consul

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDevServer_List(t *testing.T) {
	server := NewDevServer(t)
	server.Start()
	defer server.Stop()

	err := server.Put("brokers/ids/", "1", []byte("{}"))
	assert.NoError(t, err)

	keys, err := server.List("brokers/ids/")

	assert.NoError(t, err)
	assert.Equal(t, []string{"1"}, keys)
}
