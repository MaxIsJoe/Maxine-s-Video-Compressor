package main

import (
	"bufio"
	"fmt"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"bytes"
	"image"
	"image/jpeg"

	"fyne.io/fyne/v2/canvas"
)

var selectedFile string
var lastFolder string
var useGPU bool = true

var application fyne.App
var window fyne.Window
var logWindow fyne.Window
var baseSize fyne.Size = fyne.NewSize(555, 455)

var logContent *widget.Entry
var logScrollContainer *container.Scroll

var sizeEntry *widget.Entry
var statusLabel *widget.Label
var fileLabel *widget.Label
var compressBtn *widget.Button
var estimateLabel *widget.Label
var stripAudioCheckbox *widget.Check
var dndFileList binding.StringList = binding.NewStringList()
var progressBar *widget.ProgressBar

var thumbnailImage *canvas.Image

func main() {
	application = app.New()
	window = application.NewWindow("Maxine's Video Compressor")
	window.Resize(baseSize)

	window.SetIcon(resourceIconPng)

	if !ffmpegAvailable() {
		showFFmpegInstructions(window)
	}

	sizeContainer := createSizeEntry()

	statusLabel = widget.NewLabel("")
	statusLabel.Alignment = fyne.TextAlignCenter

	thumbnailImage = &canvas.Image{}
	thumbnailImage.FillMode = canvas.ImageFillContain

	compressBtn = widget.NewButton("Compress", OnCompressButtonPressed)
	compressBtn.Disable() // Initially disabled

	fileLabel = widget.NewLabel("No file selected")
	selectBtn := widget.NewButton("Select Video", FileSelectionInit)

	gpuCheckbox := widget.NewCheck("Use GPU", func(checked bool) {
		useGPU = checked
	})

	gpuCheckbox.SetChecked(true)
	useGPU = true

	stripAudioCheckbox = widget.NewCheck("Extract Audio", nil)

	optionsLabel := widget.NewLabel("Options")
	optionsLabel.TextStyle = fyne.TextStyle{Underline: true, Bold: true}
	ffmpegOptions := container.NewGridWrap(fyne.Size{Width: baseSize.Width / 6, Height: baseSize.Height / 12},
		gpuCheckbox, stripAudioCheckbox)

	estimateLabel = widget.NewLabel("")
	estimateLabel.Alignment = fyne.TextAlignCenter

	progressBar = widget.NewProgressBar()
	progressBar.Hide()

	content := container.NewVBox(
		fileLabel,
		thumbnailImage,
		selectBtn,
		sizeContainer,
		compressBtn,
		statusLabel,
		estimateLabel,
		progressBar,
		optionsLabel,
		ffmpegOptions,
	)

	// Create footer
	footer := createFooter()

	// Add footer to the main content
	mainContent := container.NewBorder(nil, footer, nil, nil, content)

	window.SetOnDropped(OnDroppedFiles)

	window.SetContent(mainContent)
	window.ShowAndRun()
}

func OnDroppedFiles(pos fyne.Position, URIs []fyne.URI) {
	dndFileList = binding.NewStringList()
	for _, file := range URIs {
		if isValidVideoFile(file.Path()) {
			dndFileList.Append(file.Path())
		}
	}
	if dndFileList.Length() > 0 {
		showFileListDialog()
	}
}

func showFileListDialog() {
	customConfirm := dialog.NewCustomConfirm("Files to Compress", "Compress", "Cancel", createFileListContainer(), func(ok bool) {
		if ok {
			OnCompressButtonPressed()
		}
	}, window)
	customConfirm.Resize(fyne.NewSize(400, 300)) // Set the desired size of the customConfirm
	customConfirm.Show()
}

func createFileListContainer() fyne.CanvasObject {
	list := widget.NewListWithData(dndFileList, func() fyne.CanvasObject {
		return widget.NewLabel("")
	}, func(i binding.DataItem, o fyne.CanvasObject) {
		o.(*widget.Label).Bind(i.(binding.String))
	})

	sizeEntryDialog := widget.NewEntry()
	sizeEntryDialog.SetPlaceHolder("Enter target size (MB) - 9.5 for Discord.")
	sizeEntryDialog.SetText("9.5 MB")

	sizeEntryDialog.OnChanged = func(s string) {
		sizeEntry.SetText(s)
	}

	listBorder := container.NewBorder(nil, container.NewVBox(widget.NewLabel("Target Size (MB):"), sizeEntryDialog), nil, nil, list)

	return listBorder
}

func isValidVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv"
}

func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./.video_compressor_config"
	}
	return filepath.Join(homeDir, ".video_compressor_config")
}

func saveLastUsedFolder(path string) {
	configPath := getConfigPath()
	folder := filepath.Dir(path)
	os.WriteFile(configPath, []byte(folder), 0644)
}

func loadLastUsedFolder() string {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "" // fallback to default if not found
	}
	return string(data)
}

