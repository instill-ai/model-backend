package rpc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"mime/multipart"
	"net/http"
	"path"

	consts "github.com/instill-ai/model-backend/internal"
	"github.com/instill-ai/protogen-go/model"
)

// Internally used image metadata
type imageMetadata struct {
	filename string
	format   string
	width    int
	height   int
}

type imageFormDataInputs struct {
	Contents []*multipart.FileHeader `form:"contents"`
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

	if numBytes > int64(consts.MaxImageSizeBytes) {
		return nil, nil, fmt.Errorf(
			"Image size must be smaller than %vMB. Got %vMB",
			float32(consts.MaxImageSizeBytes)/float32(consts.MB),
			float32(numBytes)/float32(consts.MB),
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
	if numBytes > consts.MaxImageSizeBytes {
		return nil, nil, fmt.Errorf(
			"Image size must be smaller than %vMB. Got %vMB",
			float32(consts.MaxImageSizeBytes)/float32(consts.MB),
			float32(numBytes)/float32(consts.MB),
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

func ParseImageRequestInputsToBytes(inputs *model.PredictModelImageRequest) (imgsBytes [][]byte, imgsMetadata []*imageMetadata, err error) {
	for idx, content := range inputs.Contents {
		var (
			img      *image.Image
			metadata *imageMetadata
			err      error
		)

		if len(content.Url) > 0 {
			img, metadata, err = parseImageFromURL(content.Url)
			if err != nil {
				log.Printf("Unable to parse image %v from url. %v", idx, err) // The internal error err only appears in the logs, because it's only useful to us
				return nil, nil, fmt.Errorf("Unable to parse image %v from url", idx)
			}
		} else if len(content.Base64) > 0 {
			img, metadata, err = parseImageFromBase64(content.Base64)
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

func parseImageFormDataInputsToBytes(r *http.Request) (imgsBytes [][]byte, imgsMetadata []*imageMetadata, err error) {
	inputs := r.MultipartForm.File["contents"]
	for _, content := range inputs {
		file, err := content.Open()
		if err != nil {
			log.Printf("Unable to open file for image %v. %v", content.Filename, err) // The internal error err only appears in the logs, because it's only useful to us
			return nil, nil, fmt.Errorf("Unable to open file for image %v", content.Filename)
		}

		buff := new(bytes.Buffer) // pointer
		numBytes, err := buff.ReadFrom(file)
		if err != nil {
			log.Printf("Unable to read content body from image %v. %v", content.Filename, err) // The internal error err only appears in the logs, because it's only useful to us
			return nil, nil, fmt.Errorf("Unable to read content body from image %v", content.Filename)
		}

		if numBytes > int64(consts.MaxImageSizeBytes) {
			return nil, nil, fmt.Errorf(
				"Image size must be smaller than %vMB. Got %vMB from image %v",
				float32(consts.MaxImageSizeBytes)/float32(consts.MB),
				float32(numBytes)/float32(consts.MB),
				content.Filename,
			)
		}

		img, format, err := image.Decode(buff)
		if err != nil {
			log.Printf("Unable to decode image: %v. %v", content.Filename, err) // The internal error err only appears in the logs, because it's only useful to us
			return nil, nil, fmt.Errorf("Unable to decode image %v", content.Filename)
		}

		// Encode into jpeg to remove alpha channel (hack)
		// This may slightly degrade the image quality
		buff = new(bytes.Buffer)
		err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
		if err != nil {
			log.Printf("Unable to process image: %v. %v", content.Filename, err) // The internal error err only appears in the logs, because it's only useful to us
			return nil, nil, fmt.Errorf("Unable to process image %v", content.Filename)
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
