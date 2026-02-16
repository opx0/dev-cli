package infra

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// GPUProvider is the interface for detecting and querying GPU hardware.
type GPUProvider interface {
	GetStats() GPUStats
	Vendor() string
}

// GPUStats contains GPU utilization data.
type GPUStats struct {
	Available      bool
	Vendor         string
	UsedMemoryMB   int
	TotalMemoryMB  int
	UtilizationPct int
	Temperature    int
	Error          error
}

// DetectGPU detects the available GPU provider (NVIDIA, AMD, Apple, or none).
func DetectGPU() GPUProvider {
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		return &NvidiaGPUProvider{}
	}

	if _, err := exec.LookPath("rocm-smi"); err == nil {
		return &AMDGPUProvider{}
	}

	if runtime.GOOS == "darwin" {
		return &AppleGPUProvider{}
	}

	return &NoGPUProvider{}
}

// --- NVIDIA GPU ---

type NvidiaGPUProvider struct{}

func (p *NvidiaGPUProvider) Vendor() string {
	return "nvidia"
}

func (p *NvidiaGPUProvider) GetStats() GPUStats {
	stats := GPUStats{Vendor: "nvidia"}

	cmd := exec.Command("nvidia-smi",
		"--query-gpu=memory.used,memory.total,utilization.gpu,temperature.gpu",
		"--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		stats.Available = false
		stats.Error = err
		return stats
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 2 {
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

	if len(parts) >= 3 {
		if util, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
			stats.UtilizationPct = util
		}
	}

	if len(parts) >= 4 {
		if temp, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil {
			stats.Temperature = temp
		}
	}

	return stats
}

// --- AMD GPU ---

type AMDGPUProvider struct{}

func (p *AMDGPUProvider) Vendor() string {
	return "amd"
}

func (p *AMDGPUProvider) GetStats() GPUStats {
	stats := GPUStats{Vendor: "amd"}

	cmd := exec.Command("rocm-smi", "--showmeminfo", "vram", "--csv")
	output, err := cmd.Output()
	if err != nil {
		stats.Available = false
		stats.Error = err
		return stats
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "GPU") && strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				if used, err := parseMemoryMB(strings.TrimSpace(parts[1])); err == nil {
					stats.UsedMemoryMB = used
				}
				if total, err := parseMemoryMB(strings.TrimSpace(parts[2])); err == nil {
					stats.TotalMemoryMB = total
				}
			}
		}
	}

	tempCmd := exec.Command("rocm-smi", "-t", "--csv")
	tempOutput, err := tempCmd.Output()
	if err == nil {
		lines := strings.Split(string(tempOutput), "\n")
		for _, line := range lines {
			if strings.Contains(line, "GPU") {
				parts := strings.Split(line, ",")
				if len(parts) >= 2 {
					if temp, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
						stats.Temperature = int(temp)
					}
				}
			}
		}
	}

	stats.Available = stats.TotalMemoryMB > 0
	if stats.TotalMemoryMB > 0 {
		stats.UtilizationPct = (stats.UsedMemoryMB * 100) / stats.TotalMemoryMB
	}

	return stats
}

func parseMemoryMB(s string) (int, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")

	if strings.HasSuffix(s, "GB") || strings.HasSuffix(s, "G") {
		s = strings.TrimSuffix(strings.TrimSuffix(s, "GB"), "G")
		if val, err := strconv.ParseFloat(s, 64); err == nil {
			return int(val * 1024), nil
		}
	}
	if strings.HasSuffix(s, "MB") || strings.HasSuffix(s, "M") {
		s = strings.TrimSuffix(strings.TrimSuffix(s, "MB"), "M")
		if val, err := strconv.ParseFloat(s, 64); err == nil {
			return int(val), nil
		}
	}
	if val, err := strconv.Atoi(s); err == nil {
		return val, nil
	}
	return 0, fmt.Errorf("cannot parse: %s", s)
}

// --- Apple GPU ---

type AppleGPUProvider struct{}

func (p *AppleGPUProvider) Vendor() string {
	return "apple"
}

func (p *AppleGPUProvider) GetStats() GPUStats {
	stats := GPUStats{Vendor: "apple"}

	cmd := exec.Command("system_profiler", "SPDisplaysDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		stats.Available = false
		stats.Error = err
		return stats
	}

	if strings.Contains(string(output), "Apple") || strings.Contains(string(output), "M1") ||
		strings.Contains(string(output), "M2") || strings.Contains(string(output), "M3") {
		stats.Available = true
	}

	return stats
}

// --- No GPU fallback ---

type NoGPUProvider struct{}

func (p *NoGPUProvider) Vendor() string {
	return "none"
}

func (p *NoGPUProvider) GetStats() GPUStats {
	return GPUStats{
		Available: false,
		Vendor:    "none",
		Error:     fmt.Errorf("no supported GPU detected"),
	}
}

// GetGPUStats is a convenience function that auto-detects GPU and returns stats.
func GetGPUStats() GPUStats {
	return DetectGPU().GetStats()
}
