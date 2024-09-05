package handler

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"image"
// 	"image/jpeg"
// 	_ "image/png"
// 	"log"
// 	"net/http"
// 	"strconv"

// 	_ "golang.org/x/image/tiff"

// 	"github.com/instill-ai/model-backend/config"
// 	"github.com/instill-ai/model-backend/pkg/constant"
// 	"github.com/instill-ai/model-backend/pkg/ray"
// 	"github.com/instill-ai/model-backend/pkg/utils"

// 	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
// )

// func parseImageFormDataInputsToBytes(req *http.Request) (imgsBytes [][]byte, err error) {

// 	logger, _ := custom_logger.GetZapLogger(req.Context())

// 	inputs := req.MultipartForm.File["file"]
// 	for _, content := range inputs {
// 		file, err := content.Open()

// 		if err != nil {
// 			if err := file.Close(); err != nil {
// 				return nil, err
// 			}
// 			logger.Error(fmt.Sprintf("Unable to open file for image %v", err))
// 			return nil, fmt.Errorf("unable to open file for image")
// 		}

// 		buff := new(bytes.Buffer) // pointer
// 		numBytes, err := buff.ReadFrom(file)
// 		if err != nil {
// 			if err := file.Close(); err != nil {
// 				return nil, err
// 			}
// 			logger.Error(fmt.Sprintf("Unable to read content body from image %v", err))
// 			return nil, fmt.Errorf("unable to read content body from image")
// 		}
// 		if err := file.Close(); err != nil {
// 			return nil, err
// 		}

// 		if numBytes > int64(config.Config.Server.MaxDataSize*constant.MB) {
// 			return nil, fmt.Errorf(
// 				"image size must be smaller than %vMB. Got %vMB from image %v",
// 				config.Config.Server.MaxDataSize,
// 				float32(numBytes)/float32(constant.MB),
// 				content.Filename,
// 			)
// 		}

// 		img, _, err := image.Decode(buff)
// 		if err != nil {
// 			logger.Error(fmt.Sprintf("Unable to decode image: %v", err))
// 			return nil, fmt.Errorf("unable to decode image")
// 		}

// 		// Encode into jpeg to remove alpha channel (hack)
// 		// This may slightly degrade the image quality
// 		buff = new(bytes.Buffer)
// 		err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
// 		if err != nil {
// 			logger.Error(fmt.Sprintf("Unable to process image: %v", err))
// 			return nil, fmt.Errorf("unable to process image")
// 		}

// 		imgsBytes = append(imgsBytes, buff.Bytes())
// 	}

// 	return imgsBytes, nil
// }

// func parseImageFormDataTextToImageInputs(req *http.Request) (textToImageInput *ray.TextToImageInput, err error) {
// 	prompts := req.MultipartForm.Value["prompt"]
// 	if len(prompts) == 0 {
// 		return nil, fmt.Errorf("missing prompt input")
// 	}
// 	if len(prompts) > 1 {
// 		return nil, fmt.Errorf("invalid prompt input, only support a single prompt")
// 	}
// 	stepStr := req.MultipartForm.Value["steps"]
// 	cfgScaleStr := req.MultipartForm.Value["cfg_scale"]
// 	seedStr := req.MultipartForm.Value["seed"]
// 	samplesStr := req.MultipartForm.Value["samples"]
// 	extraParamsInput := req.MultipartForm.Value["extra_params"]

// 	if len(stepStr) > 1 {
// 		return nil, fmt.Errorf("invalid steps input, only support a single steps")
// 	}
// 	if len(cfgScaleStr) > 1 {
// 		return nil, fmt.Errorf("invalid cfg_scale input, only support a single cfg_scale")
// 	}
// 	if len(seedStr) > 1 {
// 		return nil, fmt.Errorf("invalid seed input, only support a single seed")
// 	}
// 	if len(samplesStr) > 1 {
// 		return nil, fmt.Errorf("invalid samples input, only support a single samples")
// 	}

// 	step := utils.ToImageSteps
// 	if len(stepStr) > 0 {
// 		parseStep, err := strconv.ParseInt(stepStr[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid step input %w", err)
// 		}
// 		step = int32(parseStep)
// 	}

// 	cfgScale := float64(utils.ToImageCFGScale)
// 	if len(cfgScaleStr) > 0 {
// 		cfgScale, err = strconv.ParseFloat(cfgScaleStr[0], 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid cfgScale input %w", err)
// 		}
// 	}

