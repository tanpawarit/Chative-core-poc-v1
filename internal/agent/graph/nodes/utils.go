package nodes

import (
	"github.com/Chative-core-poc-v1/server/internal/agent/model"
)

const DefaultMaxToolCalls = 10

// ===== Small helpers to keep handlers simple/readable =====
// normalizeMaxToolCalls returns a sane default when the provided value is invalid.
func normalizeMaxToolCalls(n int) int {
	if n <= 0 {
		return DefaultMaxToolCalls
	}
	return n
}

// checkAndMarkToolLimit evaluates whether another tool call would exceed the
// limit and, if so, marks the state accordingly. Returns true when marked now.
func checkAndMarkToolLimit(state *model.AppState, max int) bool {
	max = normalizeMaxToolCalls(max)
	if !state.ToolCallLimitReached && state.ToolCallCount >= max {
		state.ToolCallLimitReached = true
		return true
	}
	return false
}

// incrementToolCallAndCheck increments the count and marks the state if it
// exceeds the limit after incrementing. Returns true when exceeded.
func incrementToolCallAndCheck(state *model.AppState, max int) bool {
	max = normalizeMaxToolCalls(max)
	state.ToolCallCount++
	if state.ToolCallCount > max {
		state.ToolCallLimitReached = true
		return true
	}
	return false
}
