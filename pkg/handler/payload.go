package handler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"

	_ "golang.org/x/image/tiff"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/internal/triton"
	"github.com/instill-ai/model-backend/internal/util"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

func parseImageFromURL(url string) (*image.Image, error) {

	logger, _ := logger.GetZapLogger()

	response, err := http.Get(url)
	if err != nil {
		logger.Error(fmt.Sprintf("logUnable to download image at %v. %v", url, err))
		return nil, fmt.Errorf("unable to download image at %v", url)
	}
	defer response.Body.Close()

	buff := new(bytes.Buffer) // pointer
	numBytes, err := buff.ReadFrom(response.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to read content body from image at %v. %v", url, err))
		return nil, fmt.Errorf("unable to read content body from image at %v", url)
	}

	if numBytes > int64(config.Config.Server.MaxDataSize*util.MB) {
		return nil, fmt.Errorf(
			"image size must be smaller than %vMB. Got %vMB",
			config.Config.Server.MaxDataSize,
			float32(numBytes)/float32(util.MB),
		)
	}

	img, _, err := image.Decode(buff)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode image at %v. %v", url, err))
		return nil, fmt.Errorf("unable to decode image at %v", url)
	}

	return &img, nil
}

func parseImageFromBase64(encoded string) (*image.Image, error) {

	logger, _ := logger.GetZapLogger()

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode base64 image. %v", err))
		return nil, fmt.Errorf("unable to decode base64 image")
	}
	numBytes := len(decoded)
	if numBytes > config.Config.Server.MaxDataSize*util.MB {
		return nil, fmt.Errorf(
			"image size must be smaller than %vMB. Got %vMB",
			config.Config.Server.MaxDataSize,
			float32(numBytes)/float32(util.MB),
		)
	}
	img, _, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode base64 image. %v", err))
		return nil, fmt.Errorf("unable to decode base64 image")
	}

	return &img, nil
}

func parseImageRequestInputsToBytes(req *modelPB.TriggerModelInstanceRequest) (inputBytes [][]byte, err error) {
	logger, _ := logger.GetZapLogger()

	for idx, taskInput := range req.TaskInputs {
		var visionInp triton.VisionInput
		switch taskInput.Input.(type) {
		case *modelPB.TaskInput_Classification:
			visionInp = triton.VisionInput{
				ImgUrl:    taskInput.GetClassification().GetImageUrl(),
				ImgBase64: taskInput.GetClassification().GetImageBase64(),
			}
		case *modelPB.TaskInput_Detection:
			visionInp = triton.VisionInput{
				ImgUrl:    taskInput.GetDetection().GetImageUrl(),
				ImgBase64: taskInput.GetDetection().GetImageBase64(),
			}
		case *modelPB.TaskInput_Ocr:
			visionInp = triton.VisionInput{
				ImgUrl:    taskInput.GetOcr().GetImageUrl(),
				ImgBase64: taskInput.GetOcr().GetImageBase64(),
			}
		case *modelPB.TaskInput_Keypoint:
			visionInp = triton.VisionInput{
				ImgUrl:    taskInput.GetKeypoint().GetImageUrl(),
				ImgBase64: taskInput.GetKeypoint().GetImageBase64(),
			}
		case *modelPB.TaskInput_InstanceSegmentation:
			visionInp = triton.VisionInput{
				ImgUrl:    taskInput.GetInstanceSegmentation().GetImageUrl(),
				ImgBase64: taskInput.GetInstanceSegmentation().GetImageBase64(),
			}
		case *modelPB.TaskInput_SemanticSegmentation:
			visionInp = triton.VisionInput{
				ImgUrl:    taskInput.GetSemanticSegmentation().GetImageUrl(),
				ImgBase64: taskInput.GetSemanticSegmentation().GetImageBase64(),
			}
		default:
			return nil, fmt.Errorf("unknown task input type")
		}
		var (
			img *image.Image
			err error
		)

		if visionInp.ImgUrl != "" || visionInp.ImgBase64 != "" {
			if len(visionInp.ImgUrl) > 0 {
				img, err = parseImageFromURL(visionInp.ImgUrl)
				if err != nil {
					logger.Error(fmt.Sprintf("Unable to parse image %v from url. %v", idx, err))
					return nil, fmt.Errorf("unable to parse image %v from url", idx)
				}
			} else if len(visionInp.ImgBase64) > 0 {
				img, err = parseImageFromBase64(visionInp.ImgBase64)
				if err != nil {
					logger.Error(fmt.Sprintf("Unable to parse base64 image %v. %v", idx, err))
					return nil, fmt.Errorf("unable to parse base64 image %v", idx)
				}
			} else {
				return nil, fmt.Errorf(`image %v must define either a "url" or "base64" field. None of them were defined`, idx)
			}

			// Encode into jpeg to remove alpha channel (hack)
			// This may slightly degrade the image quality
			buff := new(bytes.Buffer)
			err = jpeg.Encode(buff, *img, &jpeg.Options{Quality: 100})
			if err != nil {
				logger.Error(fmt.Sprintf("Unable to process image %v. %v", idx, err))
				return nil, fmt.Errorf("unable to process image %v", idx)
			}

			inputBytes = append(inputBytes, buff.Bytes())
		} else {
			return nil, fmt.Errorf("invalid input image")
		}
	}
	return inputBytes, nil
}

