package infra

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type GPUStats struct {
	Available      bool
	UsedMemoryMB   int
	TotalMemoryMB  int
	UtilizationPct int
	Error          error
}

func GetGPUStats() GPUStats {
	stats := GPUStats{}

	cmd := exec.Command("nvidia-smi", "--query-gpu=memory.used,memory.total", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		stats.Available = false
		stats.Error = err
		return stats
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) != 2 {
		stats.Available = false
		stats.Error = fmt.Errorf("unexpected output format")
		return stats
	}

	used, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	total, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))

	if err1 != nil || err2 != nil {
		stats.Available = false
		stats.Error = fmt.Errorf("failed to parse memory values")
		return stats
	}

	stats.Available = true
	stats.UsedMemoryMB = used
	stats.TotalMemoryMB = total
	if total > 0 {
		stats.UtilizationPct = (used * 100) / total
	}

	return stats
}
