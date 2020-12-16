package hlsdl

import (
	"errors"
	"io"
	"io/ioutil"
	"runtime"

	"github.com/grafov/m3u8"
	"gopkg.in/cheggaaa/pb.v1"
)

type Client struct {
	// number of concurrent NumWorkers
	NumWorkers int
	// whether to enable a TUI progress bar
	EnableBar bool
	// additional HTTP Headers when making requests
	Headers map[string]string

	// for telling the workers to stop early
	workerStopChans []chan bool
	// for telling the writer to stop early
	writerStopChan chan bool
	// for a pretty display
	bar *pb.ProgressBar
}

// Stops all the workers
func (client *Client) Stop() {
	for _, stopChan := range client.workerStopChans {
		select {
			case stopChan <- true:
			default:
		}
	}
	select {
		case client.writerStopChan <- true:
		default:
	}
}

func initializeClient(client *Client) {
	if client.NumWorkers <= 0 {
		client.NumWorkers = runtime.NumCPU()
	}
	client.workerStopChans = make([]chan bool, client.NumWorkers)
	for i := range client.workerStopChans {
		client.workerStopChans[i] = make(chan bool, 1)
	}
	client.writerStopChan = make(chan bool, 1)
	client.bar = nil
}

// Do downloads an HLS video from a Request containing the URL of an
// M3U8 file. The data is returned in an io.Reader.
func (client *Client) Do(m3u8url string) (reader io.Reader, err error) {
	initializeClient(client)
	// Fetch the M3U8 file
	res, err := client.doRequest(m3u8url)
	if err != nil {
		return
	}
	// Parse the M3U8 file
	playlist, playlistType, err := m3u8.DecodeFrom(res.Body, false)
	if err != nil {
		return
	} else if playlistType != m3u8.MEDIA {
		err = errors.New("Playlists of type MASTER are not supported yet")
		return
	}
	mediaPlaylist := playlist.(*m3u8.MediaPlaylist)
	// Check the segments and remove the ones which are nil
	if err = adjustSegments(mediaPlaylist); err != nil {
		return
	}
	// Adjust the Key and IV for each segment
	if err = adjustKeys(mediaPlaylist); err != nil {
		return
	}
	if client.EnableBar {
		client.bar = pb.New(len(mediaPlaylist.Segments)).SetMaxWidth(100).Prefix("Downloading...")
		client.bar.ShowElapsedTime = true
		client.bar.Start()
	}

	workerChans := make([]chan []byte, client.NumWorkers)
	for i := range workerChans {
		workerChans[i] = make(chan []byte)
	}
	reader, writer := io.Pipe()
	go client.writeServer(workerChans, writer, len(mediaPlaylist.Segments))

	for i := 0; i < client.NumWorkers; i++ {
		go client.downloadSegments(
			i,
			m3u8url,
			mediaPlaylist.Segments,
			workerChans[i],
			client.workerStopChans[i],
		)
	}
	return
}

func (client *Client) downloadSegments(
	idx int,
	m3u8url string,
	segments []*m3u8.MediaSegment,
	writeChan chan<- []byte,
	stopChan <-chan bool,
) {
	origIdx := idx
	// TODO: propagate the errors back to the user without panicking
	for ; idx < len(segments); idx += client.NumWorkers {
		seg := segments[idx]
		uri, err := absURL(m3u8url, seg.URI)
		if err != nil {
			panic(err)
		}
		res, err := client.doRequest(uri)
		if err != nil {
			panic(err)
		}
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		data, err = client.decryptSegment(m3u8url, seg, data)
		if err != nil {
			panic(err)
		}
		select {
			case <-stopChan:
				logger.Printf("Worker %d stopping early\n", origIdx)
				return
			case writeChan <- data:
		}
	}
}

func (client *Client) writeServer(
	workerChans []chan []byte,
	writer *io.PipeWriter,
	numSegments int,
) {
	defer func() {
		writer.Close()
		if client.EnableBar {
			client.bar.Finish()
		}
	}()
	for i, j := 0, 0; i < numSegments; i, j = i+1, (j+1)%client.NumWorkers {
		var chunk []byte
		select {
			case <-client.writerStopChan:
				return
			case chunk = <-workerChans[j]:
		}
		writer.Write(chunk)
		if client.EnableBar {
			client.bar.Increment()
		}
	}
}

func adjustSegments(mediaPlaylist *m3u8.MediaPlaylist) (err error) {
	for i, seg := range mediaPlaylist.Segments {
		if seg == nil {
			// The m3u8 library appears to always create a playlist of
			// capacity 1024, so we can just truncate the list
			mediaPlaylist.Segments = mediaPlaylist.Segments[:i]
			return
		} else if seg.Discontinuity {
			return errors.New("Discontinuities are not currently supported")
		} else if seg.Map != nil {
			return errors.New("X-EXT-MAP tag is not currently supported")
		}
	}
	return
}

// convenience function for downloading video with default parameters
func Download(m3u8url string) (res io.Reader, err error) {
	client := &Client{}
	res, err = client.Do(m3u8url)
	return
}
