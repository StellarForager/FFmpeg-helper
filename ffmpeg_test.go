package ffmpeghelper_test

import (
	"testing"

	ffmpeghelper "github.com/StellarForager/FFmpeg-helper"
)

func TestFfmpeg(t *testing.T) {
	ffmpeghelper.FetchFfmpeg()
	// url := os.Getenv("M3U8_URL")
	// ffmpeg, err := ffmpeghelper.Ffmpeg()
	// if err != nil {
	// 	t.Fatalf("err: %v", err)
	// }
	// t.Logf("ffmpeg: %v", ffmpeg)
	// cmd := exec.Command(ffmpeg, "-version")
	// stdout := &bytes.Buffer{}
	// cmd.Stdout = stdout
	// if err := cmd.Run(); err != nil {
	// 	t.Fatalf("err: %v", err)
	// }
	// t.Log(stdout.String())
	// ts := time.Now()
	// img, err := ffmpeghelper.H264M3U8GetImage(url)
	// te := time.Since(ts)
	// if err != nil {
	// 	t.Fatalf("err: %v", err)
	// }
	// t.Logf("img: %v, time: %v", img.(*image.YCbCr).Bounds(), te)
	// ofile, _ := os.Create("TestFfmpeg_H264M3U8GetImage.jpg")
	// defer ofile.Close()
	// jpeg.Encode(ofile, img, nil)
	// t.Log(ffmpeghelper.ImgScanQrcode(img))
}
