# Maxine's Video Compressor

![https://github.com/MaxIsJoe/Maxine-s-Video-Compressor/blob/main/.github/assets/sc_preview_v2.png](https://github.com/MaxIsJoe/Maxine-s-Video-Compressor/blob/main/.github/assets/sc_preview_v2.png)

A simple FFMPEG GUI wrapper that is designed to compress videos for uploading content on size-limitng platforms such as Discord. Written in Go, and utilizes Fyne for the graphical interface.

Comes with the option to extract audio from videos, dragging and dropping multiple video files for bulk compressing, and the ability to utilise your GPU for faster encoding.

## Requirements

Requires [FFMPEG](https://ffmpeg.org/download.html) to use. 

If you don't have it installed, the application will attempt to install it for you by default using Choco or Scoop on windows, or apt-get on Linux.

## Building

Simply run `go run main.go` in your terminal to quickly test it.

This app uses Fyne, so the first time you compile this project might take a little while as it prepares all the Fyne packages.

## Plans

I originally created this as a replacement to [Cheezo's Video Compressor](https://github.com/cheezos/video-compressor) for myself. I may or may not add additonal features in the future, but unless I absolutely need something for my own use cases or there is clear interest in this project, I do not have any plans for this project other than maintaining it for compressing my gameplay clips that I upload on [Bluesky](https://bsky.app/profile/maxisjoe.xyz/post/3lll76fvz5s2w) and Discord, showing this project on my portfolio, and extracting music from music videos on my hard-drive.

## Donate

https://maxisjoe.xyz/maxfund