func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func showFFmpegInstructions(win fyne.Window) {
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

func compressVideo(inputPath string, targetSizeMB float64, outputPath string, onComplete func(error)) {
	go func() {
		duration, proberr := getVideoDuration(inputPath)
		if proberr != nil || duration <= 0 {
			onComplete(fmt.Errorf("could not determine video duration"))
			return
		}
		targetBitrate := int((float64(targetSizeMB) * 8192) / duration)

		cmdArgs := []string{
			"-i", inputPath,
			"-b:v", fmt.Sprintf("%dk", targetBitrate),
			"-bufsize", fmt.Sprintf("%dk", targetBitrate),
		}

		if useGPU {
			gpuEncoder := detectGPUEncoder()
			if gpuEncoder != "" {
				cmdArgs = append(cmdArgs, "-c:v", gpuEncoder)
			}
		}

		if stripAudioCheckbox.Checked {
			audioOutputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "_audio.mp3"
			audioCmd := exec.Command("ffmpeg", "-i", inputPath, "-q:a", "0", "-map", "a", audioOutputPath)
			if err := audioCmd.Run(); err != nil {
				onComplete(fmt.Errorf("failed to extract audio: %w", err))
				return
			}
		}

		cmdArgs = append(cmdArgs, outputPath)
		cmd := exec.Command("ffmpeg", cmdArgs...)

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			onComplete(err)
			return
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			onComplete(err)
			return
		}

		if err := cmd.Start(); err != nil {
			onComplete(err)
			return
		}

		// Display logs in a new window
		logWindow, logContent = showLogWindow()

		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				logContent.SetText(logContent.Text + scanner.Text() + "\n")
				logScrollContainer.ScrollToBottom() //not working?
				logScrollContainer.Refresh()
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				logContent.SetText(logContent.Text + scanner.Text() + "\n")
				logScrollContainer.ScrollToBottom()
				logScrollContainer.Refresh()
			}
		}()

		err = cmd.Wait()
		onComplete(err)
	}()
}

func detectGPUEncoder() string {
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

func getVideoDuration(videoPath string) (float64, error) {
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

func showLogWindow() (fyne.Window, *widget.Entry) {
	logWindow = application.NewWindow("Compression Logs")
	logWindow.Resize(fyne.NewSize(600, 400))

	logContent = widget.NewMultiLineEntry()

	logScrollContainer = container.NewScroll(logContent)
	logWindow.SetContent(logScrollContainer)
	logWindow.Show()

	return logWindow, logContent
}

func updateEstimate() {
	if selectedFile == "" {
		estimateLabel.SetText("")
		return
	}

	duration, err := getVideoDuration(selectedFile)
	if err != nil || duration <= 0 {
		estimateLabel.SetText("Failed to get duration")
		return
	}

	targetSizeStr := strings.TrimSuffix(sizeEntry.Text, " MB")
	targetSizeMB, err := strconv.ParseFloat(targetSizeStr, 64)
	if err != nil || targetSizeMB <= 0 {
		estimateLabel.SetText("Invalid size")
		return
	}

	targetBitrate := (targetSizeMB * 8192) / duration // kbps
	estimateLabel.SetText(fmt.Sprintf("Estimated Bitrate: %.0f kbps", targetBitrate))
}

func createSizeEntry() *fyne.Container {
	sizeEntry = widget.NewEntry()
	sizeEntry.SetPlaceHolder("Enter target size (MB) - 9.5 for Discord.")
	sizeEntry.SetText("9.5 MB")
	sizeEntry.Validator = func(s string) error {
		updateEstimate()
		if _, err := strconv.ParseFloat(strings.TrimSuffix(s, " MB"), 64); err != nil {
			compressBtn.Disable()
			return fmt.Errorf("invalid number")
		}
		if selectedFile != "" {
			compressBtn.Enable()
		}
		return nil
	}

	incrementBtn := widget.NewButton("+", func() {
		updateEstimate()
		if sizeEntry.Text == "" {
			sizeEntry.SetText("0.5 MB")
		}
		value, err := strconv.ParseFloat(strings.TrimSuffix(sizeEntry.Text, " MB"), 64)
		if err == nil {
			value += 0.5
			sizeEntry.SetText(fmt.Sprintf("%.1f MB", value))
		}
	})

	decrementBtn := widget.NewButton("-", func() {
		updateEstimate()
		if sizeEntry.Text == "" {
			sizeEntry.SetText("0.5 MB")
		}
		value, err := strconv.ParseFloat(strings.TrimSuffix(sizeEntry.Text, " MB"), 64)
		if err == nil && value > 0.5 {
			value -= 0.5
			sizeEntry.SetText(fmt.Sprintf("%.1f MB", value))
		}
	})

	/*
		prefilledSizes := []string{"5 MB", "9.5 MB", "14.5 MB", "19.5 MB", "24.5 MB", "29.5 MB", "34.5 MB", "39.5 MB", "44.5 MB"}
		menuItems := make([]*fyne.MenuItem, len(prefilledSizes))
		for i, sizeElement := range prefilledSizes {
			menuItems[i] = fyne.NewMenuItem(sizeElement, func() {
				sizeEntry.SetText(sizeElement)
			})
		}

		contextMenu := fyne.NewMenu("Set Size", menuItems...)
		sizeEntry.TappedSecondary()
		widget.ShowPopUpMenuAtPosition(contextMenu, fyne.CurrentApp().Driver().CanvasForObject(sizeEntry), pe.Position)
	*/

	finalResult := container.NewBorder(nil, nil, nil, container.NewHBox(decrementBtn, incrementBtn), sizeEntry)
	return finalResult
}

func FileSelectionInit() {
	lastFolder = loadLastUsedFolder()
	fd := dialog.NewFileOpen(OnFileSelectionOpened, window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".mp4", ".mov", ".avi", ".mkv"}))
	if lastFolder != "" {
		uri, err := storage.ParseURI("file://" + lastFolder)
		if err == nil {
			listable, err := storage.ListerForURI(uri)
			if err == nil {
				fd.SetLocation(listable)
			}
		}
	}
	fd.Show()
}

