package importer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

type shadowRecorder struct {
	calls    [][]int
	received map[int]int
	failNext bool
}

func (r *shadowRecorder) ReceiveBatch(_ context.Context, b ingest.Batch) ([]ingest.Result, error) {
	ordinals := make([]int, 0, len(b.Items))
	for _, item := range b.Items {
		ordinals = append(ordinals, item.Ordinal)
	}
	r.calls = append(r.calls, ordinals)
	if r.failNext {
		r.failNext = false
		return nil, errors.New("shadow failure")
	}
	for _, item := range b.Items {
		r.received[item.Ordinal]++
	}
	return make([]ingest.Result, len(b.Items)), nil
}

// TestShadowFileBatching covers bounded pages, ignored ordinals and a failed page retried by reimport.
func TestShadowFileBatching(t *testing.T) {
	recorder := &shadowRecorder{received: make(map[int]int), failNext: true}
	run := func() {
		shadow := &fileShadow{receiver: recorder, ordinal: -1}
		ctx := context.WithValue(context.Background(), shadowKey{}, shadow)
		for range shadowBatchSize + 3 {
			seen(ctx)
			if shadow.ordinal == 1 {
				continue
			}
			captureFileListen(ctx, ingest.Snapshot{Artist: "A", Title: "T"}, time.Unix(100, 0))
		}
		flushFileShadow(ctx)
		flushFileShadow(ctx)
	}
	run()
	require.Len(t, recorder.calls, 2)
	require.Len(t, recorder.calls[0], shadowBatchSize)
	require.Equal(t, []int{shadowBatchSize + 1, shadowBatchSize + 2}, recorder.calls[1])
	require.Len(t, recorder.received, 2)
	run()
	require.Equal(t, recorder.calls[0], recorder.calls[2])
	require.Equal(t, recorder.calls[1], recorder.calls[3])
	require.Len(t, recorder.received, shadowBatchSize+2)
	require.Zero(t, recorder.received[1])
}