func parseTexToImageRequestInputs(req *modelPB.TriggerModelInstanceRequest) (textToImageInput []triton.TextToImageInput, err error) {
	var textToImageInputs []triton.TextToImageInput
	for _, taskInput := range req.TaskInputs {
		steps := int64(util.TEXT_TO_IMAGE_STEPS)
		if taskInput.GetTextToImage().Steps != nil {
			steps = int64(*taskInput.GetTextToImage().Steps)
		}
		cfgScale := float32(util.IMAGE_TO_TEXT_CFG_SCALE)
		if taskInput.GetTextToImage().CfgScale != nil {
			cfgScale = float32(*taskInput.GetTextToImage().CfgScale)
		}
		seed := int64(util.IMAGE_TO_TEXT_SEED)
		if taskInput.GetTextToImage().Seed != nil {
			seed = int64(*taskInput.GetTextToImage().Seed)
		}
		samples := int64(util.IMAGE_TO_TEXT_SAMPLES)
		if taskInput.GetTextToImage().Samples != nil {
			samples = int64(*taskInput.GetTextToImage().Samples)
		}
		textToImageInputs = append(textToImageInputs, triton.TextToImageInput{
			Prompt:   taskInput.GetTextToImage().Prompt,
			Steps:    steps,
			CfgScale: cfgScale,
			Seed:     seed,
			Samples:  samples,
		})
	}
	return textToImageInputs, nil
}

func parseTexGenerationRequestInputs(req *modelPB.TriggerModelInstanceRequest) (textGenerationInput []triton.TextGenerationInput, err error) {
	var textGenerationInputs []triton.TextGenerationInput
	for _, taskInput := range req.TaskInputs {
		outputLen := taskInput.GetTextGeneration().OutputLen
		badWordsList := taskInput.GetTextGeneration().BadWordsList
		stopWordsList := taskInput.GetTextGeneration().StopWordsList
		topK := taskInput.GetTextGeneration().Topk
		seed := taskInput.GetTextGeneration().Seed
		if outputLen == nil {
			outputLen = new(int64)
			*outputLen = 100
		} else if badWordsList == nil {
			badWordsList = new(string)
			*badWordsList = ""
		} else if stopWordsList == nil {
			stopWordsList = new(string)
			*stopWordsList = ""
		} else if topK == nil {
			topK = new(int64)
			*topK = 1
		} else if seed == nil {
			seed = new(int64)
			*seed = 0
		}
		textGenerationInputs = append(textGenerationInputs, triton.TextGenerationInput{
			Prompt:        taskInput.GetTextGeneration().Prompt,
			OutputLen:     *outputLen,
			BadWordsList:  *badWordsList,
			StopWordsList: *stopWordsList,
			TopK:          *topK,
			Seed:          *seed,
		})
	}
	return textGenerationInputs, nil
}

