package alfredo

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type TaskStatusType int64

const (
	TSTNone TaskStatusType = iota
	TSTQuened
	TSTRunning
	TSTCompleted
)

type TaskStruct struct {
	ID          int           `json:"id"`
	Description string        `json:"description"`
	Command     string        `json:"command"`
	Status      string        `json:"status"`
	Output      []string      `json:"output,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	ctx         context.Context
	cancel      context.CancelFunc
	sigChan     chan os.Signal
}

func (tst TaskStatusType) String() string {
	switch tst {
	case TSTQuened:
		return "queued"
	case TSTRunning:
		return "running"
	case TSTCompleted:
		return "completed"
	}
	return "none"
}

func getTaskStatusTypeOf(tst string) TaskStatusType {
	if strings.ToLower(tst) == TSTQuened.String() {
		return TSTQuened
	}
	if strings.ToLower(tst) == TSTRunning.String() {
		return TSTRunning
	}
	if strings.ToLower(tst) == TSTCompleted.String() {
		return TSTCompleted
	}
	return TSTNone
}

func (ts *TaskStruct) Init() {
	//setup context and cancel function
	ts.ctx, ts.cancel = context.WithCancel(context.Background())

	//add the defer in the function running the task
	//defer ts.cancel()

	//setup back channel to catch signal
	ts.sigChan = make(chan os.Signal, 1)
	signal.Notify(ts.sigChan, syscall.SIGINT, syscall.SIGTERM)

}
