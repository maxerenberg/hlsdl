package hlsdl

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
)

type Recorder struct {
	client     *http.Client
	outputPath string
	url        string
	runing     bool
}

func NewRecorder(url string, outputPath string) *Recorder {
	return &Recorder{
		url:        url,
		outputPath: outputPath,
		client:     &http.Client{},
	}
}

// Start starts a record a live streaming
func (r *Recorder) Start() (string, error) {
	log.Println("Start record live streaming movie...")

	quitSignal := make(chan os.Signal, 1)
	signal.Notify(quitSignal, os.Interrupt)
	// TODO: pass headers from args
	puller := pullSegment(r.client, r.url, nil, quitSignal)

	file, err := os.Create(r.outputPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

LOOP:
	for segment := range puller {
		if segment.Err != nil {
			return "", segment.Err
		}

		dc := r.downloadSegmentC(segment.Segment)

		select {
		case report := <-dc:
			if report.Err != nil {
				return "", report.Err
			}

			if _, err := file.Write(report.Data); err != nil {
				return "", err
			}
		case <-quitSignal:
			break LOOP
		}

		log.Println("Recorded segment ", segment.Segment.SeqId)
	}

	return r.outputPath, nil
}

type DownloadSegmentReport struct {
	Data []byte
	Err  error
}

func (r *Recorder) downloadSegmentC(segment *Segment) chan *DownloadSegmentReport {
	c := make(chan *DownloadSegmentReport, 1)
	go func() {
		data, err := r.downloadSegment(segment)
		c <- &DownloadSegmentReport{
			Data: data,
			Err:  err,
		}
	}()

	return c
}

func (r *Recorder) downloadSegment(segment *Segment) ([]byte, error) {
	// TODO: include segment headers in request
	res, err := r.client.Get(segment.URI)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if segment.Key != nil {

		key, iv, err := r.getKey(segment)
		if err != nil {
			return nil, err
		}
		data, err = AES128Decrypt(data, key, iv)

		if err != nil {
			return nil, err
		}
	}

	syncByte := uint8(71) //0x47
	bLen := len(data)
	for j := 0; j < bLen; j++ {
		if data[j] == syncByte {
			data = data[j:]
			break
		}
	}

	return data, nil
}

func (r *Recorder) getKey(segment *Segment) (key []byte, iv []byte, err error) {

	res, err := r.client.Get(segment.Key.URI)
	if err != nil {
		return nil, nil, err
	}

	if res.StatusCode != 200 {
		return nil, nil, errors.New("Failed to get descryption key")
	}

	key, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	iv = []byte(segment.Key.IV)
	return
}
