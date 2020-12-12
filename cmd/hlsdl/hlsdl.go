package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/maxerenberg/hlsdl"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var cmd = &cobra.Command{
	Use:          "hlsdl <URL>",
	Short:        "Downloads an HLS video from an M3U8 URL",
	Args:         cobra.ExactArgs(1),
	RunE:         cmdF,
	SilenceUsage: true,
}

func main() {
	cmd.Flags().StringP("output", "o", "video.ts", "The name of the output file")
	cmd.Flags().BoolP("keep", "k", false, "Keep segments after downloading")
	cmd.Flags().BoolP("record", "r", false, "Indicate whether the m3u8 is a live stream video and you want to record it")
	cmd.Flags().IntP("workers", "w", 2, "Number of workers to execute concurrent operations")
	cmd.Flags().StringArray(
		"playlist-header", []string{},
		"HTTP header for fetching playlist (may be specified multiple times)")
	cmd.Flags().StringArray(
		"segment-header", []string{},
		"HTTP header for fetching segment (may be specified multiple times)")
	cmd.Flags().StringArray(
		"key-header", []string{},
		"HTTP header for fetching decryption key (may be specified multiple times)")
	cmd.SetArgs(os.Args[1:])

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func cmdF(command *cobra.Command, args []string) (err error) {
	flags := command.Flags()
	m3u8URL := args[0]

	outputPath, err := flags.GetString("output")
	if err != nil {
		return
	}

	keep, err := flags.GetBool("keep")
	if err != nil {
		return
	}

	workers, err := flags.GetInt("workers")
	if err != nil {
		return
	}

	playlistHeadersMap, err := makeHeadersFromFlags(flags, "playlist-header")
	if err != nil {
		return
	}

	segmentHeadersMap, err := makeHeadersFromFlags(flags, "segment-header")
	if err != nil {
		return
	}

	keyHeadersMap, err := makeHeadersFromFlags(flags, "key-header")
	if err != nil {
		return
	}

	if record, err := flags.GetBool("record"); err != nil {
		return err
	} else if record {
		return recordLiveStream(hlsdl.NewRecorder(
			m3u8URL, outputPath,
		))
	}

	return downloadVodMovie(hlsdl.New(
		m3u8URL, outputPath, keep, playlistHeadersMap, segmentHeadersMap,
		keyHeadersMap, workers, true,
	))
}

func makeHeadersFromFlags(flags *flag.FlagSet, name string) (map[string]string, error) {
	headersArray, err := flags.GetStringArray(name)
	if err != nil {
		return nil, err
	}
	headersMap, err := makeHeadersFromArray(headersArray)
	if err != nil {
		return nil, err
	}
	return headersMap, nil
}

func makeHeadersFromArray(headers []string) (map[string]string, error) {
	headersMap := make(map[string]string)
	for _, header := range headers {
		tokens := strings.SplitN(header, ":", 2)
		if len(tokens) < 2 {
			return nil, fmt.Errorf("invalid header %s", header)
		}
		headersMap[tokens[0]] = tokens[1]
	}
	return headersMap, nil
}

func downloadVodMovie(hlsDl *hlsdl.HlsDl) (err error) {
	filepath, err := hlsDl.Download()
	if err != nil {
		return
	}
	log.Println("Downloaded file to " + filepath)
	return
}

func recordLiveStream(recorder *hlsdl.Recorder) error {
	recordedFile, err := recorder.Start()
	if err != nil {
		os.RemoveAll(recordedFile)
		return err
	}

	log.Println("Recorded file at ", recordedFile)
	return nil
}