func OnFileSelectionOpened(reader fyne.URIReadCloser, err error) {
	if err == nil && reader != nil {
		selectedFile = reader.URI().Path()
		fileLabel.SetText(filepath.Base(selectedFile))
		compressBtn.Enable() // Enable when a file is selected
		saveLastUsedFolder(selectedFile)
		img := extractThumbnail(selectedFile)
		if img != nil {
			buf := new(bytes.Buffer)
			if err := jpeg.Encode(buf, img, nil); err == nil {
				res := fyne.NewStaticResource("thumbnail.jpg", buf.Bytes())
				thumbnailImage.Resource = res
				thumbnailImage.SetMinSize(fyne.NewSize(200, 200))
				thumbnailImage.Refresh()
			}
		}
		updateEstimate()
	}
}

func OnCompressButtonPressed() {
	dragAndDropAmount := dndFileList.Length()
	if selectedFile == "" && dragAndDropAmount == 0 {
		dialog.ShowError(fmt.Errorf("no video selected"), window)
		return
	}

	targetSizeStr := strings.TrimSuffix(sizeEntry.Text, " MB")
	targetSize, err := strconv.ParseFloat(targetSizeStr, 64)
	println(targetSize)
	if err != nil || targetSize <= 0 {
		dialog.ShowError(fmt.Errorf("invalid target size"), window)
		return
	}

	if dragAndDropAmount > 0 {
		if CompressMultipleFilesInDragAndDropList(targetSize) {
			return
		}
	} else {
		outputPath := strings.TrimSuffix(selectedFile, filepath.Ext(selectedFile)) + "_compressed.mp4"
		statusLabel.SetText("Compressing...")

		compressVideo(selectedFile, targetSize, outputPath, func(err error) {
			if err != nil {
				statusLabel.SetText("Compression failed")
				dialog.ShowError(err, window)
			} else {
				statusLabel.SetText("Done: " + filepath.Base(outputPath))
				dialog.ShowInformation("Success", "Video compressed successfully!", window)
			}
		})
	}
}

func CompressMultipleFilesInDragAndDropList(targetSize float64) bool {
	index := 0.0
	failCount := 0
	files, err := dndFileList.Get()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to get file list"), window)
		return true
	}
	progressBar.Show()
	progressBar.Max = float64(len(files))
	for _, file := range files {
		progressBar.SetValue(index)
		selectedFile = file
		fileLabel.SetText(filepath.Base(selectedFile))
		saveLastUsedFolder(selectedFile)
		img := extractThumbnail(selectedFile)
		if img != nil {
			buf := new(bytes.Buffer)
			if err := jpeg.Encode(buf, img, nil); err == nil {
				res := fyne.NewStaticResource("thumbnail.jpg", buf.Bytes())
				thumbnailImage.Resource = res
				thumbnailImage.SetMinSize(fyne.NewSize(200, 200))
				thumbnailImage.Refresh()
			}
		}
		outputPath := strings.TrimSuffix(selectedFile, filepath.Ext(selectedFile)) + "_compressed.mp4"
		statusLabel.SetText("Compressing...")

		compressVideo(selectedFile, targetSize, outputPath, func(err error) {
			if err != nil {
				failCount++
			}
		})
		index++
	}
	if failCount > 0 {
		statusLabel.SetText(fmt.Sprintf("Compression failed for %d files", failCount))
		dialog.ShowError(fmt.Errorf("compression failed for %d files", failCount), window)
	} else {
		statusLabel.SetText("Done: " + filepath.Base(selectedFile))
		dialog.ShowInformation("Success", "All videos compressed successfully!", window)
	}
	progressBar.Hide()
	dndFileList = binding.NewStringList()
	return false
}

func extractThumbnail(videoPath string) image.Image {
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

func createFooter() *fyne.Container {
	buyMeACoffeeURL, _ := url.Parse("https://www.maxisjoe.xyz/maxfund")
	buyMeACoffeeLink := widget.NewHyperlink("buy me a coffee", buyMeACoffeeURL)

	dollarIcon := widget.NewIcon(theme.ContentAddIcon())
	dollarIcon.SetResource(theme.NewThemedResource(resourceKofiIconPng))

	footer := container.NewHBox(
		dollarIcon,
		buyMeACoffeeLink,
	)

	return footer
}