// 	seed := utils.ToImageSeed
// 	if len(seedStr) > 0 {
// 		parseSeed, err := strconv.ParseInt(seedStr[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid seed input %w", err)
// 		}
// 		seed = int32(parseSeed)
// 	}

// 	samples := utils.ToImageSamples
// 	if len(samplesStr) > 0 {
// 		parseSamples, err := strconv.ParseInt(samplesStr[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid samples input %w", err)
// 		}
// 		samples = int32(parseSamples)
// 	}

// 	if samples > 1 {
// 		return nil, fmt.Errorf("we only allow samples=1 for now and will improve to allow the generation of multiple samples in the future")
// 	}

// 	extraParams := ""
// 	if len(extraParamsInput) > 0 {
// 		extraParams = extraParamsInput[0]
// 	}

// 	parsedImages, err := parseImageFormDataInputsToBytes(req)
// 	var promptImage string
// 	if err != nil && len(parsedImages) == 1 {
// 		promptImage = string(parsedImages[0])
// 	}

// 	return &ray.TextToImageInput{
// 		Prompt:      prompts[0],
// 		PromptImage: promptImage,
// 		Steps:       step,
// 		CfgScale:    float32(cfgScale),
// 		Seed:        seed,
// 		Samples:     samples,
// 		ExtraParams: extraParams,
// 	}, nil
// }

// func parseImageFormDataImageToImageInputs(req *http.Request) (imageToImageInput *ray.ImageToImageInput, err error) {
// 	prompts := req.MultipartForm.Value["prompt"]
// 	if len(prompts) == 0 {
// 		return nil, fmt.Errorf("missing prompt input")
// 	}
// 	if len(prompts) > 1 {
// 		return nil, fmt.Errorf("invalid prompt input, only support a single prompt")
// 	}
// 	stepStr := req.MultipartForm.Value["steps"]
// 	cfgScaleStr := req.MultipartForm.Value["cfg_scale"]
// 	seedStr := req.MultipartForm.Value["seed"]
// 	samplesStr := req.MultipartForm.Value["samples"]
// 	extraParamsInput := req.MultipartForm.Value["extra_params"]

// 	if len(stepStr) > 1 {
// 		return nil, fmt.Errorf("invalid steps input, only support a single steps")
// 	}
// 	if len(cfgScaleStr) > 1 {
// 		return nil, fmt.Errorf("invalid cfg_scale input, only support a single cfg_scale")
// 	}
// 	if len(seedStr) > 1 {
// 		return nil, fmt.Errorf("invalid seed input, only support a single seed")
// 	}
// 	if len(samplesStr) > 1 {
// 		return nil, fmt.Errorf("invalid samples input, only support a single samples")
// 	}

// 	step := utils.ToImageSteps
// 	if len(stepStr) > 0 {
// 		parseStep, err := strconv.ParseInt(stepStr[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid step input %w", err)
// 		}
// 		step = int32(parseStep)
// 	}

// 	cfgScale := float64(utils.ToImageCFGScale)
// 	if len(cfgScaleStr) > 0 {
// 		cfgScale, err = strconv.ParseFloat(cfgScaleStr[0], 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid cfgScale input %w", err)
// 		}
// 	}

// 	seed := utils.ToImageSeed
// 	if len(seedStr) > 0 {
// 		parseSeed, err := strconv.ParseInt(seedStr[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid seed input %w", err)
// 		}
// 		seed = int32(parseSeed)
// 	}

// 	samples := utils.ToImageSamples
// 	if len(samplesStr) > 0 {
// 		parseSamples, err := strconv.ParseInt(samplesStr[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid samples input %w", err)
// 		}
// 		samples = int32(parseSamples)
// 	}

// 	if samples > 1 {
// 		return nil, fmt.Errorf("we only allow samples=1 for now and will improve to allow the generation of multiple samples in the future")
// 	}

// 	extraParams := ""
// 	if len(extraParamsInput) > 0 {
// 		extraParams = extraParamsInput[0]
// 	}

// 	parsedImages, err := parseImageFormDataInputsToBytes(req)
// 	var promptImage string
// 	if err != nil && len(parsedImages) == 1 {
// 		promptImage = string(parsedImages[0])
// 	}

// 	return &ray.ImageToImageInput{
// 		Prompt:      prompts[0],
// 		PromptImage: promptImage,
// 		Steps:       step,
// 		CfgScale:    float32(cfgScale),
// 		Seed:        seed,
// 		Samples:     samples,
// 		ExtraParams: extraParams,
// 	}, nil
// }

