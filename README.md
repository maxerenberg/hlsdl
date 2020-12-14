# HLS downloader
This is a library and application to download an HLS video from a m3u8 file. All segments
will be downloaded individually and then concatenated into a single TS file. The default file
name is `video.ts`.


### Features
* Concurrent download segments with multiple HTTP connections
* Decrypt HLS encoded segments
* Display downloading progress bar

### Limitations
* Master playlists are not supported (playlists which point to other playlists)
* Fragmented MP4 (fMP4) is not supported (only MPEG-TS)
* Playlists with the EXT-X-MAP tag are not supported

## Library usage

Get the library:
```
go get github.com/maxerenberg/hlsdl
```

Sample:

```
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

```

## CLI Installation

```
go get -u -v github.com/maxerenberg/hlsdl/cmd/hlsdl
```
The command `hlsdl` will now be available in your $GOBIN directory.

To see all options, run `hlsdl --help`.


### Example usage

```
$ hlsdl "https://bitdash-a.akamaihd.net/content/sintel/hls/video/1500kbit.m3u8"
```

You can also specify custom HTTP headers:
```
$ hlsdl -H "Accept-Language: en-US,en;q=0.5" \
	"https://bitdash-a.akamaihd.net/content/sintel/hls/video/1500kbit.m3u8"
```
