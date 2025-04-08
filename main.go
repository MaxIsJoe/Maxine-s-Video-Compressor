package main

import (
	"bufio"
	"fmt"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"log"
	"maxine-compressor.xyz/v2/ffmpegWrapper"
	"maxine-compressor.xyz/v2/persistence"
	"maxine-compressor.xyz/v2/ui"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"bytes"
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
var sizeContainer *fyne.Container
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
		ui.ShowFFmpegInstructions(window)
	}

	statusLabel = widget.NewLabel("")
	statusLabel.Alignment = fyne.TextAlignCenter

	thumbnailImage = &canvas.Image{}
	thumbnailImage.FillMode = canvas.ImageFillContain

	compressBtn = widget.NewButton("Compress", OnCompressButtonPressed)
	compressBtn.Disable() // Initially disabled

	sizeContainer, sizeEntry = ui.CreateSizeEntry(sizeEntry, compressBtn, func() bool { return selectedFile != "" }, updateEstimate)

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

	footer := ui.CreateFooter(resourceKofiIconPng)

	// Add footer to the main content
	mainContent := container.NewBorder(nil, footer, nil, nil, content)

	window.SetOnDropped(OnDroppedFiles)

	window.SetContent(mainContent)
	window.ShowAndRun()
}

func OnDroppedFiles(pos fyne.Position, URIs []fyne.URI) {
	dndFileList = binding.NewStringList()
	for _, file := range URIs {
		if ffmpegWrapper.IsValidVideoFile(file.Path()) {
			dndFileList.Append(file.Path())
		}
	}
	if dndFileList.Length() > 0 {
		ui.ShowFileListDialog(dndFileList, sizeEntry, window, OnCompressButtonPressed)
	}
}

func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func compressVideo(inputPath string, targetSizeMB float64, outputPath string, onComplete func(error)) {
	go func() {
		duration, proberr := ffmpegWrapper.GetVideoDuration(inputPath)
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
			gpuEncoder := ffmpegWrapper.DetectGPUEncoder()
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
		logWindow, logContent, logScrollContainer = ui.ShowLogWindow(application)

		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				logContent.SetText(logContent.Text + scanner.Text() + "\n")
				logScrollContainer.ScrollToBottom() //FIXME: not working?
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

func updateEstimate() {
	if selectedFile == "" {
		estimateLabel.SetText("")
		return
	}

	duration, err := ffmpegWrapper.GetVideoDuration(selectedFile)
	if err != nil || duration <= 0 {
		estimateLabel.SetText("Failed to get duration")
		return
	}

	if sizeEntry == nil {
		estimateLabel.SetText("Invalid size")
		log.Fatal("sizeEntry is nil")
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

func FileSelectionInit() {
	lastFolder = persistence.LoadLastUsedFolder()
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
		persistence.SaveLastUsedFolder(selectedFile)
		img := ffmpegWrapper.ExtractThumbnail(selectedFile)
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
		persistence.SaveLastUsedFolder(selectedFile)
		img := ffmpegWrapper.ExtractThumbnail(selectedFile)
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
