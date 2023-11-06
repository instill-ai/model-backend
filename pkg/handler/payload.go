package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"

	_ "golang.org/x/image/tiff"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/triton"
	"github.com/instill-ai/model-backend/pkg/utils"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func parseImageFromURL(ctx context.Context, url string) (*image.Image, error) {

	logger, _ := logger.GetZapLogger(ctx)

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

	if numBytes > int64(config.Config.Server.MaxDataSize*utils.MB) {
		return nil, fmt.Errorf(
			"image size must be smaller than %vMB. Got %vMB",
			config.Config.Server.MaxDataSize,
			float32(numBytes)/float32(utils.MB),
		)
	}

	img, _, err := image.Decode(buff)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode image at %v. %v", url, err))
		return nil, fmt.Errorf("unable to decode image at %v", url)
	}

	return &img, nil
}

func parseImageFromBase64(ctx context.Context, encoded string) (*image.Image, error) {

	logger, _ := logger.GetZapLogger(ctx)

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode base64 image. %v", err))
		return nil, fmt.Errorf("unable to decode base64 image")
	}
	numBytes := len(decoded)
	if numBytes > config.Config.Server.MaxDataSize*utils.MB {
		return nil, fmt.Errorf(
			"image size must be smaller than %vMB. Got %vMB",
			config.Config.Server.MaxDataSize,
			float32(numBytes)/float32(utils.MB),
		)
	}
	img, _, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode base64 image. %v", err))
		return nil, fmt.Errorf("unable to decode base64 image")
	}

	return &img, nil
}

