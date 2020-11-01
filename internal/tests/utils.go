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
	cleanRuntime  bool
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

// MetricsV2Writer writes metricsV2 to the terminal
func MetricsV2Writer(metrics *[]stats.MetricsV2) {
	tableWriter := table.NewWriter()
	tableWriter.SetOutputMirror(os.Stdout)
	tableWriter.AppendHeader(table.Row{"Run", "Memory", "CPU Total", "Disk"})
	for _, m := range *metrics {
		tableWriter.AppendRow(table.Row{m.Name, m.Mem, m.CPU, m.Disk})
	}
	tableWriter.Render()
}
