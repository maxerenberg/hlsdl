package hlsdl

import (
	"fmt"
	"net/http"
	"os"
	"testing"
)

// stream URLS taken from here:
// https://ottverse.com/free-hls-m3u8-test-urls/
// https://www.radiantmediaplayer.com/docs/latest/aes-hls-documentation.html

const (
	testUrl  = "https://moctobpltc-i.akamaihd.net/hls/live/571329/eight/stream_2000.m3u8"
	testUrl2 = "https://www.radiantmediaplayer.com/media/rmp-segment/bbb-abr-aes/chunklist_b607794.m3u8"
)

func TestDecrypt(t *testing.T) {
	segs, err := parseHlsSegments(
		&http.Client{},
		testUrl,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	hlsDl := New(
		testUrl,
		"video.ts",
		false,
		nil, nil, nil,
		2,
		false,
	)

	seg := segs[0]
	seg.Path = fmt.Sprintf("seg%d.ts", seg.SeqId)
	defer os.Remove(seg.Path)
	if err := hlsDl.downloadSegment(seg); err != nil {
		t.Fatal(err)
	}

	if _, err := hlsDl.Decrypt(seg); err != nil {
		t.Fatal(err)
	}
}

func TestDownload(t *testing.T) {

	hlsDl := New(
		testUrl2,
		"video.ts",
		false,
		nil, nil, nil,
		2,
		false,
	)
	filepath, err := hlsDl.Download()
	defer os.Remove(filepath)
	if err != nil {
		t.Fatal(err)
	}
}
