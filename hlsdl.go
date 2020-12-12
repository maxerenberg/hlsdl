package hlsdl

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/grafov/m3u8"
	"gopkg.in/cheggaaa/pb.v1"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

// HlsDl present a HLS downloader
type HlsDl struct {
	client          *http.Client
	hlsURL          string
	outputPath      string
	keep            bool
	playlistHeaders map[string]string
	segmentHeaders  map[string]string
	keyHeaders      map[string]string
	workers         int
	bar             *pb.ProgressBar
	enableBar       bool
}

type Segment struct {
	*m3u8.MediaSegment
	Path string
}

type DownloadResult struct {
	Err   error
	SeqId uint64
}

func New(hlsURL string, outputPath string, keep bool, playlistHeaders map[string]string,
	segmentHeaders map[string]string, keyHeaders map[string]string, workers int,
	enableBar bool) *HlsDl {

	hlsdl := &HlsDl{
		hlsURL:          hlsURL,
		outputPath:      outputPath,
		keep:            keep,
		playlistHeaders: playlistHeaders,
		segmentHeaders:  segmentHeaders,
		keyHeaders:      keyHeaders,
		client:          &http.Client{},
		workers:         workers,
		enableBar:       enableBar,
	}

	return hlsdl
}

func wait(wg *sync.WaitGroup) chan bool {
	c := make(chan bool, 1)
	go func() {
		wg.Wait()
		c <- true
	}()
	return c
}

func (hlsDl *HlsDl) downloadSegment(segment *Segment) (err error) {
	req, err := http.NewRequest("GET", segment.URI, nil)
	if err != nil {
		return
	}
	for key, val := range hlsDl.segmentHeaders {
		req.Header.Add(key, val)
	}
	res, err := hlsDl.client.Do(req)
	if err != nil {
		return
	}

	if res.StatusCode != 200 {
		return errors.New(res.Status)
	}

	file, err := os.Create(segment.Path)
	if err != nil {
		return
	}
	defer file.Close()

	if _, err := io.Copy(file, res.Body); err != nil {
		return err
	}

	return
}

func (hlsDl *HlsDl) downloadSegments(segments []*Segment) error {

	wg := &sync.WaitGroup{}
	wg.Add(hlsDl.workers)

	finishedChan := wait(wg)
	quitChan := make(chan bool)
	segmentChan := make(chan *Segment)
	downloadResultChan := make(chan *DownloadResult, hlsDl.workers)

	for i := 0; i < hlsDl.workers; i++ {
		go func() {
			defer wg.Done()

			for segment := range segmentChan {

				tried := 0
			DOWNLOAD:
				tried++

				select {
				case <-quitChan:
					return
				default:
				}

				if err := hlsDl.downloadSegment(segment); err != nil {
					if strings.Contains(err.Error(), "connection reset by peer") && tried < 3 {
						time.Sleep(time.Second)
						log.Println("Retry download segment ", segment.SeqId)
						goto DOWNLOAD
					}

					downloadResultChan <- &DownloadResult{Err: err, SeqId: segment.SeqId}
					return
				}

				downloadResultChan <- &DownloadResult{SeqId: segment.SeqId}
			}
		}()
	}

	go func() {
		defer close(segmentChan)

		for _, segment := range segments {
			segment.Path = fmt.Sprintf("seg%d.ts", segment.SeqId)

			select {
			case segmentChan <- segment:
			case <-quitChan:
				return
			}
		}

	}()

	if hlsDl.enableBar {
		hlsDl.bar = pb.New(len(segments)).SetMaxWidth(100).Prefix("Downloading...")
		hlsDl.bar.ShowElapsedTime = true
		hlsDl.bar.Start()
	}

	defer func() {
		if hlsDl.enableBar {
			hlsDl.bar.Finish()
		}
	}()

	for {
		select {
		case <-finishedChan:
			return nil
		case result := <-downloadResultChan:
			if result.Err != nil {
				close(quitChan)
				return result.Err
			}

			if hlsDl.enableBar {
				hlsDl.bar.Increment()
			}
		}
	}

}

func (hlsDl *HlsDl) join(segments []*Segment) error {
	log.Printf("Joining %d segments\n", len(segments))

	file, err := os.Create(hlsDl.outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	sort.Slice(segments, func(i, j int) bool {
		return segments[i].SeqId < segments[j].SeqId
	})

	for _, segment := range segments {

		d, err := hlsDl.Decrypt(segment)
		if err != nil {
			return err
		}

		if _, err := file.Write(d); err != nil {
			return err
		}

		if !hlsDl.keep {
			if err := os.Remove(segment.Path); err != nil {
				return err
			}
		}
	}

	return nil
}

func (hlsDl *HlsDl) Download() (outputPath string, err error) {
	segs, err := parseHlsSegments(hlsDl.client, hlsDl.hlsURL, hlsDl.playlistHeaders)
	if err != nil {
		return
	}

	if err = hlsDl.downloadSegments(segs); err != nil {
		return
	}

	if err = hlsDl.join(segs); err != nil {
		return
	}

	outputPath = hlsDl.outputPath
	return
}
