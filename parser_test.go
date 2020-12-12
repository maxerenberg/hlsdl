package hlsdl

import (
	"net/http"
	"testing"
)

func Test(t *testing.T) {
	_, err := parseHlsSegments(
		&http.Client{},
		"https://moctobpltc-i.akamaihd.net/hls/live/571329/eight/stream_2000.m3u8",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
}