// func parseTextFormDataTextGenerationInputs(req *http.Request) (textGeneration *ray.TextGenerationInput, err error) {
// 	prompts := req.MultipartForm.Value["prompt"]
// 	if len(prompts) != 1 {
// 		return nil, fmt.Errorf("only support batchsize 1")
// 	}
// 	maxNewTokenInput := req.MultipartForm.Value["max_new_tokens"]
// 	temperatureInput := req.MultipartForm.Value["temperature"]
// 	topKInput := req.MultipartForm.Value["top_k"]
// 	seedInput := req.MultipartForm.Value["seed"]
// 	extraParamsInput := req.MultipartForm.Value["extra_params"]
// 	chatHistoryInput := req.MultipartForm.Value["chat_history"]
// 	systemMessageInput := req.MultipartForm.Value["system_message"]

// 	maxNewTokens := utils.TextGenerationMaxNewTokens
// 	if len(maxNewTokenInput) > 0 {
// 		parseMaxNewToken, err := strconv.ParseInt(maxNewTokenInput[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		maxNewTokens = int32(parseMaxNewToken)
// 	}

// 	temperature := utils.TextGenerationTemperature
// 	if len(temperatureInput) > 0 {
// 		parseTemperature, err := strconv.ParseFloat(temperatureInput[0], 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		temperature = float32(parseTemperature)
// 	}

// 	topK := utils.TextGenerationTopK
// 	if len(topKInput) > 0 {
// 		parseTopK, err := strconv.ParseInt(topKInput[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		topK = int32(parseTopK)
// 	}

// 	seed := utils.TextGenerationSeed
// 	if len(seedInput) > 0 {
// 		parseSeed, err := strconv.ParseInt(seedInput[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		seed = int32(parseSeed)
// 	}

// 	extraParams := ""
// 	if len(extraParamsInput) > 0 {
// 		extraParams = extraParamsInput[0]
// 	}
// 	chatHistory := ""
// 	if len(chatHistoryInput) > 0 {
// 		chatHistory = chatHistoryInput[0]
// 	}
// 	systemMessage := ""
// 	if len(systemMessageInput) > 0 {
// 		systemMessage = systemMessageInput[0]
// 	}

// 	promptImages := ""
// 	parsedImages, err := parseImageFormDataInputsToBytes(req)
// 	if err == nil {
// 		jsonData, err := json.Marshal(parsedImages)
// 		if err != nil {
// 			log.Fatalf("Error marshaling to JSON: %v", err)
// 		} else {
// 			promptImages = string(jsonData)
// 		}
// 	}

// 	return &ray.TextGenerationInput{
// 		Prompt:        prompts[0],
// 		PromptImages:  promptImages,
// 		ChatHistory:   chatHistory,
// 		SystemMessage: systemMessage,
// 		MaxNewTokens:  maxNewTokens,
// 		Temperature:   temperature,
// 		TopK:          topK,
// 		Seed:          seed,
// 		ExtraParams:   extraParams,
// 	}, nil
// }

// func parseTextFormDataTextGenerationChatInputs(req *http.Request) (textGenerationChat *ray.TextGenerationChatInput, err error) {
// 	prompts := req.MultipartForm.Value["prompt"]
// 	if len(prompts) != 1 {
// 		return nil, fmt.Errorf("only support batchsize 1")
// 	}
// 	maxNewTokenInput := req.MultipartForm.Value["max_new_tokens"]
// 	temperatureInput := req.MultipartForm.Value["temperature"]
// 	topKInput := req.MultipartForm.Value["top_k"]
// 	seedInput := req.MultipartForm.Value["seed"]
// 	extraParamsInput := req.MultipartForm.Value["extra_params"]
// 	chatHistoryInput := req.MultipartForm.Value["chat_history"]
// 	systemMessageInput := req.MultipartForm.Value["system_message"]

// 	maxNewTokens := utils.TextGenerationMaxNewTokens
// 	if len(maxNewTokenInput) > 0 {
// 		parseMaxNewToken, err := strconv.ParseInt(maxNewTokenInput[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		maxNewTokens = int32(parseMaxNewToken)
// 	}

// 	temperature := utils.TextGenerationTemperature
// 	if len(temperatureInput) > 0 {
// 		parseTemperature, err := strconv.ParseFloat(temperatureInput[0], 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		temperature = float32(parseTemperature)
// 	}

// 	topK := utils.TextGenerationTopK
// 	if len(topKInput) > 0 {
// 		parseTopK, err := strconv.ParseInt(topKInput[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		topK = int32(parseTopK)
// 	}

