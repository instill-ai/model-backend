package handler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"path"

	_ "golang.org/x/image/tiff"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/logger"
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

	logger, _ := logger.GetZapLogger()

	response, err := http.Get(url)
	if err != nil {
		logger.Error(fmt.Sprintf("logUnable to download image at %v. %v", url, err))
		return nil, nil, fmt.Errorf("unable to download image at %v", url)
	}
	defer response.Body.Close()

	buff := new(bytes.Buffer) // pointer
	numBytes, err := buff.ReadFrom(response.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to read content body from image at %v. %v", url, err))
		return nil, nil, fmt.Errorf("unable to read content body from image at %v", url)
	}

	if numBytes > int64(config.Config.Server.MaxDataSize*util.MB) {
		return nil, nil, fmt.Errorf(
			"image size must be smaller than %vMB. Got %vMB",
			config.Config.Server.MaxDataSize,
			float32(numBytes)/float32(util.MB),
		)
	}

	img, format, err := image.Decode(buff)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode image at %v. %v", url, err))
		return nil, nil, fmt.Errorf("unable to decode image at %v", url)
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

	logger, _ := logger.GetZapLogger()

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode base64 image. %v", err))
		return nil, nil, fmt.Errorf("unable to decode base64 image")
	}
	numBytes := len(decoded)
	if numBytes > config.Config.Server.MaxDataSize*util.MB {
		return nil, nil, fmt.Errorf(
			"image size must be smaller than %vMB. Got %vMB",
			config.Config.Server.MaxDataSize,
			float32(numBytes)/float32(util.MB),
		)
	}
	img, format, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode base64 image. %v", err))
		return nil, nil, fmt.Errorf("unable to decode base64 image")
	}

	metadata := imageMetadata{
		filename: "",
		format:   format,
		width:    img.Bounds().Dx(),
		height:   img.Bounds().Dy(),
	}

	return &img, &metadata, nil
}

func parseImageRequestInputsToBytes(req *modelPB.TriggerModelInstanceRequest) (imgsBytes [][]byte, imgsMetadata []*imageMetadata, err error) {

	logger, _ := logger.GetZapLogger()

	for idx, taskInput := range req.TaskInputs {
		var imgUrl, imgBase64 string
		switch taskInput.Input.(type) {
		case *modelPB.TaskInput_Classification:
			imgUrl = taskInput.GetClassification().GetImageUrl()
			imgBase64 = taskInput.GetClassification().GetImageBase64()
		case *modelPB.TaskInput_Detection:
			imgUrl = taskInput.GetDetection().GetImageUrl()
			imgBase64 = taskInput.GetDetection().GetImageBase64()
		case *modelPB.TaskInput_Ocr:
			imgUrl = taskInput.GetOcr().GetImageUrl()
			imgBase64 = taskInput.GetOcr().GetImageBase64()
		case *modelPB.TaskInput_InstanceSegmentation:
			imgUrl = taskInput.GetInstanceSegmentation().GetImageUrl()
			imgBase64 = taskInput.GetInstanceSegmentation().GetImageBase64()
		case *modelPB.TaskInput_SemanticSegmentation:
			imgUrl = taskInput.GetSemanticSegmentation().GetImageUrl()
			imgBase64 = taskInput.GetSemanticSegmentation().GetImageBase64()
		default:
			return nil, nil, fmt.Errorf("unknown task input type")
		}
		var (
			img      *image.Image
			metadata *imageMetadata
			err      error
		)

		if len(imgUrl) > 0 {
			img, metadata, err = parseImageFromURL(imgUrl)
			if err != nil {
				logger.Error(fmt.Sprintf("Unable to parse image %v from url. %v", idx, err))
				return nil, nil, fmt.Errorf("unable to parse image %v from url", idx)
			}
		} else if len(imgBase64) > 0 {
			img, metadata, err = parseImageFromBase64(imgBase64)
			if err != nil {
				logger.Error(fmt.Sprintf("Unable to parse base64 image %v. %v", idx, err))
				return nil, nil, fmt.Errorf("unable to parse base64 image %v", idx)
			}
		} else {
			return nil, nil, fmt.Errorf(`image %v must define either a "url" or "base64" field. None of them were defined`, idx)
		}

		// Encode into jpeg to remove alpha channel (hack)
		// This may slightly degrade the image quality
		buff := new(bytes.Buffer)
		err = jpeg.Encode(buff, *img, &jpeg.Options{Quality: 100})
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to process image %v. %v", idx, err))
			return nil, nil, fmt.Errorf("unable to process image %v", idx)
		}

		imgsBytes = append(imgsBytes, buff.Bytes())
		imgsMetadata = append(imgsMetadata, metadata)
	}
	return imgsBytes, imgsMetadata, nil
}

func parseImageFormDataInputsToBytes(req *http.Request) (imgsBytes [][]byte, imgsMetadata []*imageMetadata, err error) {

	logger, _ := logger.GetZapLogger()

	inputs := req.MultipartForm.File["file"]
	for _, content := range inputs {
		file, err := content.Open()
		defer func() {
			err = file.Close()
		}()

		if err != nil {
			logger.Error(fmt.Sprintf("Unable to open file for image %v", err))
			return nil, nil, fmt.Errorf("unable to open file for image")
		}

		buff := new(bytes.Buffer) // pointer
		numBytes, err := buff.ReadFrom(file)
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to read content body from image %v", err))
			return nil, nil, fmt.Errorf("unable to read content body from image")
		}

		if numBytes > int64(config.Config.Server.MaxDataSize*util.MB) {
			return nil, nil, fmt.Errorf(
				"image size must be smaller than %vMB. Got %vMB from image %v",
				config.Config.Server.MaxDataSize,
				float32(numBytes)/float32(util.MB),
				content.Filename,
			)
		}

		img, format, err := image.Decode(buff)
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to decode image: %v", err))
			return nil, nil, fmt.Errorf("unable to decode image")
		}

		// Encode into jpeg to remove alpha channel (hack)
		// This may slightly degrade the image quality
		buff = new(bytes.Buffer)
		err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to process image: %v", err))
			return nil, nil, fmt.Errorf("unable to process image")
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
