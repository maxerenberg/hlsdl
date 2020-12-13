package main

import (
	"io"
	"os"

	"github.com/maxerenberg/hlsdl"
)

func main() {
	url := "https://bitdash-a.akamaihd.net/content/sintel/hls/video/1500kbit.m3u8"
	reader, err := hlsdl.Download(url)
	if err != nil {
		panic(err)
	}
	file, _ := os.Create("video.ts")
	io.Copy(file, reader)
	file.Close()
}
