package hlsdl

import (
	"io"
	"io/ioutil"
	"testing"
	"time"
)

// stream URLS taken from here:
// https://ottverse.com/free-hls-m3u8-test-urls/
// https://www.radiantmediaplayer.com/docs/latest/aes-hls-documentation.html
// https://github.com/video-dev/hls.js/blob/master/tests/test-streams.js

const (
	testUrl  = "https://moctobpltc-i.akamaihd.net/hls/live/571329/eight/stream_2000.m3u8"
	testUrl2 = "https://www.radiantmediaplayer.com/media/rmp-segment/bbb-abr-aes/chunklist_b607794.m3u8"
	testUrl3 = "https://playertest.longtailvideo.com/adaptive/oceans_aes/oceans_aes-audio=65000-video=236000.m3u8"
)

func init() {
	EnableDebugMessages()
}

func doDownload(url string, t *testing.T) {
	reader, err := Download(url)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(ioutil.Discard, reader)
	return
}

func TestUnencrypted(t *testing.T) {
	doDownload(testUrl, t)
}

func TestEncrypted(t *testing.T) {
	doDownload(testUrl2, t)
}

func TestEncryptedNoIV(t *testing.T) {
	doDownload(testUrl3, t)
}

func TestStop(t *testing.T) {
	var err error
	for numWorkers := 1; numWorkers <= 2; numWorkers++ {
		client := &Client{
			NumWorkers: numWorkers,
		}
		done := make(chan bool)
		go func() {
			var reader io.Reader
			reader, err = client.Do(testUrl)
			io.Copy(ioutil.Discard, reader)
			done <- true
		}()
		// sleep a bit to let the downloader start
		time.Sleep(500 * time.Millisecond)
		client.Stop()
		timer := time.NewTimer(3 * time.Second)
		select {
		case <-done:
			break
		case <-timer.C:
			t.Fatal("Stop operation timed out")
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}
