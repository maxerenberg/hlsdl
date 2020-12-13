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

	// used for telling the workers to stop early
	_syncChans []chan bool
	// for a pretty display
	_bar *pb.ProgressBar
}

// Stops all the workers
func (client *Client) Stop() {
	for _, syncChan := range client._syncChans {
		syncChan <- false
	}
}

func initializeClient(client *Client) {
	if client.NumWorkers <= 0 {
		client.NumWorkers = runtime.NumCPU()
	}
	client._syncChans = make([]chan bool, client.NumWorkers)
	for i := 0; i < client.NumWorkers; i++ {
		// Use a capacity of 2 so that after Stop() is called,
		// workers don't block writing to a channel
		client._syncChans[i] = make(chan bool, 2)
	}
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
		client._bar = pb.New(len(mediaPlaylist.Segments)).SetMaxWidth(100).Prefix("Downloading...")
		client._bar.ShowElapsedTime = true
		client._bar.Start()
	}

	reader, writer := io.Pipe()

	// The idea is that each worker is responsible for downloading segments
	// at certain intervals. For example, if there are 4 workers, then
	// worker 0 downloads segments 0, 4, 8, etc.; worker 1 downloads
	// segments 1, 5, 9, etc.; and so on.
	// To ensure that each segment is written in order, each worker gets two
	// channels: one to check if it's OK to write, and one to tell the next
	// worker that it's OK to write.
	for i := 0; i < client.NumWorkers; i++ {
		go client.downloadSegments(
			i,
			m3u8url,
			mediaPlaylist.Segments,
			writer,
			client._syncChans[i],
			client._syncChans[(i+1)%client.NumWorkers],
		)
	}
	// Kick the ball to get it rolling
	client._syncChans[0] <- true
	return
}

func (client *Client) downloadSegments(
	idx int,
	m3u8url string,
	segments []*m3u8.MediaSegment,
	writer *io.PipeWriter,
	allowMe chan bool,
	allowNext chan bool,
) {
	// The very last worker who writes a segment should be the one
	// who closes the writer
	if (len(segments)-1-idx)%client.NumWorkers == 0 {
		defer writer.Close()
		if client.EnableBar {
			defer client._bar.Finish()
		}
	}
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
		if client.EnableBar {
			client._bar.Increment()
		}
		allowedToContinue := <-allowMe
		if !allowedToContinue {
			logger.Printf("Worker %d stopping early\n", origIdx)
			break
		}
		// TODO: deal with case where write fails
		writer.Write(data)
		allowNext <- true
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
