package utils

import (
	"runtime"
	"strings"
)

func GetGoRoutineID() string {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idStr := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	return idStr
}

func GetCurrentStackFrame() runtime.Frame {
	return getFrame(1)
}

func GetCallerStackFrame() runtime.Frame {
	return getFrame(2)
}

func getFrame(skipFrames int) runtime.Frame {
	targetFrameIdx := skipFrames + 2
	programCounters := make([]uintptr, targetFrameIdx+2)
	n := runtime.Callers(0, programCounters)
	frame := runtime.Frame{Function: "unknown"}

	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIdx := true, 0; more && frameIdx <= targetFrameIdx; frameIdx++ {
			var frameIdxCandidate runtime.Frame
			frameIdxCandidate, more = frames.Next()
			if frameIdx == targetFrameIdx {
				frame = frameIdxCandidate
				break
			}
		}
	}
	return frame
}
