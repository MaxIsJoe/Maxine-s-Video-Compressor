package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"os/exec"
	"runtime"
)

func ShowFFmpegInstructions(win fyne.Window) {
	var installCmd *exec.Cmd

	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("choco"); err == nil {
			installCmd = exec.Command("choco", "install", "ffmpeg", "-y")
		} else if _, err := exec.LookPath("scoop"); err == nil {
			installCmd = exec.Command("scoop", "install", "ffmpeg")
		} else {
			dialog.ShowInformation("FFmpeg Not Found", "FFmpeg is required for this app to work.\n\nVisit: https://ffmpeg.org/download.html\n\nInstall it, then restart the app.", win)
			return
		}
	} else {
		installCmd = exec.Command("sh", "-c", "sudo apt-get update && sudo apt-get install -y ffmpeg")
	}

	err := installCmd.Run()
	if err != nil {
		dialog.ShowInformation("FFmpeg Not Found", "FFmpeg is required for this app to work.\n\nVisit: https://ffmpeg.org/download.html\n\nInstall it, then restart the app.", win)
	} else {
		dialog.ShowInformation("FFmpeg Installed", "FFmpeg has been successfully installed. Please restart the app.", win)
	}
}
