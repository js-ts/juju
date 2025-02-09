// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver

import (
	"github.com/juju/errors"

	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/core/raft/queue"
	"github.com/juju/juju/worker/raft"
)

type Logger interface {
	Debugf(string, ...interface{})
	Tracef(string, ...interface{})
	IsTraceEnabled() bool
}

// Queue is a blocking queue to guard access and to serialize raft applications,
// allowing for client side backoff.
type Queue interface {
	// Enqueue will add an operation to the queue. As this is a blocking queue, any
	// additional enqueue operations will block and wait for subsequent operations
	// to be completed.
	// The design of this is to ensure that people calling this will have to
	// correctly handle backing off from enqueueing.
	Enqueue(queue.Operation) error
}

// raftMediator encapsulates raft related capabilities to the facades.
type raftMediator struct {
	queue  Queue
	logger Logger
}

// ApplyLease attempts to apply the command on to the raft FSM. It only takes a
// command and enqueues that against the raft instance. If the raft instance is
// already processing a application, then back pressure is applied to the
// caller and a ErrEnqueueDeadlineExceeded will be sent. It's up to the caller
// to retry or drop depending on how the retry algorithm is implemented.
func (m *raftMediator) ApplyLease(cmd []byte) error {
	if m.logger.IsTraceEnabled() {
		m.logger.Tracef("Applying Lease with command %s", string(cmd))
	}

	err := m.queue.Enqueue(queue.Operation{
		Commands: [][]byte{cmd},
	})

	switch {
	case err == nil:
		return nil

	case raft.IsNotLeaderError(err):
		// Lift the worker NotLeaderError into the apiserver NotLeaderError. Ensure
		// the correct boundaries.
		leaderErr := errors.Cause(err).(*raft.NotLeaderError)
		m.logger.Tracef("Not currently the leader, go to %v %v", leaderErr.ServerAddress(), leaderErr.ServerID())
		return apiservererrors.NewNotLeaderError(leaderErr.ServerAddress(), leaderErr.ServerID())

	case queue.IsDeadlineExceeded(err):
		// If the deadline is exceeded, get original callee to handle the
		// timeout correctly.
		return apiservererrors.NewDeadlineExceededError(err.Error())

	}
	return errors.Trace(err)
}
