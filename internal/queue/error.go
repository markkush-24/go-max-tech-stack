package queue

import "errors"

var ErrQueueFull = errors.New("queue is full")
var ErrQueueStopped = errors.New("queue is stopped")
