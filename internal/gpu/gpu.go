// Package gpu detects NVIDIA GPUs and their free VRAM by parsing nvidia-smi.
// It avoids cgo/NVML so the binary stays portable and cross-compilable.
package gpu

import (
	"encoding/json"
	"os/exec"
	"sort"
	"strings"
)

// GPU describes one NVIDIA device.
type GPU struct {
	Index     int    `json:"index"`
	Name      string `json:"name"`
	TotalMiB  int64  `json:"total_mib"`
	UsedMiB   int64  `json:"used_mib"`
	FreeMiB   int64  `json:"free_mib"`
}

// Detect returns all NVIDIA GPUs. On systems without nvidia-smi it returns
// an empty slice (callers should fall back to CPU / a single default).
func Detect() ([]GPU, error) {
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=index,name,memory.total,memory.used",
		"--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var gpus []GPU
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		g := GPU{Name: strings.TrimSpace(parts[1])}
		// index
		idx := 0
		//nolint:errcheck
		json.Unmarshal([]byte(strings.TrimSpace(parts[0])), &idx)
		g.Index = idx
		//nolint:errcheck
		json.Unmarshal([]byte(strings.TrimSpace(parts[2])), &g.TotalMiB)
		//nolint:errcheck
		json.Unmarshal([]byte(strings.TrimSpace(parts[3])), &g.UsedMiB)
		g.FreeMiB = g.TotalMiB - g.UsedMiB
		gpus = append(gpus, g)
	}
	return gpus, nil
}

// BestForVision returns the GPU with the most free VRAM, or nil if none.
// This is the GPU used for image/video/audio inference.
func BestForVision(gpus []GPU) *GPU {
	if len(gpus) == 0 {
		return nil
	}
	sorted := append([]GPU(nil), gpus...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].FreeMiB > sorted[j].FreeMiB })
	return &sorted[0]
}
