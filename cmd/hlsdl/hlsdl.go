package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/maxerenberg/hlsdl"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

// channel for receiving Interrupt signals
var sigChan chan os.Signal = make(chan os.Signal, 1)

func init() {
	signal.Notify(sigChan, os.Interrupt)
}

var cmd = &cobra.Command{
	Use:          "hlsdl <URL>",
	Short:        "Download an HLS video from an M3U8 URL",
	Args:         cobra.ExactArgs(1),
	RunE:         cmdF,
	SilenceUsage: true,
}

func main() {
	cmd.Flags().StringP("output", "o", "video.ts", "The name of the output file")
	cmd.Flags().IntP("workers", "w", 2, "Number of workers to execute concurrent operations")
	cmd.Flags().BoolP("quiet", "q", false, "No progress bar")
	cmd.Flags().StringArrayP(
		"add-header", "H", []string{},
		"HTTP header to use (may be specified multiple times)")
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
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return
	}
	defer outputFile.Close()
	workers, err := flags.GetInt("workers")
	if err != nil {
		return
	}
	quiet, err := flags.GetBool("quiet")
	if err != nil {
		return
	}
	headersMap, err := makeHeadersFromFlags(flags, "add-header")
	if err != nil {
		return
	}

	client := &hlsdl.Client{
		NumWorkers: workers,
		EnableBar:  !quiet,
		Headers:    headersMap,
	}
	// stop the downloading if we get a signal
	go func() {
		<-sigChan
		println("Received Interrupt signal, stopping now.")
		client.Stop()
	}()
	reader, err := client.Do(m3u8URL)
	if err != nil {
		return
	}
	_, err = io.Copy(outputFile, reader)
	return
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
