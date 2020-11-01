package stats

import (
	"fmt"
	"strings"
	"time"

	"github.com/containerd/cgroups"
	v1 "github.com/containerd/cgroups/stats/v1"
	v2 "github.com/containerd/cgroups/v2"
	"github.com/jedib0t/go-pretty/list"
	"github.com/pkg/errors"
)

// CGroupsSampler represents Linux cgroups sampler
type CGroupsSampler struct {
	control      cgroups.Cgroup
	lastCPUUsage uint64
	lastCPUTime  time.Time
}

// CGroupSamplerV2 move too cgroups v2
type CGroupSamplerV2 struct {
	manager      *v2.Manager
	lastCPUUsage uint64
	lastCPUTime  time.Time
}

// NewCGroupsSamplerV2 creates a new
func NewCGroupsSamplerV2(path string) (*CGroupSamplerV2, error) {
	//manager, err := v2.LoadSystemd("systemd/system.slice", path)
	manager, err := v2.LoadSystemd("/systemd/system.slice/", path)

	if err != nil {
		return nil, err
	}
	return &CGroupSamplerV2{manager: manager}, nil
}

// ListProcesses list processes
func (s *CGroupSamplerV2) ListProcesses() error {
	prosses, err := s.manager.Procs(true)
	if err != nil {
		return err
	}
	l := list.NewWriter()
	for _, proc := range prosses {
		l.AppendItem(proc)
	}

	fmt.Println("length: ", len(prosses))
	listPrinter(l.Render(), "")
	return nil
}

func listPrinter(list, prefix string) {
	for _, line := range strings.Split(list, "\n") {
		fmt.Printf("%s%s\n", prefix, line)
	}
	fmt.Println()
}

// ListGroups list Groups
func (s *CGroupSamplerV2) ListGroups() error {
	controllers, err := s.manager.Controllers()
	if err != nil {
		return err
	}
	l := list.NewWriter()
	for _, ctrl := range controllers {
		l.AppendItem(ctrl)
	}
	return nil
}

// NewCGroupsSampler creates a stats sampler from existing control group
func NewCGroupsSampler(path string) (*CGroupsSampler, error) {
	control, err := cgroups.Load(reportControllers, cgroups.StaticPath(path))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load cgroup: '%s'", path)
	}

	return &CGroupsSampler{control: control}, nil
}

// reportControllers returns v1 controllers only required for measuring resource usage
func reportControllers() ([]cgroups.Subsystem, error) {
	v1, err := cgroups.V1()
	if err != nil {
		return nil, err
	}

	var out []cgroups.Subsystem
	for _, sub := range v1 {
		if sub.Name() == cgroups.Memory || sub.Name() == cgroups.Cpuacct {
			out = append(out, sub)
		}
	}

	return out, nil
}

// Sample gets a process metrics from control cgroup
func (s *CGroupSamplerV2) Sample(name string) (*Metrics, error) {
	metrics, err := s.manager.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get metrics from cgroup")
	}

	memStat := metrics.Memory

	mem := (memStat.Usage) / bytesInMiB
	cpu := metrics.CPU.UsageUsec * 1000

	now := time.Now()

	cpuUsage := float64(cpu - s.lastCPUUsage) // float64(now.Sub(s.lastCPUTime).Nanoseconds())
	cpuPercent := cpuUsage / float64(now.Sub(s.lastCPUTime).Nanoseconds())

	s.lastCPUUsage = cpu
	s.lastCPUTime = now

	return &Metrics{
		Name:       name,
		Mem:        mem,
		CPU:        cpuUsage,
		CPUPercent: cpuPercent,
	}, nil
}

// Sample gets a process metrics from control cgroup
func (s *CGroupsSampler) Sample(name string) (*Metrics, error) {
	metrics, err := s.control.Stat(cgroups.IgnoreNotExist)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get metrics from cgroup")
	}

	memStat := metrics.Memory

	// memory.memsw.usage_in_bytes (current usage for memory+swap) + memory.kmem.usage_in_bytes (current
	// kernel memory allocation)
	mem := (memStat.TotalRSS) / bytesInMiB
	cpu := metrics.CPU.Usage.Total

	now := time.Now()

	cpuUsage := float64(cpu - s.lastCPUUsage) // float64(now.Sub(s.lastCPUTime).Nanoseconds())
	cpuPercent := cpuUsage / float64(now.Sub(s.lastCPUTime).Nanoseconds())

	s.lastCPUUsage = cpu
	s.lastCPUTime = now

	return &Metrics{
		Name:       name,
		Mem:        mem,
		CPU:        cpuUsage,
		CPUPercent: cpuPercent,
	}, nil
}

// Stat gets the stats
func (s *CGroupsSampler) Stat() (*v1.Metrics, error) {
	return s.control.Stat()
}
