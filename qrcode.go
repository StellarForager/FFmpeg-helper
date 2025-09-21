package ffmpeghelper

import (
	"image"
	_ "image/jpeg"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/multi/qrcode"
)

var qrReader = qrcode.NewQRCodeMultiReader()

func ImgScanQrcode(img image.Image) ([]string, error) {
	bmp, _ := gozxing.NewBinaryBitmapFromImage(img)
	results, err := qrReader.DecodeMultiple(bmp, nil)
	if err != nil {
		return nil, err
	}
	data := make([]string, 0, len(results))
	for _, r := range results {
		data = append(data, r.String())
	}
	return data, nil
}
