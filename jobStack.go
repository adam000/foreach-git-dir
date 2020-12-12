package main

import (
	"context"
	"time"
)

type JobStack struct {
	stack  []string
	inbox  chan string
	outbox chan string
}

func (j *JobStack) Run(ctx context.Context) {
	for {
		// Send messages as long as possible (while someone is waiting)
	Send:
		for len(j.stack) > 0 {
			select {
			case <-ctx.Done():
				panic("Context cancelled while there are jobs on the stack")
			case j.outbox <- j.stack[len(j.stack)-1]:
				j.stack = j.stack[:len(j.stack)-1]
			default:
				break Send
			}
		}
		// Receive messages, locked for 5ms to avoid lots of thrash
	Receive:
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-j.inbox:
				j.stack = append(j.stack, job)
			case <-time.After(5 * time.Millisecond):
				break Receive
			}
		}
	}
}

func (j *JobStack) Close() {
	close(j.outbox)
	close(j.inbox)
}
