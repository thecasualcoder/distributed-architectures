package kafka_test

import (
	"fmt"
	"github.com/thecasualcoder/distributed-architectures/testutils/consul"
	"testing"
	"time"
)

func TestAddBroker(t *testing.T) {
	t.Run("in first test", func(t *testing.T) {
		t.Parallel()
		server := consul.NewDevServer(t)
		server.Start()
		defer server.Stop()
		fmt.Printf("First serving on %d\n", server.HTTPPort())
		time.Sleep(10 * time.Second)
	})

	t.Run("in second test", func(t *testing.T) {
		t.Parallel()
		server := consul.NewDevServer(t)
		server.Start()
		defer server.Stop()

		fmt.Printf("Second serving on %d\n", server.HTTPPort())

		time.Sleep(10 * time.Second)
	})
}
