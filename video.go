package ffmpeghelper

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"os/exec"
	"strings"
)

var (
	ErrTsFetchFailed = errors.New("failed to fetch ts url")
	ErrTsParseFailed = errors.New("failed to parse ts url")
	ErrTsReadFailed  = errors.New("failed to get ts data")
)

// Get .ts url from m3u8 url
func m3u8GetTsUrl(url string) (string, error) {
	res, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", ErrTsFetchFailed
	}
	body, _ := io.ReadAll(res.Body)
	// parse the last .ts url
	if parts := strings.Split(
		strings.TrimSuffix(string(body), "\r\n"), "\r\n"); len(parts) > 0 {
		ts := parts[len(parts)-1]
		return url[:strings.LastIndex(url, "/")+1] + ts, nil
	}
	return "", ErrTsParseFailed
}

// Get a jpeg image from a H.264 M3U8 stream.
//
// Args:
//
//	url: url of the stream
//
// Returns:
//
//	image.Image: the jpeg image
//	error: error
func H264M3U8GetImage(url string) (image.Image, error) {
	// get ffmpeg path
	ffmpeg, err := Ffmpeg()
	if err != nil {
		return nil, err
	}
	// get .ts url
	tsUrl, err := m3u8GetTsUrl(url)
	if err != nil {
		return nil, err
	}
	// get .ts body
	res, err := httpClient.Get(tsUrl)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, ErrTsReadFailed
	}
	cmd := exec.Command(
		ffmpeg,
		"-v", "quiet", // no logs
		"-flags", "low_delay", // low delay
		"-fflags", "discardcorrupt+flush_packets", // low delay
		"-probesize", "2048", // low delay
		"-i", "pipe:", // read from stdin
		"-an",                  // no audio
		"-pix_fmt", "yuvj420p", // source video format
		"-vframes", "1", // 1 frame
		"-g", "1", // force all frames to be key frames
		"-f", "image2", // output as jpeg
		"-", // print to stdout
	)
	out := &bytes.Buffer{}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = res.Body, out, nil
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return jpeg.Decode(out)
}