func parseImageRequestInputsToBytes(ctx context.Context, req *modelPB.TriggerUserModelRequest) (inputBytes [][]byte, err error) {
	logger, _ := logger.GetZapLogger(ctx)

	for idx, taskInput := range req.TaskInputs {
		var imageInput triton.ImageInput
		switch taskInput.Input.(type) {
		case *modelPB.TaskInput_Classification:
			imageInput = triton.ImageInput{
				ImgUrl:    taskInput.GetClassification().GetImageUrl(),
				ImgBase64: taskInput.GetClassification().GetImageBase64(),
			}
		case *modelPB.TaskInput_Detection:
			imageInput = triton.ImageInput{
				ImgUrl:    taskInput.GetDetection().GetImageUrl(),
				ImgBase64: taskInput.GetDetection().GetImageBase64(),
			}
		case *modelPB.TaskInput_Ocr:
			imageInput = triton.ImageInput{
				ImgUrl:    taskInput.GetOcr().GetImageUrl(),
				ImgBase64: taskInput.GetOcr().GetImageBase64(),
			}
		case *modelPB.TaskInput_Keypoint:
			imageInput = triton.ImageInput{
				ImgUrl:    taskInput.GetKeypoint().GetImageUrl(),
				ImgBase64: taskInput.GetKeypoint().GetImageBase64(),
			}
		case *modelPB.TaskInput_InstanceSegmentation:
			imageInput = triton.ImageInput{
				ImgUrl:    taskInput.GetInstanceSegmentation().GetImageUrl(),
				ImgBase64: taskInput.GetInstanceSegmentation().GetImageBase64(),
			}
		case *modelPB.TaskInput_SemanticSegmentation:
			imageInput = triton.ImageInput{
				ImgUrl:    taskInput.GetSemanticSegmentation().GetImageUrl(),
				ImgBase64: taskInput.GetSemanticSegmentation().GetImageBase64(),
			}
		case *modelPB.TaskInput_TextGeneration:
			imageInput = triton.ImageInput{
				ImgUrl:    taskInput.GetTextGeneration().GetPromptImageUrl(),
				ImgBase64: taskInput.GetTextGeneration().GetPromptImageBase64(),
			}
		default:
			return nil, fmt.Errorf("unknown task input type")
		}
		var (
			img *image.Image
			err error
		)

		if imageInput.ImgUrl != "" || imageInput.ImgBase64 != "" {
			if len(imageInput.ImgUrl) > 0 {
				img, err = parseImageFromURL(ctx, imageInput.ImgUrl)
				if err != nil {
					logger.Error(fmt.Sprintf("Unable to parse image %v from url. %v", idx, err))
					return nil, fmt.Errorf("unable to parse image %v from url", idx)
				}
			} else if len(imageInput.ImgBase64) > 0 {
				img, err = parseImageFromBase64(ctx, imageInput.ImgBase64)
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

func parseTexToImageRequestInputs(req *modelPB.TriggerUserModelRequest) (textToImageInput *triton.TextToImageInput, err error) {
	if len(req.TaskInputs) > 1 {
		return nil, fmt.Errorf("text to image only support single batch")
	}

	for _, taskInput := range req.TaskInputs {
		steps := utils.TEXT_TO_IMAGE_STEPS
		if taskInput.GetTextToImage().Steps != nil {
			steps = *taskInput.GetTextToImage().Steps
		}
		cfgScale := float32(utils.IMAGE_TO_TEXT_CFG_SCALE)
		if taskInput.GetTextToImage().CfgScale != nil {
			cfgScale = float32(*taskInput.GetTextToImage().CfgScale)
		}
		seed := utils.IMAGE_TO_TEXT_SEED
		if taskInput.GetTextToImage().Seed != nil {
			seed = *taskInput.GetTextToImage().Seed
		}
		samples := utils.IMAGE_TO_TEXT_SAMPLES
		if taskInput.GetTextToImage().Samples != nil {
			samples = *taskInput.GetTextToImage().Samples
		}
		if samples > 1 {
			return nil, fmt.Errorf("we only allow samples=1 for now and will improve to allow the generation of multiple samples in the future")
		}
		textToImageInput = &triton.TextToImageInput{
			Prompt:   taskInput.GetTextToImage().Prompt,
			Steps:    steps,
			CfgScale: cfgScale,
			Seed:     seed,
			Samples:  samples,
		}
	}
	return textToImageInput, nil
}

func parseTexGenerationRequestInputs(ctx context.Context, req *modelPB.TriggerUserModelRequest) (textGenerationInput *triton.TextGenerationInput, err error) {
	pargedImages, parsedImageErr := parseImageRequestInputsToBytes(ctx, req)
	for idx, taskInput := range req.TaskInputs {

		maxNewTokens := utils.TEXT_GENERATION_MAX_NEW_TOKENS
		if taskInput.GetTextGeneration().MaxNewTokens != nil {
			maxNewTokens = *taskInput.GetTextGeneration().MaxNewTokens
		}
		stopWordsList := string("")
		if taskInput.GetTextGeneration().StopWordsList != nil {
			stopWordsList = *taskInput.GetTextGeneration().StopWordsList
		}
		temperature := utils.TEXT_GENERATION_TEMPERATURE
		if taskInput.GetTextGeneration().Temperature != nil {
			temperature = *taskInput.GetTextGeneration().Temperature
		}
		topK := utils.TEXT_GENERATION_TOP_K
		if taskInput.GetTextGeneration().TopK != nil {
			topK = *taskInput.GetTextGeneration().TopK
		}
		seed := utils.TEXT_GENERATION_SEED
		if taskInput.GetTextGeneration().Seed != nil {
			seed = *taskInput.GetTextGeneration().Seed
		}
		extraParams := string("")
		if taskInput.GetTextGeneration().ExtraParams != nil {
			extraParams = *taskInput.GetTextGeneration().ExtraParams
		}

		// Handling Image Input
		var inputBytes []byte
		if parsedImageErr == nil {
			inputBytes = pargedImages[idx]
		}

		textGenerationInput = &triton.TextGenerationInput{
			Prompt:        taskInput.GetTextGeneration().Prompt,
			PromptImage:   string(inputBytes),
			MaxNewTokens:  maxNewTokens,
			StopWordsList: stopWordsList,
			Temperature:   temperature,
			TopK:          topK,
			Seed:          seed,
			ExtraParams:   extraParams,
		}
	}
	return textGenerationInput, nil
}

func parseImageFormDataInputsToBytes(req *http.Request) (imgsBytes [][]byte, err error) {

	logger, _ := logger.GetZapLogger(req.Context())

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

		if numBytes > int64(config.Config.Server.MaxDataSize*utils.MB) {
			return nil, fmt.Errorf(
				"image size must be smaller than %vMB. Got %vMB from image %v",
				config.Config.Server.MaxDataSize,
				float32(numBytes)/float32(utils.MB),
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

func parseImageFormDataTextToImageInputs(req *http.Request) (textToImageInput *triton.TextToImageInput, err error) {
	prompts := req.MultipartForm.Value["prompt"]
	if len(prompts) == 0 {
		return nil, fmt.Errorf("missing prompt input")
	}
	if len(prompts) > 1 {
		return nil, fmt.Errorf("invalid prompt input, only support a single prompt")
	}
	stepStr := req.MultipartForm.Value["steps"]
	cfgScaleStr := req.MultipartForm.Value["cfg_scale"]
	seedStr := req.MultipartForm.Value["seed"]
	samplesStr := req.MultipartForm.Value["samples"]

	if len(stepStr) > 1 {
		return nil, fmt.Errorf("invalid steps input, only support a single steps")
	}
	if len(cfgScaleStr) > 1 {
		return nil, fmt.Errorf("invalid cfg_scale input, only support a single cfg_scale")
	}
	if len(seedStr) > 1 {
		return nil, fmt.Errorf("invalid seed input, only support a single seed")
	}
	if len(samplesStr) > 1 {
		return nil, fmt.Errorf("invalid samples input, only support a single samples")
	}

	step := utils.TEXT_TO_IMAGE_STEPS
	if len(stepStr) > 0 {
		parseStep, err := strconv.ParseInt(stepStr[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid step input %w", err)
		}
		step = int32(parseStep)
	}

	cfgScale := float64(utils.IMAGE_TO_TEXT_CFG_SCALE)
	if len(cfgScaleStr) > 0 {
		cfgScale, err = strconv.ParseFloat(cfgScaleStr[0], 32)
		if err != nil {
			return nil, fmt.Errorf("invalid cfgScale input %w", err)
		}
	}

	seed := utils.IMAGE_TO_TEXT_SEED
	if len(seedStr) > 0 {
		parseSeed, err := strconv.ParseInt(seedStr[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid seed input %w", err)
		}
		seed = int32(parseSeed)
	}

	samples := utils.IMAGE_TO_TEXT_SAMPLES
	if len(samplesStr) > 0 {
		parseSamples, err := strconv.ParseInt(samplesStr[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid samples input %w", err)
		}
		samples = int32(parseSamples)
	}

	if samples > 1 {
		return nil, fmt.Errorf("we only allow samples=1 for now and will improve to allow the generation of multiple samples in the future")
	}

	return &triton.TextToImageInput{
		Prompt:   prompts[0],
		Steps:    step,
		CfgScale: float32(cfgScale),
		Seed:     seed,
		Samples:  samples,
	}, nil
}

func parseTextFormDataTextGenerationInputs(req *http.Request) (textGeneration *triton.TextGenerationInput, err error) {
	prompts := req.MultipartForm.Value["prompt"]
	if len(prompts) != 1 {
		return nil, fmt.Errorf("only support batchsize 1")
	}
	stopWordsListInput := req.MultipartForm.Value["stop_words_list"]
	maxNewTokenInput := req.MultipartForm.Value["max_new_token"]
	temperatureInput := req.MultipartForm.Value["temperature"]
	topKInput := req.MultipartForm.Value["topk"]
	seedInput := req.MultipartForm.Value["seed"]
	extraParamsInput := req.MultipartForm.Value["extra_params"]

	stopWordsList := string("")
	if len(stopWordsListInput) > 0 {
		stopWordsList = stopWordsListInput[0]
	}

	maxNewTokens := utils.TEXT_GENERATION_MAX_NEW_TOKENS
	if len(maxNewTokenInput) > 0 {
		parseMaxNewToken, err := strconv.ParseInt(maxNewTokenInput[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		maxNewTokens = int32(parseMaxNewToken)
	}

	temperature := utils.TEXT_GENERATION_TEMPERATURE
	if len(temperatureInput) > 0 {
		parseTemperature, err := strconv.ParseFloat(temperatureInput[0], 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		temperature = float32(parseTemperature)
	}

	topK := utils.TEXT_GENERATION_TOP_K
	if len(topKInput) > 0 {
		parseTopK, err := strconv.ParseInt(topKInput[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		topK = int32(parseTopK)
	}

	seed := utils.TEXT_GENERATION_SEED
	if len(seedInput) > 0 {
		parseSeed, err := strconv.ParseInt(seedInput[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		seed = int32(parseSeed)
	}

	extraParams := ""
	if len(extraParamsInput) > 0 {
		extraParams = extraParamsInput[0]
	}

	parsedImages, err := parseImageFormDataInputsToBytes(req)
	var promptImage string
	if err != nil && len(parsedImages) == 1 {
		promptImage = string(parsedImages[0])
	}

	return &triton.TextGenerationInput{
		Prompt:        prompts[0],
		PromptImage:   promptImage,
		MaxNewTokens:  maxNewTokens,
		StopWordsList: stopWordsList,
		Temperature:   temperature,
		TopK:          topK,
		Seed:          seed,
		ExtraParams:   extraParams,
	}, nil
}
