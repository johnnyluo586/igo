package log

import (
	"testing"
)

// Try each log level in decreasing order of priority.
func testConsoleCalls(bl *SimpleLogger) {
	bl.Emergency("emergency")
	bl.Alert("alert")
	bl.Critical("critical")
	bl.Error("error")
	bl.Warning("warning")
	bl.Notice("notice")
	bl.Informational("informational")
	bl.Debug("debug")
}

// Test console logging by visually comparing the lines being output with and
// without a log level specification.
func TestConsole(t *testing.T) {
	log1 := NewLogger(10000)
	log1.EnableFuncCallDepth(true)
	log1.SetLogger("console", "")
	testConsoleCalls(log1)

	log2 := NewLogger(100)
	log2.SetLogger("console", `{"level":3}`)
	testConsoleCalls(log2)
}

func BenchmarkConsole(b *testing.B) {
	log := NewLogger(10000)
	log.EnableFuncCallDepth(true)
	log.SetLogger("console", "")
	for i := 0; i < b.N; i++ {
		log.Debug("debug")
	}
}