func parseImageFormDataInputsToBytes(req *http.Request) (imgsBytes [][]byte, err error) {

	logger, _ := logger.GetZapLogger()

	inputs := req.MultipartForm.File["file"]
	for _, content := range inputs {
		file, err := content.Open()
		defer func() {
			err = file.Close()
		}()

		if err != nil {
			logger.Error(fmt.Sprintf("Unable to open file for image %v", err))
			return nil, fmt.Errorf("unable to open file for image")
		}

		buff := new(bytes.Buffer) // pointer
		numBytes, err := buff.ReadFrom(file)
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to read content body from image %v", err))
			return nil, fmt.Errorf("unable to read content body from image")
		}

		if numBytes > int64(config.Config.Server.MaxDataSize*util.MB) {
			return nil, fmt.Errorf(
				"image size must be smaller than %vMB. Got %vMB from image %v",
				config.Config.Server.MaxDataSize,
				float32(numBytes)/float32(util.MB),
				content.Filename,
			)
		}

		img, _, err := image.Decode(buff)
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to decode image: %v", err))
			return nil, fmt.Errorf("unable to decode image")
		}

		// Encode into jpeg to remove alpha channel (hack)
		// This may slightly degrade the image quality
		buff = new(bytes.Buffer)
		err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to process image: %v", err))
			return nil, fmt.Errorf("unable to process image")
		}

		imgsBytes = append(imgsBytes, buff.Bytes())
	}

	return imgsBytes, nil
}

func parseImageFormDataTextToImageInputs(req *http.Request) (textToImageInput []triton.TextToImageInput, err error) {
	prompts := req.MultipartForm.Value["prompt"]
	if len(prompts) != 1 {
		return nil, fmt.Errorf("invalid input")
	}

	stepStr := req.MultipartForm.Value["steps"]
	cfgScaleStr := req.MultipartForm.Value["cfg_scale"]
	seedStr := req.MultipartForm.Value["seed"]

	step := 10
	if len(stepStr) > 0 {
		step, err = strconv.Atoi(stepStr[0])
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
	}

	cfgScale := 7.0
	if len(cfgScaleStr) > 0 {
		cfgScale, err = strconv.ParseFloat(cfgScaleStr[0], 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
	}

	seed := 1024
	if len(seedStr) > 0 {
		seed, err = strconv.Atoi(seedStr[0])
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
	}

	return []triton.TextToImageInput{{
		Prompt:   prompts[0],
		Steps:    int64(step),
		CfgScale: float32(cfgScale),
		Seed:     int64(seed),
		Samples:  1,
	}}, nil
}

func parseTextFormDataTextGenerationInputs(req *http.Request) (textGeneration []triton.TextGenerationInput, err error) {
	prompts := req.MultipartForm.Value["prompt"]
	if len(prompts) != 1 {
		return nil, fmt.Errorf("batch size cannot exceed 1")
	}

	outputLenInput := req.MultipartForm.Value["output_len"]
	topKInput := req.MultipartForm.Value["top_k"]
	seedInput := req.MultipartForm.Value["seed"]

	outputLen := 100
	if len(outputLenInput) > 0 {
		outputLen, err = strconv.Atoi(outputLenInput[0])
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
	}

	topK := 1
	if len(topKInput) > 0 {
		topK, err = strconv.Atoi(topKInput[0])
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
	}

	seed := 0
	if len(seedInput) > 0 {
		seed, err = strconv.Atoi(seedInput[0])
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
	}
	// TODO: add support for bad/stop words

	return []triton.TextGenerationInput{{
		Prompt:        prompts[0],
		OutputLen:     int64(outputLen),
		BadWordsList:  "",
		StopWordsList: "",
		TopK:          int64(topK),
		Seed:          int64(seed),
	}}, nil
}
