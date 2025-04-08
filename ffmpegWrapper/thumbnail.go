package ffmpegWrapper

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ExtractThumbnail(videoPath string) image.Image {
	thumbPath := strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + "_thumb.jpg"

	// Extract a frame at 1s
	cmd := exec.Command("ffmpeg", "-y", "-i", videoPath, "-ss", "00:00:01.000", "-vframes", "1", thumbPath)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Failed to extract thumbnail:", err)
		return nil
	}

	file, err := os.Open(thumbPath)
	if err != nil {
		fmt.Println("Failed to open thumbnail:", err)
		return nil
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		fmt.Println("Failed to decode image:", err)
		return nil
	}

	return img
}
