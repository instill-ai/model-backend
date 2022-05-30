package handler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"path"

	"github.com/instill-ai/model-backend/internal/util"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// Internally used image metadata
type imageMetadata struct {
	filename string
	format   string
	width    int
	height   int
}

func parseImageFromURL(url string) (*image.Image, *imageMetadata, error) {
	response, err := http.Get(url)
	if err != nil {
		log.Printf("Unable to download image at %v. %v", url, err) // The internal error err only appears in the logs, because it's only useful to us
		return nil, nil, fmt.Errorf("Unable to download image at %v", url)
	}
	defer response.Body.Close()

	buff := new(bytes.Buffer) // pointer
	numBytes, err := buff.ReadFrom(response.Body)
	if err != nil {
		log.Printf("Unable to read content body from image at %v. %v", url, err) // The internal error err only appears in the logs, because it's only useful to us
		return nil, nil, fmt.Errorf("Unable to read content body from image at %v", url)
	}

	if numBytes > int64(util.MaxImageSizeBytes) {
		return nil, nil, fmt.Errorf(
			"Image size must be smaller than %vMB. Got %vMB",
			float32(util.MaxImageSizeBytes)/float32(util.MB),
			float32(numBytes)/float32(util.MB),
		)
	}

	img, format, err := image.Decode(buff)
	if err != nil {
		log.Printf("Unable to decode image at %v. %v", url, err) // The internal error err only appears in the logs, because it's only useful to us
		return nil, nil, fmt.Errorf("Unable to decode image at %v", url)
	}

	metadata := imageMetadata{
		filename: path.Base(url),
		format:   format,
		width:    img.Bounds().Dx(),
		height:   img.Bounds().Dy(),
	}

	return &img, &metadata, nil
}

func parseImageFromBase64(encoded string) (*image.Image, *imageMetadata, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Printf("Unable to decode base64 image. %v", err) // The internal error err only appears in the logs, because it's only useful to us
		return nil, nil, fmt.Errorf("Unable to decode base64 image")
	}
	numBytes := len(decoded)
	if numBytes > util.MaxImageSizeBytes {
		return nil, nil, fmt.Errorf(
			"Image size must be smaller than %vMB. Got %vMB",
			float32(util.MaxImageSizeBytes)/float32(util.MB),
			float32(numBytes)/float32(util.MB),
		)
	}
	img, format, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		log.Printf("Unable to decode base64 image. %v", err) // The internal error err only appears in the logs, because it's only useful to us
		return nil, nil, fmt.Errorf("Unable to decode base64 image")
	}

	metadata := imageMetadata{
		filename: "",
		format:   format,
		width:    img.Bounds().Dx(),
		height:   img.Bounds().Dy(),
	}

	return &img, &metadata, nil
}

func ParseImageRequestInputsToBytes(req *modelPB.TriggerModelInstanceRequest) (imgsBytes [][]byte, imgsMetadata []*imageMetadata, err error) {
	for idx, content := range req.Inputs {
		var (
			img      *image.Image
			metadata *imageMetadata
			err      error
		)
		if len(content.GetImageUrl()) > 0 {
			img, metadata, err = parseImageFromURL(content.GetImageUrl())
			if err != nil {
				log.Printf("Unable to parse image %v from url. %v", idx, err) // The internal error err only appears in the logs, because it's only useful to us
				return nil, nil, fmt.Errorf("Unable to parse image %v from url", idx)
			}
		} else if len(content.GetImageBase64()) > 0 {
			img, metadata, err = parseImageFromBase64(content.GetImageBase64())
			if err != nil {
				log.Printf("Unable to parse base64 image %v. %v", idx, err) // The internal error err only appears in the logs, because it's only useful to us
				return nil, nil, fmt.Errorf("Unable to parse base64 image %v", idx)
			}
		} else {
			return nil, nil, fmt.Errorf(`Image %v must define either a "url" or "base64" field. None of them were defined`, idx)
		}

		// Encode into jpeg to remove alpha channel (hack)
		// This may slightly degrade the image quality
		buff := new(bytes.Buffer)
		err = jpeg.Encode(buff, *img, &jpeg.Options{Quality: 100})
		if err != nil {
			log.Printf("Unable to process image %v. %v", idx, err) // The internal error err only appears in the logs, because it's only useful to us
			return nil, nil, fmt.Errorf("Unable to process image %v", idx)
		}

		imgsBytes = append(imgsBytes, buff.Bytes())
		imgsMetadata = append(imgsMetadata, metadata)
	}
	return imgsBytes, imgsMetadata, nil
}

func parseImageFormDataInputsToBytes(req *http.Request) (imgsBytes [][]byte, imgsMetadata []*imageMetadata, err error) {
	inputs := req.MultipartForm.File["file"]
	for _, content := range inputs {
		file, err := content.Open()
		if err != nil {
			log.Printf("Unable to open file for image %v", err) // The internal error err only appears in the logs, because it's only useful to us
			return nil, nil, fmt.Errorf("Unable to open file for image")
		}

		buff := new(bytes.Buffer) // pointer
		numBytes, err := buff.ReadFrom(file)
		if err != nil {
			log.Printf("Unable to read content body from image %v", err) // The internal error err only appears in the logs, because it's only useful to us
			return nil, nil, fmt.Errorf("Unable to read content body from image")
		}

		if numBytes > int64(util.MaxImageSizeBytes) {
			return nil, nil, fmt.Errorf(
				"Image size must be smaller than %vMB. Got %vMB from image %v",
				float32(util.MaxImageSizeBytes)/float32(util.MB),
				float32(numBytes)/float32(util.MB),
				content.Filename,
			)
		}

		img, format, err := image.Decode(buff)
		if err != nil {
			log.Printf("Unable to decode image: %v", err) // The internal error err only appears in the logs, because it's only useful to us
			return nil, nil, fmt.Errorf("Unable to decode image")
		}

		// Encode into jpeg to remove alpha channel (hack)
		// This may slightly degrade the image quality
		buff = new(bytes.Buffer)
		err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
		if err != nil {
			log.Printf("Unable to process image: %v", err) // The internal error err only appears in the logs, because it's only useful to us
			return nil, nil, fmt.Errorf("Unable to process image")
		}

		imgsBytes = append(imgsBytes, buff.Bytes())
		imgsMetadata = append(imgsMetadata, &imageMetadata{
			filename: content.Filename,
			format:   format,
			width:    img.Bounds().Dx(),
			height:   img.Bounds().Dy(),
		})
	}

	return imgsBytes, imgsMetadata, nil
}
