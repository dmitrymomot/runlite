# System Statistics

## Getting Server Stats in Go

Use `github.com/shirou/gopsutil/v3` for comprehensive cross-platform system statistics.

### Installation

```bash
go get github.com/shirou/gopsutil/v3
```

### CPU Usage

```go
import "github.com/shirou/gopsutil/v3/cpu"

// Get CPU usage percentage (averaged across all cores)
cpuPercent, err := cpu.Percent(time.Second, false)
if err != nil {
    return err
}
// cpuPercent[0] contains the percentage

// Get per-core usage
perCoreCPU, err := cpu.Percent(time.Second, true)
```

### Memory Stats

```go
import "github.com/shirou/gopsutil/v3/mem"

vmStat, err := mem.VirtualMemory()
if err != nil {
    return err
}

totalMem := vmStat.Total      // Total RAM in bytes
freeMem := vmStat.Free        // Free RAM in bytes
availMem := vmStat.Available  // Available RAM (more accurate than Free)
usedPercent := vmStat.UsedPercent
```

### Disk Space

```go
import "github.com/shirou/gopsutil/v3/disk"

// For root filesystem
diskStat, err := disk.Usage("/")
if err != nil {
    return err
}

totalDisk := diskStat.Total      // Total disk space in bytes
freeDisk := diskStat.Free        // Free disk space in bytes
usedDisk := diskStat.Used        // Used disk space in bytes
usedPercent := diskStat.UsedPercent
```

### Example: Dashboard Stats

```go
type SystemStats struct {
    CPUPercent  float64
    MemTotal    uint64
    MemUsed     uint64
    MemPercent  float64
    DiskTotal   uint64
    DiskUsed    uint64
    DiskPercent float64
}

func GetSystemStats() (*SystemStats, error) {
    stats := &SystemStats{}

    // CPU
    cpuPercent, err := cpu.Percent(time.Second, false)
    if err != nil {
        return nil, err
    }
    stats.CPUPercent = cpuPercent[0]

    // Memory
    vmStat, err := mem.VirtualMemory()
    if err != nil {
        return nil, err
    }
    stats.MemTotal = vmStat.Total
    stats.MemUsed = vmStat.Used
    stats.MemPercent = vmStat.UsedPercent

    // Disk
    diskStat, err := disk.Usage("/")
    if err != nil {
        return nil, err
    }
    stats.DiskTotal = diskStat.Total
    stats.DiskUsed = diskStat.Used
    stats.DiskPercent = diskStat.UsedPercent

    return stats, nil
}
```

### Notes

- First call to `cpu.Percent()` may return 0; it needs time to measure
- Use `mem.Available` rather than `mem.Free` for more accurate available memory
- The library handles platform differences (Linux, macOS, Windows)
- All sizes are in bytes - convert to MB/GB as needed
