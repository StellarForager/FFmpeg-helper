package ffmpeghelper_test

import (
	"image"
	"image/jpeg"
	"os"
	"testing"

	ffmpeghelper "github.com/StellarForager/FFmpeg-helper"
)

func TestFfmpeg(t *testing.T) {
	url := os.Getenv("M3U8_URL")
	t.Log(ffmpeghelper.Ffmpeg())
	img, err := ffmpeghelper.H264M3U8GetImage(url)
	t.Log(img.(*image.YCbCr).Bounds(), err)
	ofile, _ := os.Create("TestFfmpeg_H264M3U8GetImage.jpg")
	defer ofile.Close()
	jpeg.Encode(ofile, img, nil)
	t.Log(ffmpeghelper.ImgScanQrcode(img))
}
