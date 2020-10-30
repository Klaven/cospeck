package tests

import (
	"os"

	"github.com/Klaven/cospeck/internal/stats"
	"github.com/jedib0t/go-pretty/table"
)

// TestFlags is a struct that represents the flags that can be passed to flags
type TestFlags struct {
	Tests         string
	OCIRuntime    string
	CGroupPath    string
	PodConfigFile string
	Threads       int
}

// MetricsWriter writes metrics to the terminal
func MetricsWriter(metrics *[]stats.Metrics) {
	tableWriter := table.NewWriter()
	tableWriter.SetOutputMirror(os.Stdout)
	tableWriter.AppendHeader(table.Row{"Run", "Memory", "CPU Total", "CPU %"})
	for _, m := range *metrics {
		tableWriter.AppendRow(table.Row{m.Name, m.Mem, m.CPU, m.CPUPercent})
	}
	tableWriter.Render()
}
