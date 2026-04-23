package timing

import (
	"fmt"
	"strings"
	"time"

	"github.com/cmd184psu/alfredo"
)

type TimerStruct struct {
	StartTime int64
	EndTime   int64
	Task      string
	LogFile   string
}

func (timer *TimerStruct) WithTask(t string) *TimerStruct {
	timer.Task = t
	return timer
}

func (timer *TimerStruct) WithLogFile(logFile string) *TimerStruct {
	timer.LogFile = logFile
	return timer
}

func NewTimer() *TimerStruct {
	return &TimerStruct{}
}

func (timer *TimerStruct) Reset() *TimerStruct {
	timer.StartTime = 0
	timer.EndTime = 0
	return timer
}

func (timer *TimerStruct) GetEllapsedms() int64 {
	if timer.EndTime == 0 {
		return time.Now().UnixMilli() - timer.StartTime
	}
	return timer.EndTime - timer.StartTime
}

func (timer *TimerStruct) StartTiming() *TimerStruct {
	timer.StartTime = time.Now().Unix()
	return timer
}

func (timer *TimerStruct) EndTiming() *TimerStruct {
	timer.EndTime = time.Now().Unix()
	return timer
}

func (timer *TimerStruct) GetEllapsedseconds() int64 {
	if timer.EndTime == 0 {
		return time.Now().Unix() - timer.StartTime
	}
	return timer.EndTime - timer.StartTime
}

func (timer *TimerStruct) GenerateTimingContent() string {
	ellapsedSeconds := timer.GetEllapsedseconds()
	timingContent := make([]string, 0)
	if timer.EndTime == 0 {
		//timer is still running, get current ellapsed time
		s := ""
		if len(timer.Task) > 0 {
			s = fmt.Sprintf(" (%s)", timer.Task)
		}

		return fmt.Sprintf("Timer%s is still running, current ellapsed time: ( %d seconds ) %s", s, ellapsedSeconds, alfredo.HumanReadableTimeStamp(ellapsedSeconds))

	} else {

		if len(timer.Task) > 0 {
			timingContent = append(timingContent, fmt.Sprintf("Activity: %s", timer.Task))
		}
		timingContent = append(timingContent, fmt.Sprintf("Start time: %s ( %d seconds )\n", alfredo.HumanReadableTimeStamp(timer.StartTime), timer.StartTime))
		timingContent = append(timingContent, fmt.Sprintf("End time: %s ( %d seconds )\n", alfredo.HumanReadableTimeStamp(timer.EndTime), timer.EndTime))
		timingContent = append(timingContent, fmt.Sprintf("Duration: %s ( %d seconds )\n", alfredo.HumanReadableSeconds(timer.EndTime-timer.StartTime), timer.EndTime-timer.StartTime))

	}
	return strings.Join(timingContent, "\n")
}
func (timer *TimerStruct) RecordTime() error {
	if len(timer.LogFile) == 0 {
		return fmt.Errorf("log file not set for timer, cannot record time")
	}
	content := timer.GenerateTimingContent()
	return alfredo.WriteStringToFile(timer.LogFile, content)
}
