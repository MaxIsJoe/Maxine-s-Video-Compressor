package ffmpegWrapper

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func DetectGPUEncoder() string {
	cmd := exec.Command("ffmpeg", "-hide_banner", "-encoders")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	if strings.Contains(string(output), "h264_nvenc") {
		return "h264_nvenc"
	} else if strings.Contains(string(output), "h264_qsv") { // Intel QuickSync
		return "h264_qsv"
	} else if strings.Contains(string(output), "h264_amf") { // AMD
		return "h264_amf"
	} else {
		return ""
	}
}

func GetVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries",
		"format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get video duration: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %w", err)
	}
	return duration, nil
}

func IsValidVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv"
}
