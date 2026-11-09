package audit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordStripsNewlines(t *testing.T) {
	recorder := NewMemoryRecorder()
	recorder.Record(Event{
		Action:   "disable",
		Operator: "ops\nFORGED-LOG-LINE",
		Detail:   "line\r\nbreak",
		Result:   "ok",
	})
	events := recorder.List()
	require.Len(t, events, 1)
	require.NotContains(t, events[0].Operator, "\n")
	require.NotContains(t, events[0].Detail, "\n")
	require.Contains(t, events[0].Operator, "FORGED-LOG-LINE")
}