// 	seed := utils.TextGenerationSeed
// 	if len(seedInput) > 0 {
// 		parseSeed, err := strconv.ParseInt(seedInput[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		seed = int32(parseSeed)
// 	}

// 	extraParams := ""
// 	if len(extraParamsInput) > 0 {
// 		extraParams = extraParamsInput[0]
// 	}
// 	chatHistory := ""
// 	if len(chatHistoryInput) > 0 {
// 		chatHistory = chatHistoryInput[0]
// 	}
// 	systemMessage := ""
// 	if len(systemMessageInput) > 0 {
// 		systemMessage = systemMessageInput[0]
// 	}
// 	promptImages := ""
// 	parsedImages, err := parseImageFormDataInputsToBytes(req)
// 	if err == nil {
// 		jsonData, err := json.Marshal(parsedImages)
// 		if err != nil {
// 			log.Fatalf("Error marshaling to JSON: %v", err)
// 		} else {
// 			promptImages = string(jsonData)
// 		}
// 	}

// 	return &ray.TextGenerationChatInput{
// 		Prompt:        prompts[0],
// 		PromptImages:  promptImages,
// 		ChatHistory:   chatHistory,
// 		SystemMessage: systemMessage,
// 		MaxNewTokens:  maxNewTokens,
// 		Temperature:   temperature,
// 		TopK:          topK,
// 		Seed:          seed,
// 		ExtraParams:   extraParams,
// 	}, nil
// }

// func parseTextFormDataVisualQuestionAnsweringInputs(req *http.Request) (visualQuestionAnswering *ray.VisualQuestionAnsweringInput, err error) {
// 	prompts := req.MultipartForm.Value["prompt"]
// 	if len(prompts) != 1 {
// 		return nil, fmt.Errorf("only support batchsize 1")
// 	}
// 	maxNewTokenInput := req.MultipartForm.Value["max_new_tokens"]
// 	temperatureInput := req.MultipartForm.Value["temperature"]
// 	topKInput := req.MultipartForm.Value["top_k"]
// 	seedInput := req.MultipartForm.Value["seed"]
// 	extraParamsInput := req.MultipartForm.Value["extra_params"]
// 	chatHistoryInput := req.MultipartForm.Value["chat_history"]
// 	systemMessageInput := req.MultipartForm.Value["system_message"]

// 	maxNewTokens := utils.TextGenerationMaxNewTokens
// 	if len(maxNewTokenInput) > 0 {
// 		parseMaxNewToken, err := strconv.ParseInt(maxNewTokenInput[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		maxNewTokens = int32(parseMaxNewToken)
// 	}

// 	temperature := utils.TextGenerationTemperature
// 	if len(temperatureInput) > 0 {
// 		parseTemperature, err := strconv.ParseFloat(temperatureInput[0], 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		temperature = float32(parseTemperature)
// 	}

// 	topK := utils.TextGenerationTopK
// 	if len(topKInput) > 0 {
// 		parseTopK, err := strconv.ParseInt(topKInput[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		topK = int32(parseTopK)
// 	}

// 	seed := utils.TextGenerationSeed
// 	if len(seedInput) > 0 {
// 		parseSeed, err := strconv.ParseInt(seedInput[0], 10, 32)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid input %w", err)
// 		}
// 		seed = int32(parseSeed)
// 	}

// 	extraParams := ""
// 	if len(extraParamsInput) > 0 {
// 		extraParams = extraParamsInput[0]
// 	}
// 	chatHistory := ""
// 	if len(chatHistoryInput) > 0 {
// 		chatHistory = chatHistoryInput[0]
// 	}
// 	systemMessage := ""
// 	if len(systemMessageInput) > 0 {
// 		systemMessage = systemMessageInput[0]
// 	}

// 	promptImages := ""
// 	parsedImages, err := parseImageFormDataInputsToBytes(req)
// 	if err == nil {
// 		jsonData, err := json.Marshal(parsedImages)
// 		if err != nil {
// 			log.Fatalf("Error marshaling to JSON: %v", err)
// 		} else {
// 			promptImages = string(jsonData)
// 		}
// 	}

// 	return &ray.VisualQuestionAnsweringInput{
// 		Prompt:        prompts[0],
// 		PromptImages:  promptImages,
// 		ChatHistory:   chatHistory,
// 		SystemMessage: systemMessage,
// 		MaxNewTokens:  maxNewTokens,
// 		Temperature:   temperature,
// 		TopK:          topK,
// 		Seed:          seed,
// 		ExtraParams:   extraParams,
// 	}, nil
// }
