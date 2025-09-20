package ffmpeghelper

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

var (
	errTsFetchFailed = errors.New("failed to fetch ts url")
	errTsParseFailed = errors.New("failed to parse ts url")
)

// Get .ts url from m3u8 url
func m3u8GetTsUrl(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errTsFetchFailed
	}
	body, _ := io.ReadAll(res.Body)
	// parse the last .ts url
	if parts := strings.Split(
		strings.TrimSuffix(string(body), "\r\n"), "\r\n"); len(parts) > 0 {
		ts := parts[len(parts)-1]
		return url[:strings.LastIndex(url, "/")+1] + ts, nil
	}
	return "", errTsParseFailed
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
	ffmpeg, err := Ffmpeg()
	if err != nil {
		return nil, err
	}
	ts, err := m3u8GetTsUrl(url)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(
		ffmpeg,
		"-v", "quiet", // no logs
		"-flags", "low_delay", // low delay
		"-fflags", "discardcorrupt+flush_packets", // low delay
		"-probesize", "2048", // low delay
		"-i", ts,
		"-an", // no audio
		"-pix_fmt", "yuvj420p",
		"-vframes", "1",
		"-f", "image2", // jpeg
		"-g", "1",
		"-",
	)
	out := &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = out, nil
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return jpeg.Decode(out)
}
