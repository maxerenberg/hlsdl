package main

import (
	"fmt"

	"github.com/maxerenberg/hlsdl"
)

func main() {
	hlsDL := hlsdl.New(
		"https://bitdash-a.akamaihd.net/content/sintel/hls/video/1500kbit.m3u8",
		"video.ts",
		false,
		nil, nil, nil,
		4,
		true,
	)
	filepath, err := hlsDL.Download()
	if err != nil {
		panic(err)
	}

	fmt.Println(filepath)
}
