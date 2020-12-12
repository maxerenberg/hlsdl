package hlsdl

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/grafov/m3u8"
)

func parseHlsSegments(
	client *http.Client,
	hlsURL string,
	headers map[string]string,
) ([]*Segment, error) {
	baseURL, err := url.Parse(hlsURL)
	if err != nil {
		return nil, errors.New("Invalid m3u8 url")
	}

	p, t, err := getM3u8ListType(client, hlsURL, headers)
	if err != nil {
		return nil, err
	}
	if t != m3u8.MEDIA {
		log.Printf("ListType: %d\n", t)
		return nil, errors.New("No support the m3u8 format")
	}

	mediaList := p.(*m3u8.MediaPlaylist)
	segments := []*Segment{}
	for _, seg := range mediaList.Segments {
		if seg == nil {
			continue
		}

		if !strings.Contains(seg.URI, "http") {
			segmentURL, err := baseURL.Parse(seg.URI)
			if err != nil {
				return nil, err
			}

			seg.URI = segmentURL.String()
		}

		if seg.Key == nil && mediaList.Key != nil {
			seg.Key = mediaList.Key
		}

		if seg.Key != nil && !strings.Contains(seg.Key.URI, "http") {
			keyURL, err := baseURL.Parse(seg.Key.URI)
			if err != nil {
				return nil, err
			}

			seg.Key.URI = keyURL.String()
		}

		segment := &Segment{MediaSegment: seg}
		segments = append(segments, segment)
	}

	return segments, nil
}

func getM3u8ListType(
	client *http.Client,
	hlsURL string,
	headers map[string]string,
) (p m3u8.Playlist, t m3u8.ListType, err error) {
	req, err := http.NewRequest("GET", hlsURL, nil)
	if err != nil {
		return
	}
	for key, val := range headers {
		req.Header.Add(key, val)
	}

	res, err := client.Do(req)
	if err != nil {
		return
	} else if res.StatusCode != 200 {
		err = errors.New(res.Status)
		return
	}

	p, t, err = m3u8.DecodeFrom(res.Body, false)
	if err != nil {
		return
	}

	return
}
