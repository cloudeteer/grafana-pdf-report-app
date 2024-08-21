package worker_test

import (
	"context"
	"testing"

	"github.com/cloudeteer/grafana-pdf-report-app/pkg/plugin/worker"
	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	pool := worker.New(ctx, 1)

	resultCh := make(chan int, 10)

	for i := range 10 {
		pool.Do(func() {
			resultCh <- i
		})
	}

	for i := range 10 {
		assert.Equal(t, i, <-resultCh)
	}
}
