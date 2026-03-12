package rabbitmq

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"gophermind/pkg/contracts/events"
)

func TestNoopProducer(t *testing.T) {
	p := NewNoopProducer(zap.NewNop())
	err := p.PublishTask(context.Background(), events.TaskMessage{JobID: "j1"})
	require.NoError(t, err)
	err = p.PublishResult(context.Background(), events.ResultMessage{JobID: "j1"})
	require.NoError(t, err)
	err = p.PublishRetryTask(context.Background(), events.TaskMessage{JobID: "j1"})
	require.NoError(t, err)
	err = p.PublishDLQTask(context.Background(), events.TaskMessage{JobID: "j1"})
	require.NoError(t, err)
	require.NoError(t, p.Close())
}
