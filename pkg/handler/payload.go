package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"strconv"
	"strings"

	_ "golang.org/x/image/tiff"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/ray"
	"github.com/instill-ai/model-backend/pkg/utils"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func trimBase64Mime(b64 string) string {
	splitB64 := strings.Split(b64, ",")
	return splitB64[len(splitB64)-1]
}

func parseImageFromURL(ctx context.Context, url string) (image.Image, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	response, err := http.DefaultClient.Do(req)
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

	if numBytes > int64(config.Config.Server.MaxDataSize*constant.MB) {
		return nil, fmt.Errorf(
			"image size must be smaller than %vMB. Got %vMB",
			config.Config.Server.MaxDataSize,
			float32(numBytes)/float32(constant.MB),
		)
	}

	img, _, err := image.Decode(buff)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode image at %v. %v", url, err))
		return nil, fmt.Errorf("unable to decode image at %v", url)
	}

	return img, nil
}

func parseImageFromBase64(ctx context.Context, encoded string) (image.Image, error) {

	logger, _ := custom_logger.GetZapLogger(ctx)

	encoded = trimBase64Mime(encoded)

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode base64 image. %v", err))
		return nil, fmt.Errorf("unable to decode base64 image")
	}
	numBytes := len(decoded)
	if numBytes > config.Config.Server.MaxDataSize*constant.MB {
		return nil, fmt.Errorf(
			"image size must be smaller than %vMB. Got %vMB",
			config.Config.Server.MaxDataSize,
			float32(numBytes)/float32(constant.MB),
		)
	}
	img, _, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		logger.Error(fmt.Sprintf("Unable to decode base64 image. %v", err))
		return nil, fmt.Errorf("unable to decode base64 image")
	}

	return img, nil
}

