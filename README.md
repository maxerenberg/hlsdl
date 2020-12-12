# HLS downloader
This is a library and application to download an HLS video from a m3u8 file. All ts segments
will be downloaded individually and then concatenated into a single TS file. The default file
name is `video.ts`.


### Features:
* Concurrent download segments with multiple HTTP connections
* Decrypt HLS encoded segments
* Auto retry download
* Display downloading progress bar
* Record a live stream video


### Library usage

Get the library:
```
go get github.com/maxerenberg/hlsdl
```

Sample:

```
package main

import (
	"github.com/maxerenberg/hlsdl"
)

func main() {
	hlsDl := New(
		"https://bitdash-a.akamaihd.net/content/sintel/hls/video/10000kbit.m3u8",
		"video.ts",
		false,
		nil, nil, nil,
		2,
		false,
	)
	filepath, err := hlsDl.Download()
	if err != nil {
		panic(err)
	}
}

```

### CLI Installation

```
go install github.com/maxerenberg/hlsdl/cmd/hlsdl
```
The command `hlsdl` will now be available in your $GOBIN directory.

To see all options, run `hlsdl --help`.


### Example usage

Download a static video:

```
./bin/hlsdl https://bitdash-a.akamaihd.net/content/sintel/hls/video/1500kbit.m3u8 -w 4
```

Record a live stream video:

```
./bin/hlsdl "http://cdn1.live-tv.od.ua:8081/bbb/bbbtv-abr/bbb/bbbtv-720p/chunks.m3u8?nimblesessionid=62115268" --record
```