func parseImageInputToByte(ctx context.Context, imageInput ray.ImageInput) (encodedImg []byte, err error) {
	var img image.Image
	if imageInput.ImgURL != "" || imageInput.ImgBase64 != "" {
		logger, _ := custom_logger.GetZapLogger(ctx)
		if imageInput.ImgURL != "" {
			img, err = parseImageFromURL(ctx, imageInput.ImgURL)
			if err != nil {
				logger.Error(fmt.Sprintf("Unable to parse image from url. %v", err))
				return nil, fmt.Errorf("unable to parse image from url")
			}
		} else if imageInput.ImgBase64 != "" {
			img, err = parseImageFromBase64(ctx, imageInput.ImgBase64)
			if err != nil {
				logger.Error(fmt.Sprintf("Unable to parse base64 image. %v", err))
				return nil, fmt.Errorf("unable to parse base64 image")
			}
		} else {
			return nil, fmt.Errorf(`image must define either a "url" or "base64" field. None of them were defined`)
		}

		// Encode into jpeg to remove alpha channel (hack)
		// This may slightly degrade the image quality
		buff := new(bytes.Buffer)
		err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
		if err != nil {
			logger.Error(fmt.Sprintf("Unable to process image. %v", err))
			return nil, fmt.Errorf("unable to process image ")
		}

		// inputBytes = append(inputBytes, buff.Bytes())
		return buff.Bytes(), nil
	} else {
		return nil, fmt.Errorf("invalid input image")
	}
}
func parseImageRequestInputsToBytes(ctx context.Context, req TriggerNamespaceModelRequestInterface) (inputBytes [][]byte, err error) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	for idx, taskInput := range req.GetTaskInputs() {
		var imageInput ray.ImageInput
		switch taskInput.Input.(type) {
		case *modelpb.TaskInput_Classification:
			imageInput = ray.ImageInput{
				ImgURL:    taskInput.GetClassification().GetImageUrl(),
				ImgBase64: taskInput.GetClassification().GetImageBase64(),
			}
		case *modelpb.TaskInput_Detection:
			imageInput = ray.ImageInput{
				ImgURL:    taskInput.GetDetection().GetImageUrl(),
				ImgBase64: taskInput.GetDetection().GetImageBase64(),
			}
		case *modelpb.TaskInput_Ocr:
			imageInput = ray.ImageInput{
				ImgURL:    taskInput.GetOcr().GetImageUrl(),
				ImgBase64: taskInput.GetOcr().GetImageBase64(),
			}
		case *modelpb.TaskInput_Keypoint:
			imageInput = ray.ImageInput{
				ImgURL:    taskInput.GetKeypoint().GetImageUrl(),
				ImgBase64: taskInput.GetKeypoint().GetImageBase64(),
			}
		case *modelpb.TaskInput_InstanceSegmentation:
			imageInput = ray.ImageInput{
				ImgURL:    taskInput.GetInstanceSegmentation().GetImageUrl(),
				ImgBase64: taskInput.GetInstanceSegmentation().GetImageBase64(),
			}
		case *modelpb.TaskInput_SemanticSegmentation:
			imageInput = ray.ImageInput{
				ImgURL:    taskInput.GetSemanticSegmentation().GetImageUrl(),
				ImgBase64: taskInput.GetSemanticSegmentation().GetImageBase64(),
			}
		case *modelpb.TaskInput_TextToImage:
			imageInput = ray.ImageInput{
				ImgURL:    taskInput.GetTextToImage().GetPromptImageUrl(),
				ImgBase64: taskInput.GetTextToImage().GetPromptImageBase64(),
			}
		case *modelpb.TaskInput_ImageToImage:
			imageInput = ray.ImageInput{
				ImgURL:    taskInput.GetImageToImage().GetPromptImageUrl(),
				ImgBase64: taskInput.GetImageToImage().GetPromptImageBase64(),
			}
		default:
			return nil, fmt.Errorf("unknown task input type")
		}
		// encodedImage, err := parseImageInputToByte(ctx, imageInput)
		// if err != nil {
		// 	logger.Error(fmt.Sprintf("Unable to process image %v. %v", idx, err))
		// 	return nil, fmt.Errorf("unable to process image %v", idx)
		// }
		// inputBytes = append(inputBytes, encodedImage)
		var (
			img image.Image
			err error
		)

		if imageInput.ImgURL != "" || imageInput.ImgBase64 != "" {
			if imageInput.ImgURL != "" {
				img, err = parseImageFromURL(ctx, imageInput.ImgURL)
				if err != nil {
					logger.Error(fmt.Sprintf("Unable to parse image %v from url. %v", idx, err))
					return nil, fmt.Errorf("unable to parse image %v from url", idx)
				}
			} else if imageInput.ImgBase64 != "" {
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
			err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
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

func parseTexToImageRequestInputs(ctx context.Context, req TriggerNamespaceModelRequestInterface) (textToImageInput *ray.TextToImageInput, err error) {
	if len(req.GetTaskInputs()) > 1 {
		return nil, fmt.Errorf("text to image only support single batch")
	}
	pargedImages, parsedImageErr := parseImageRequestInputsToBytes(ctx, req)
	for idx, taskInput := range req.GetTaskInputs() {
		steps := utils.ToImageSteps
		if taskInput.GetTextToImage().Steps != nil {
			steps = *taskInput.GetTextToImage().Steps
		}
		cfgScale := utils.ToImageCFGScale
		if taskInput.GetTextToImage().CfgScale != nil {
			cfgScale = *taskInput.GetTextToImage().CfgScale
		}
		seed := utils.ToImageSeed
		if taskInput.GetTextToImage().Seed != nil {
			seed = *taskInput.GetTextToImage().Seed
		}
		samples := utils.ToImageSamples
		if taskInput.GetTextToImage().Samples != nil {
			samples = *taskInput.GetTextToImage().Samples
		}
		if samples > 1 {
			return nil, fmt.Errorf("we only allow samples=1 for now and will improve to allow the generation of multiple samples in the future")
		}

		extraParams := string("")
		if taskInput.GetTextToImage().ExtraParams != nil {
			jsonData, err := json.Marshal(taskInput.GetTextToImage().ExtraParams)
			if err != nil {
				log.Fatalf("Error marshaling to JSON: %v", err)
			} else {
				extraParams = string(jsonData)
			}
		}

		// Handling Image Input
		var inputBytes []byte
		if parsedImageErr == nil {
			inputBytes = pargedImages[idx]
		}
		textToImageInput = &ray.TextToImageInput{
			Prompt:      taskInput.GetTextToImage().Prompt,
			PromptImage: string(inputBytes),
			Steps:       steps,
			CfgScale:    cfgScale,
			Seed:        seed,
			Samples:     samples,
			ExtraParams: extraParams,
		}
	}
	return textToImageInput, nil
}

func parseImageToImageRequestInputs(ctx context.Context, req TriggerNamespaceModelRequestInterface) (imageToImageInput *ray.ImageToImageInput, err error) {
	if len(req.GetTaskInputs()) > 1 {
		return nil, fmt.Errorf("text to image only support single batch")
	}
	pargedImages, parsedImageErr := parseImageRequestInputsToBytes(ctx, req)
	for idx, taskInput := range req.GetTaskInputs() {
		steps := utils.ToImageSteps
		if taskInput.GetImageToImage().Steps != nil {
			steps = *taskInput.GetImageToImage().Steps
		}
		cfgScale := utils.ToImageCFGScale
		if taskInput.GetImageToImage().CfgScale != nil {
			cfgScale = *taskInput.GetImageToImage().CfgScale
		}
		seed := utils.ToImageSeed
		if taskInput.GetImageToImage().Seed != nil {
			seed = *taskInput.GetImageToImage().Seed
		}
		samples := utils.ToImageSamples
		if taskInput.GetImageToImage().Samples != nil {
			samples = *taskInput.GetImageToImage().Samples
		}
		if samples > 1 {
			return nil, fmt.Errorf("we only allow samples=1 for now and will improve to allow the generation of multiple samples in the future")
		}

		extraParams := string("")
		if taskInput.GetImageToImage().ExtraParams != nil {
			jsonData, err := json.Marshal(taskInput.GetImageToImage().ExtraParams)
			if err != nil {
				log.Fatalf("Error marshaling to JSON: %v", err)
			} else {
				extraParams = string(jsonData)
			}
		}
		prompt := string("")
		if taskInput.GetImageToImage().Prompt != nil {
			prompt = *taskInput.GetImageToImage().Prompt
		}

		// Handling Image Input
		var inputBytes []byte
		if parsedImageErr == nil {
			inputBytes = pargedImages[idx]
		}
		imageToImageInput = &ray.ImageToImageInput{
			Prompt:      prompt,
			PromptImage: string(inputBytes),
			Steps:       steps,
			CfgScale:    cfgScale,
			Seed:        seed,
			Samples:     samples,
			ExtraParams: extraParams,
		}
	}
	return imageToImageInput, nil
}

func parseTexGenerationRequestInputs(ctx context.Context, req TriggerNamespaceModelRequestInterface) (textGenerationInput *ray.TextGenerationInput, err error) {
	for _, taskInput := range req.GetTaskInputs() {

		maxNewTokens := utils.TextGenerationMaxNewTokens
		if taskInput.GetTextGeneration().MaxNewTokens != nil {
			maxNewTokens = *taskInput.GetTextGeneration().MaxNewTokens
		}
		temperature := utils.TextGenerationTemperature
		if taskInput.GetTextGeneration().Temperature != nil {
			temperature = *taskInput.GetTextGeneration().Temperature
		}
		topK := utils.TextGenerationTopK
		if taskInput.GetTextGeneration().TopK != nil {
			topK = *taskInput.GetTextGeneration().TopK
		}
		seed := utils.TextGenerationSeed
		if taskInput.GetTextGeneration().Seed != nil {
			seed = *taskInput.GetTextGeneration().Seed
		}
		extraParams := string("")
		if taskInput.GetTextGeneration().ExtraParams != nil {
			jsonData, err := json.Marshal(taskInput.GetTextGeneration().ExtraParams)
			if err != nil {
				log.Fatalf("Error marshaling to JSON in ExtraParams field: %v", err)
			} else {
				extraParams = string(jsonData)
			}
		}

		chatHistory := string("")
		if taskInput.GetTextGeneration().ChatHistory != nil {
			jsonData, err := json.Marshal(taskInput.GetTextGeneration().ChatHistory)
			if err != nil {
				log.Fatalf("Error marshaling to JSON in ChatHistory field: %v", err)
			} else {
				chatHistory = string(jsonData)
			}
		}

		systemMessage := string("")
		if taskInput.GetTextGeneration().SystemMessage != nil {
			systemMessage = *taskInput.GetTextGeneration().SystemMessage
		}

		promptImages := string("")
		if taskInput.GetTextGeneration().PromptImages != nil {
			var promptImagesArr [][]byte
			for _, promptImageStruct := range taskInput.GetTextGeneration().PromptImages {
				imageInput := ray.ImageInput{
					ImgURL:    promptImageStruct.GetPromptImageUrl(),
					ImgBase64: promptImageStruct.GetPromptImageBase64(),
				}
				encodedImage, err := parseImageInputToByte(ctx, imageInput)
				if err != nil {
					return nil, err
				}
				promptImagesArr = append(promptImagesArr, encodedImage)
			}
			jsonData, err := json.Marshal(promptImagesArr)
			if err != nil {
				log.Fatalf("Error marshaling to JSON in promptImages field: %v", err)
			} else {
				promptImages = string(jsonData)
			}
		}

		textGenerationInput = &ray.TextGenerationInput{
			Prompt:        taskInput.GetTextGeneration().Prompt,
			PromptImages:  promptImages,
			ChatHistory:   chatHistory,
			SystemMessage: systemMessage,
			MaxNewTokens:  maxNewTokens,
			Temperature:   temperature,
			TopK:          topK,
			Seed:          seed,
			ExtraParams:   extraParams,
		}
	}
	return textGenerationInput, nil
}

func parseTexGenerationChatRequestInputs(ctx context.Context, req TriggerNamespaceModelRequestInterface) (textGenerationChatInput *ray.TextGenerationChatInput, err error) {
	for _, taskInput := range req.GetTaskInputs() {
		maxNewTokens := utils.TextGenerationMaxNewTokens
		if taskInput.GetTextGenerationChat().MaxNewTokens != nil {
			maxNewTokens = *taskInput.GetTextGenerationChat().MaxNewTokens
		}
		temperature := utils.TextGenerationTemperature
		if taskInput.GetTextGenerationChat().Temperature != nil {
			temperature = *taskInput.GetTextGenerationChat().Temperature
		}
		topK := utils.TextGenerationTopK
		if taskInput.GetTextGenerationChat().TopK != nil {
			topK = *taskInput.GetTextGenerationChat().TopK
		}
		seed := utils.TextGenerationSeed
		if taskInput.GetTextGenerationChat().Seed != nil {
			seed = *taskInput.GetTextGenerationChat().Seed
		}
		extraParams := string("")
		if taskInput.GetTextGenerationChat().ExtraParams != nil {
			jsonData, err := json.Marshal(taskInput.GetTextGenerationChat().ExtraParams)
			if err != nil {
				log.Fatalf("Error marshaling to JSON in ExtraParams field: %v", err)
			} else {
				extraParams = string(jsonData)
			}
		}

		chatHistory := string("")
		if taskInput.GetTextGenerationChat().ChatHistory != nil {
			jsonData, err := json.Marshal(taskInput.GetTextGenerationChat().ChatHistory)
			if err != nil {
				log.Fatalf("Error marshaling to JSON in ChatHistory field: %v", err)
			} else {
				chatHistory = string(jsonData)
			}
		}

		systemMessage := string("")
		if taskInput.GetTextGenerationChat().SystemMessage != nil {
			systemMessage = *taskInput.GetTextGenerationChat().SystemMessage
		}

		promptImages := string("")
		if taskInput.GetTextGenerationChat().PromptImages != nil {
			var promptImagesArr [][]byte
			for _, promptImageStruct := range taskInput.GetTextGenerationChat().PromptImages {
				imageInput := ray.ImageInput{
					ImgURL:    promptImageStruct.GetPromptImageUrl(),
					ImgBase64: promptImageStruct.GetPromptImageBase64(),
				}
				encodedImage, err := parseImageInputToByte(ctx, imageInput)
				if err != nil {
					return nil, err
				}
				promptImagesArr = append(promptImagesArr, encodedImage)
			}
			jsonData, err := json.Marshal(promptImagesArr)
			if err != nil {
				log.Fatalf("Error marshaling to JSON in promptImages field: %v", err)
			} else {
				promptImages = string(jsonData)
			}
		}
		textGenerationChatInput = &ray.TextGenerationChatInput{
			Prompt:        taskInput.GetTextGenerationChat().Prompt,
			PromptImages:  promptImages,
			ChatHistory:   chatHistory,
			SystemMessage: systemMessage,
			MaxNewTokens:  maxNewTokens,
			Temperature:   temperature,
			TopK:          topK,
			Seed:          seed,
			ExtraParams:   extraParams,
		}
	}
	return textGenerationChatInput, nil
}

func parseVisualQuestionAnsweringRequestInputs(ctx context.Context, req TriggerNamespaceModelRequestInterface) (visualQuestionAnsweringInput *ray.VisualQuestionAnsweringInput, err error) {
	for _, taskInput := range req.GetTaskInputs() {

		maxNewTokens := utils.TextGenerationMaxNewTokens
		if taskInput.GetVisualQuestionAnswering().MaxNewTokens != nil {
			maxNewTokens = *taskInput.GetVisualQuestionAnswering().MaxNewTokens
		}
		temperature := utils.TextGenerationTemperature
		if taskInput.GetVisualQuestionAnswering().Temperature != nil {
			temperature = *taskInput.GetVisualQuestionAnswering().Temperature
		}
		topK := utils.TextGenerationTopK
		if taskInput.GetVisualQuestionAnswering().TopK != nil {
			topK = *taskInput.GetVisualQuestionAnswering().TopK
		}
		seed := utils.TextGenerationSeed
		if taskInput.GetVisualQuestionAnswering().Seed != nil {
			seed = *taskInput.GetVisualQuestionAnswering().Seed
		}
		extraParams := string("")
		if taskInput.GetVisualQuestionAnswering().ExtraParams != nil {
			jsonData, err := json.Marshal(taskInput.GetVisualQuestionAnswering().ExtraParams)
			if err != nil {
				log.Fatalf("Error marshaling to JSON in ExtraParams field: %v", err)
			} else {
				extraParams = string(jsonData)
			}
		}

		chatHistory := string("")
		if taskInput.GetVisualQuestionAnswering().ChatHistory != nil {
			jsonData, err := json.Marshal(taskInput.GetVisualQuestionAnswering().ChatHistory)
			if err != nil {
				log.Fatalf("Error marshaling to JSON in ChatHistory field: %v", err)
			} else {
				chatHistory = string(jsonData)
			}
		}

		systemMessage := string("")
		if taskInput.GetVisualQuestionAnswering().SystemMessage != nil {
			systemMessage = *taskInput.GetVisualQuestionAnswering().SystemMessage
		}

		promptImages := string("")
		if taskInput.GetVisualQuestionAnswering().PromptImages != nil {
			var promptImagesArr [][]byte
			for _, promptImageStruct := range taskInput.GetVisualQuestionAnswering().PromptImages {
				imageInput := ray.ImageInput{
					ImgURL:    promptImageStruct.GetPromptImageUrl(),
					ImgBase64: promptImageStruct.GetPromptImageBase64(),
				}
				encodedImage, err := parseImageInputToByte(ctx, imageInput)
				if err != nil {
					return nil, err
				}
				promptImagesArr = append(promptImagesArr, encodedImage)
			}
			jsonData, err := json.Marshal(promptImagesArr)
			if err != nil {
				log.Fatalf("Error marshaling to JSON in promptImages field: %v", err)
			} else {
				promptImages = string(jsonData)
			}
		}

		visualQuestionAnsweringInput = &ray.VisualQuestionAnsweringInput{
			Prompt:        taskInput.GetVisualQuestionAnswering().Prompt,
			PromptImages:  promptImages,
			ChatHistory:   chatHistory,
			SystemMessage: systemMessage,
			MaxNewTokens:  maxNewTokens,
			Temperature:   temperature,
			TopK:          topK,
			Seed:          seed,
			ExtraParams:   extraParams,
		}
	}
	return visualQuestionAnsweringInput, nil
}

func parseImageFormDataInputsToBytes(req *http.Request) (imgsBytes [][]byte, err error) {

	logger, _ := custom_logger.GetZapLogger(req.Context())

	inputs := req.MultipartForm.File["file"]
	for _, content := range inputs {
		file, err := content.Open()

		if err != nil {
			if err := file.Close(); err != nil {
				return nil, err
			}
			logger.Error(fmt.Sprintf("Unable to open file for image %v", err))
			return nil, fmt.Errorf("unable to open file for image")
		}

		buff := new(bytes.Buffer) // pointer
		numBytes, err := buff.ReadFrom(file)
		if err != nil {
			if err := file.Close(); err != nil {
				return nil, err
			}
			logger.Error(fmt.Sprintf("Unable to read content body from image %v", err))
			return nil, fmt.Errorf("unable to read content body from image")
		}
		if err := file.Close(); err != nil {
			return nil, err
		}

		if numBytes > int64(config.Config.Server.MaxDataSize*constant.MB) {
			return nil, fmt.Errorf(
				"image size must be smaller than %vMB. Got %vMB from image %v",
				config.Config.Server.MaxDataSize,
				float32(numBytes)/float32(constant.MB),
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

func parseImageFormDataTextToImageInputs(req *http.Request) (textToImageInput *ray.TextToImageInput, err error) {
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
	extraParamsInput := req.MultipartForm.Value["extra_params"]

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

	step := utils.ToImageSteps
	if len(stepStr) > 0 {
		parseStep, err := strconv.ParseInt(stepStr[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid step input %w", err)
		}
		step = int32(parseStep)
	}

	cfgScale := float64(utils.ToImageCFGScale)
	if len(cfgScaleStr) > 0 {
		cfgScale, err = strconv.ParseFloat(cfgScaleStr[0], 32)
		if err != nil {
			return nil, fmt.Errorf("invalid cfgScale input %w", err)
		}
	}

	seed := utils.ToImageSeed
	if len(seedStr) > 0 {
		parseSeed, err := strconv.ParseInt(seedStr[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid seed input %w", err)
		}
		seed = int32(parseSeed)
	}

	samples := utils.ToImageSamples
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

	extraParams := ""
	if len(extraParamsInput) > 0 {
		extraParams = extraParamsInput[0]
	}

	parsedImages, err := parseImageFormDataInputsToBytes(req)
	var promptImage string
	if err != nil && len(parsedImages) == 1 {
		promptImage = string(parsedImages[0])
	}

	return &ray.TextToImageInput{
		Prompt:      prompts[0],
		PromptImage: promptImage,
		Steps:       step,
		CfgScale:    float32(cfgScale),
		Seed:        seed,
		Samples:     samples,
		ExtraParams: extraParams,
	}, nil
}

func parseImageFormDataImageToImageInputs(req *http.Request) (imageToImageInput *ray.ImageToImageInput, err error) {
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
	extraParamsInput := req.MultipartForm.Value["extra_params"]

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

	step := utils.ToImageSteps
	if len(stepStr) > 0 {
		parseStep, err := strconv.ParseInt(stepStr[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid step input %w", err)
		}
		step = int32(parseStep)
	}

	cfgScale := float64(utils.ToImageCFGScale)
	if len(cfgScaleStr) > 0 {
		cfgScale, err = strconv.ParseFloat(cfgScaleStr[0], 32)
		if err != nil {
			return nil, fmt.Errorf("invalid cfgScale input %w", err)
		}
	}

	seed := utils.ToImageSeed
	if len(seedStr) > 0 {
		parseSeed, err := strconv.ParseInt(seedStr[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid seed input %w", err)
		}
		seed = int32(parseSeed)
	}

	samples := utils.ToImageSamples
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

	extraParams := ""
	if len(extraParamsInput) > 0 {
		extraParams = extraParamsInput[0]
	}

	parsedImages, err := parseImageFormDataInputsToBytes(req)
	var promptImage string
	if err != nil && len(parsedImages) == 1 {
		promptImage = string(parsedImages[0])
	}

	return &ray.ImageToImageInput{
		Prompt:      prompts[0],
		PromptImage: promptImage,
		Steps:       step,
		CfgScale:    float32(cfgScale),
		Seed:        seed,
		Samples:     samples,
		ExtraParams: extraParams,
	}, nil
}

func parseTextFormDataTextGenerationInputs(req *http.Request) (textGeneration *ray.TextGenerationInput, err error) {
	prompts := req.MultipartForm.Value["prompt"]
	if len(prompts) != 1 {
		return nil, fmt.Errorf("only support batchsize 1")
	}
	maxNewTokenInput := req.MultipartForm.Value["max_new_tokens"]
	temperatureInput := req.MultipartForm.Value["temperature"]
	topKInput := req.MultipartForm.Value["top_k"]
	seedInput := req.MultipartForm.Value["seed"]
	extraParamsInput := req.MultipartForm.Value["extra_params"]
	chatHistoryInput := req.MultipartForm.Value["chat_history"]
	systemMessageInput := req.MultipartForm.Value["system_message"]

	maxNewTokens := utils.TextGenerationMaxNewTokens
	if len(maxNewTokenInput) > 0 {
		parseMaxNewToken, err := strconv.ParseInt(maxNewTokenInput[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		maxNewTokens = int32(parseMaxNewToken)
	}

	temperature := utils.TextGenerationTemperature
	if len(temperatureInput) > 0 {
		parseTemperature, err := strconv.ParseFloat(temperatureInput[0], 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		temperature = float32(parseTemperature)
	}

	topK := utils.TextGenerationTopK
	if len(topKInput) > 0 {
		parseTopK, err := strconv.ParseInt(topKInput[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		topK = int32(parseTopK)
	}

	seed := utils.TextGenerationSeed
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
	chatHistory := ""
	if len(chatHistoryInput) > 0 {
		chatHistory = chatHistoryInput[0]
	}
	systemMessage := ""
	if len(systemMessageInput) > 0 {
		systemMessage = systemMessageInput[0]
	}

	promptImages := ""
	parsedImages, err := parseImageFormDataInputsToBytes(req)
	if err == nil {
		jsonData, err := json.Marshal(parsedImages)
		if err != nil {
			log.Fatalf("Error marshaling to JSON: %v", err)
		} else {
			promptImages = string(jsonData)
		}
	}

	return &ray.TextGenerationInput{
		Prompt:        prompts[0],
		PromptImages:  promptImages,
		ChatHistory:   chatHistory,
		SystemMessage: systemMessage,
		MaxNewTokens:  maxNewTokens,
		Temperature:   temperature,
		TopK:          topK,
		Seed:          seed,
		ExtraParams:   extraParams,
	}, nil
}

func parseTextFormDataTextGenerationChatInputs(req *http.Request) (textGenerationChat *ray.TextGenerationChatInput, err error) {
	prompts := req.MultipartForm.Value["prompt"]
	if len(prompts) != 1 {
		return nil, fmt.Errorf("only support batchsize 1")
	}
	maxNewTokenInput := req.MultipartForm.Value["max_new_tokens"]
	temperatureInput := req.MultipartForm.Value["temperature"]
	topKInput := req.MultipartForm.Value["top_k"]
	seedInput := req.MultipartForm.Value["seed"]
	extraParamsInput := req.MultipartForm.Value["extra_params"]
	chatHistoryInput := req.MultipartForm.Value["chat_history"]
	systemMessageInput := req.MultipartForm.Value["system_message"]

	maxNewTokens := utils.TextGenerationMaxNewTokens
	if len(maxNewTokenInput) > 0 {
		parseMaxNewToken, err := strconv.ParseInt(maxNewTokenInput[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		maxNewTokens = int32(parseMaxNewToken)
	}

	temperature := utils.TextGenerationTemperature
	if len(temperatureInput) > 0 {
		parseTemperature, err := strconv.ParseFloat(temperatureInput[0], 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		temperature = float32(parseTemperature)
	}

	topK := utils.TextGenerationTopK
	if len(topKInput) > 0 {
		parseTopK, err := strconv.ParseInt(topKInput[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		topK = int32(parseTopK)
	}

	seed := utils.TextGenerationSeed
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
	chatHistory := ""
	if len(chatHistoryInput) > 0 {
		chatHistory = chatHistoryInput[0]
	}
	systemMessage := ""
	if len(systemMessageInput) > 0 {
		systemMessage = systemMessageInput[0]
	}
	promptImages := ""
	parsedImages, err := parseImageFormDataInputsToBytes(req)
	if err == nil {
		jsonData, err := json.Marshal(parsedImages)
		if err != nil {
			log.Fatalf("Error marshaling to JSON: %v", err)
		} else {
			promptImages = string(jsonData)
		}
	}

	return &ray.TextGenerationChatInput{
		Prompt:        prompts[0],
		PromptImages:  promptImages,
		ChatHistory:   chatHistory,
		SystemMessage: systemMessage,
		MaxNewTokens:  maxNewTokens,
		Temperature:   temperature,
		TopK:          topK,
		Seed:          seed,
		ExtraParams:   extraParams,
	}, nil
}

func parseTextFormDataVisualQuestionAnsweringInputs(req *http.Request) (visualQuestionAnswering *ray.VisualQuestionAnsweringInput, err error) {
	prompts := req.MultipartForm.Value["prompt"]
	if len(prompts) != 1 {
		return nil, fmt.Errorf("only support batchsize 1")
	}
	maxNewTokenInput := req.MultipartForm.Value["max_new_tokens"]
	temperatureInput := req.MultipartForm.Value["temperature"]
	topKInput := req.MultipartForm.Value["top_k"]
	seedInput := req.MultipartForm.Value["seed"]
	extraParamsInput := req.MultipartForm.Value["extra_params"]
	chatHistoryInput := req.MultipartForm.Value["chat_history"]
	systemMessageInput := req.MultipartForm.Value["system_message"]

	maxNewTokens := utils.TextGenerationMaxNewTokens
	if len(maxNewTokenInput) > 0 {
		parseMaxNewToken, err := strconv.ParseInt(maxNewTokenInput[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		maxNewTokens = int32(parseMaxNewToken)
	}

	temperature := utils.TextGenerationTemperature
	if len(temperatureInput) > 0 {
		parseTemperature, err := strconv.ParseFloat(temperatureInput[0], 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		temperature = float32(parseTemperature)
	}

	topK := utils.TextGenerationTopK
	if len(topKInput) > 0 {
		parseTopK, err := strconv.ParseInt(topKInput[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input %w", err)
		}
		topK = int32(parseTopK)
	}

	seed := utils.TextGenerationSeed
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
	chatHistory := ""
	if len(chatHistoryInput) > 0 {
		chatHistory = chatHistoryInput[0]
	}
	systemMessage := ""
	if len(systemMessageInput) > 0 {
		systemMessage = systemMessageInput[0]
	}

	promptImages := ""
	parsedImages, err := parseImageFormDataInputsToBytes(req)
	if err == nil {
		jsonData, err := json.Marshal(parsedImages)
		if err != nil {
			log.Fatalf("Error marshaling to JSON: %v", err)
		} else {
			promptImages = string(jsonData)
		}
	}

	return &ray.VisualQuestionAnsweringInput{
		Prompt:        prompts[0],
		PromptImages:  promptImages,
		ChatHistory:   chatHistory,
		SystemMessage: systemMessage,
		MaxNewTokens:  maxNewTokens,
		Temperature:   temperature,
		TopK:          topK,
		Seed:          seed,
		ExtraParams:   extraParams,
	}, nil
}
