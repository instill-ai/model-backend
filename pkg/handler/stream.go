package handler

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"image"
// 	"image/jpeg"
// 	"io"
// 	"log"
// 	"strings"
// 	"time"

// 	"github.com/gofrs/uuid"
// 	"github.com/pkg/errors"
// 	"go.opentelemetry.io/otel/trace"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/status"

// 	"github.com/instill-ai/model-backend/pkg/constant"
// 	"github.com/instill-ai/model-backend/pkg/datamodel"
// 	"github.com/instill-ai/model-backend/pkg/ray"
// 	"github.com/instill-ai/model-backend/pkg/resource"
// 	"github.com/instill-ai/model-backend/pkg/utils"
// 	"github.com/instill-ai/x/sterr"

// 	logx "github.com/instill-ai/x/log"
// 	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
// 	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
// 	mgmtpb "github.com/instill-ai/protogen-go/mgmt/v1beta"
// 	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
// )

// func triggerUserModelBinaryFileUploadParser(stream modelpb.ModelPublicService_TriggerUserModelBinaryFileUploadServer) (triggerInput any, path string, version string, err error) {

// 	var firstChunk = true

// 	var fileData *modelpb.TriggerUserModelBinaryFileUploadRequest

// 	var allContentFiles []byte
// 	var fileLengths []uint32

// 	var textToImageInput *ray.TextToImageInput
// 	var textGeneration *ray.TextGenerationInput

// 	var task *modelpb.TaskInputStream
// 	for {
// 		fileData, err = stream.Recv()
// 		if errors.Is(err, io.EOF) {
// 			break
// 		} else if err != nil {
// 			err = errors.Wrapf(err,
// 				"failed while reading chunks from stream")
// 			return nil, "", "", err
// 		}

// 		if firstChunk { // first chunk contains model instance name
// 			firstChunk = false
// 			path, err = resource.GetRscNameID(fileData.Name) // format "users/{user}/models/{model}"
// 			if err != nil {
// 				return nil, "", "", err
// 			}
// 			version = fileData.Version
// 			task = fileData.TaskInput
// 			switch fileData.TaskInput.Input.(type) {
// 			case *modelpb.TaskInputStream_Classification:
// 				fileLengths = fileData.TaskInput.GetClassification().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
// 			case *modelpb.TaskInputStream_Detection:
// 				fileLengths = fileData.TaskInput.GetDetection().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
// 			case *modelpb.TaskInputStream_Keypoint:
// 				fileLengths = fileData.TaskInput.GetKeypoint().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
// 			case *modelpb.TaskInputStream_Ocr:
// 				fileLengths = fileData.TaskInput.GetOcr().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
// 			case *modelpb.TaskInputStream_InstanceSegmentation:
// 				fileLengths = fileData.TaskInput.GetInstanceSegmentation().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
// 			case *modelpb.TaskInputStream_SemanticSegmentation:
// 				fileLengths = fileData.TaskInput.GetSemanticSegmentation().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
// 			case *modelpb.TaskInputStream_TextToImage:
// 				extraParams := ""
// 				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
// 					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
// 					if err != nil {
// 						log.Fatalf("Error marshaling to JSON: %v", err)
// 					} else {
// 						extraParams = string(jsonData)
// 					}
// 				}
// 				textToImageInput = &ray.TextToImageInput{
// 					Prompt:      fileData.TaskInput.GetTextToImage().Prompt,
// 					PromptImage: "", // TODO: support streaming image generation
// 					Steps:       *fileData.TaskInput.GetTextToImage().Steps,
// 					CfgScale:    *fileData.TaskInput.GetTextToImage().CfgScale,
// 					Seed:        *fileData.TaskInput.GetTextToImage().Seed,
// 					Samples:     *fileData.TaskInput.GetTextToImage().Samples,
// 					ExtraParams: extraParams, // *fileData.TaskInput.GetTextToImage().ExtraParams
// 				}
// 			case *modelpb.TaskInputStream_TextGeneration:
// 				extraParams := ""
// 				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
// 					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
// 					if err != nil {
// 						log.Fatalf("Error marshaling to JSON: %v", err)
// 					} else {
// 						extraParams = string(jsonData)
// 					}
// 				}
// 				textGeneration = &ray.TextGenerationInput{
// 					Prompt: fileData.TaskInput.GetTextGeneration().Prompt,
// 					// PromptImage:  "", // TODO: support streaming image generation
// 					MaxNewTokens: *fileData.TaskInput.GetTextGeneration().MaxNewTokens,
// 					// StopWordsList: *fileData.TaskInput.GetTextGeneration().StopWordsList,
// 					Temperature: *fileData.TaskInput.GetTextGeneration().Temperature,
// 					TopK:        *fileData.TaskInput.GetTextGeneration().TopK,
// 					Seed:        *fileData.TaskInput.GetTextGeneration().Seed,
// 					ExtraParams: extraParams, // *fileData.TaskInput.GetTextGeneration().ExtraParams,
// 				}
// 			default:
// 				return nil, "", "", errors.New("unsupported task input type")
// 			}
// 		} else {
// 			switch fileData.TaskInput.Input.(type) {
// 			case *modelpb.TaskInputStream_Classification:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
// 			case *modelpb.TaskInputStream_Detection:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
// 			case *modelpb.TaskInputStream_Keypoint:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
// 			case *modelpb.TaskInputStream_Ocr:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
// 			case *modelpb.TaskInputStream_InstanceSegmentation:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
// 			case *modelpb.TaskInputStream_SemanticSegmentation:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
// 			default:
// 				return nil, "", "", errors.New("unsupported task input type")
// 			}
// 		}
// 	}

// 	switch task.Input.(type) {
// 	case *modelpb.TaskInputStream_Classification,
// 		*modelpb.TaskInputStream_Detection,
// 		*modelpb.TaskInputStream_Keypoint,
// 		*modelpb.TaskInputStream_Ocr,
// 		*modelpb.TaskInputStream_InstanceSegmentation,
// 		*modelpb.TaskInputStream_SemanticSegmentation:
// 		if len(fileLengths) == 0 {
// 			return nil, "", "", errors.New("wrong parameter length of files")
// 		}
// 		imageBytes := make([][]byte, len(fileLengths))
// 		start := uint32(0)
// 		for i := 0; i < len(fileLengths); i++ {
// 			buff := new(bytes.Buffer)
// 			img, _, err := image.Decode(bytes.NewReader(allContentFiles[start : start+fileLengths[i]]))
// 			if err != nil {
// 				return nil, "", "", err
// 			}
// 			err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
// 			if err != nil {
// 				return nil, "", "", err
// 			}
// 			imageBytes[i] = buff.Bytes()
// 			start += fileLengths[i]
// 		}
// 		return imageBytes, path, version, nil
// 	case *modelpb.TaskInputStream_TextToImage:
// 		return textToImageInput, path, version, nil
// 	case *modelpb.TaskInputStream_TextGeneration:
// 		return textGeneration, path, version, nil
// 	}
// 	return nil, "", "", errors.New("unsupported task input type")
// }

// func triggerOrganizationModelBinaryFileUploadParser(stream modelpb.ModelPublicService_TriggerOrganizationModelBinaryFileUploadServer) (triggerInput any, path string, version string, err error) {

// 	var firstChunk = true

// 	var fileData *modelpb.TriggerOrganizationModelBinaryFileUploadRequest

// 	var allContentFiles []byte
// 	var fileLengths []uint32

// 	var textToImageInput *ray.TextToImageInput
// 	var textGeneration *ray.TextGenerationInput

// 	var task *modelpb.TaskInputStream
// 	for {
// 		fileData, err = stream.Recv()
// 		if errors.Is(err, io.EOF) {
// 			break
// 		} else if err != nil {
// 			err = errors.Wrapf(err,
// 				"failed while reading chunks from stream")
// 			return nil, "", "", err
// 		}

// 		if firstChunk { // first chunk contains model instance name
// 			firstChunk = false
// 			path, err = resource.GetRscNameID(fileData.Name) // format "users/{user}/models/{model}"
// 			if err != nil {
// 				return nil, "", "", err
// 			}
// 			version = fileData.Version
// 			task = fileData.TaskInput
// 			switch fileData.TaskInput.Input.(type) {
// 			case *modelpb.TaskInputStream_Classification:
// 				fileLengths = fileData.TaskInput.GetClassification().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
// 			case *modelpb.TaskInputStream_Detection:
// 				fileLengths = fileData.TaskInput.GetDetection().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
// 			case *modelpb.TaskInputStream_Keypoint:
// 				fileLengths = fileData.TaskInput.GetKeypoint().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
// 			case *modelpb.TaskInputStream_Ocr:
// 				fileLengths = fileData.TaskInput.GetOcr().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
// 			case *modelpb.TaskInputStream_InstanceSegmentation:
// 				fileLengths = fileData.TaskInput.GetInstanceSegmentation().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
// 			case *modelpb.TaskInputStream_SemanticSegmentation:
// 				fileLengths = fileData.TaskInput.GetSemanticSegmentation().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
// 			case *modelpb.TaskInputStream_TextToImage:
// 				extraParams := ""
// 				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
// 					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
// 					if err != nil {
// 						log.Fatalf("Error marshaling to JSON: %v", err)
// 					} else {
// 						extraParams = string(jsonData)
// 					}
// 				}
// 				textToImageInput = &ray.TextToImageInput{
// 					Prompt:      fileData.TaskInput.GetTextToImage().Prompt,
// 					PromptImage: "", // TODO: support streaming image generation
// 					Steps:       *fileData.TaskInput.GetTextToImage().Steps,
// 					CfgScale:    *fileData.TaskInput.GetTextToImage().CfgScale,
// 					Seed:        *fileData.TaskInput.GetTextToImage().Seed,
// 					Samples:     *fileData.TaskInput.GetTextToImage().Samples,
// 					ExtraParams: extraParams, // *fileData.TaskInput.GetTextToImage().ExtraParams
// 				}
// 			case *modelpb.TaskInputStream_TextGeneration:
// 				extraParams := ""
// 				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
// 					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
// 					if err != nil {
// 						log.Fatalf("Error marshaling to JSON: %v", err)
// 					} else {
// 						extraParams = string(jsonData)
// 					}
// 				}
// 				textGeneration = &ray.TextGenerationInput{
// 					Prompt: fileData.TaskInput.GetTextGeneration().Prompt,
// 					// PromptImage:  "", // TODO: support streaming image generation
// 					MaxNewTokens: *fileData.TaskInput.GetTextGeneration().MaxNewTokens,
// 					// StopWordsList: *fileData.TaskInput.GetTextGeneration().StopWordsList,
// 					Temperature: *fileData.TaskInput.GetTextGeneration().Temperature,
// 					TopK:        *fileData.TaskInput.GetTextGeneration().TopK,
// 					Seed:        *fileData.TaskInput.GetTextGeneration().Seed,
// 					ExtraParams: extraParams, // *fileData.TaskInput.GetTextGeneration().ExtraParams,
// 				}
// 			default:
// 				return nil, "", "", errors.New("unsupported task input type")
// 			}
// 		} else {
// 			switch fileData.TaskInput.Input.(type) {
// 			case *modelpb.TaskInputStream_Classification:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
// 			case *modelpb.TaskInputStream_Detection:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
// 			case *modelpb.TaskInputStream_Keypoint:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
// 			case *modelpb.TaskInputStream_Ocr:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
// 			case *modelpb.TaskInputStream_InstanceSegmentation:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
// 			case *modelpb.TaskInputStream_SemanticSegmentation:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
// 			default:
// 				return nil, "", "", errors.New("unsupported task input type")
// 			}
// 		}
// 	}

// 	switch task.Input.(type) {
// 	case *modelpb.TaskInputStream_Classification,
// 		*modelpb.TaskInputStream_Detection,
// 		*modelpb.TaskInputStream_Keypoint,
// 		*modelpb.TaskInputStream_Ocr,
// 		*modelpb.TaskInputStream_InstanceSegmentation,
// 		*modelpb.TaskInputStream_SemanticSegmentation:
// 		if len(fileLengths) == 0 {
// 			return nil, "", "", errors.New("wrong parameter length of files")
// 		}
// 		imageBytes := make([][]byte, len(fileLengths))
// 		start := uint32(0)
// 		for i := 0; i < len(fileLengths); i++ {
// 			buff := new(bytes.Buffer)
// 			img, _, err := image.Decode(bytes.NewReader(allContentFiles[start : start+fileLengths[i]]))
// 			if err != nil {
// 				return nil, "", "", err
// 			}
// 			err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
// 			if err != nil {
// 				return nil, "", "", err
// 			}
// 			imageBytes[i] = buff.Bytes()
// 			start += fileLengths[i]
// 		}
// 		return imageBytes, path, version, nil
// 	case *modelpb.TaskInputStream_TextToImage:
// 		return textToImageInput, path, version, nil
// 	case *modelpb.TaskInputStream_TextGeneration:
// 		return textGeneration, path, version, nil
// 	}
// 	return nil, "", "", errors.New("unsupported task input type")
// }

// func triggerModelBinaryFileUploadParser(stream modelpb.ModelPublicService_TriggerModelVersionBinaryFileUploadServer) (triggerInput any, namespaceID string, modelID string, version string, err error) {

// 	var firstChunk = true

// 	var fileData *modelpb.TriggerModelVersionBinaryFileUploadRequest

// 	var allContentFiles []byte
// 	var fileLengths []uint32

// 	var textToImageInput *ray.TextToImageInput
// 	var textGeneration *ray.TextGenerationInput

// 	var task *modelpb.TaskInputStream
// 	for {
// 		fileData, err = stream.Recv()
// 		if errors.Is(err, io.EOF) {
// 			break
// 		} else if err != nil {
// 			err = errors.Wrapf(err,
// 				"failed while reading chunks from stream")
// 			return nil, "", "", "", err
// 		}

// 		if firstChunk { // first chunk contains model instance name
// 			firstChunk = false
// 			namespaceID = fileData.GetNamespaceId()
// 			modelID = fileData.GetModelId()
// 			version = fileData.Version
// 			task = fileData.TaskInput
// 			switch fileData.TaskInput.Input.(type) {
// 			case *modelpb.TaskInputStream_Classification:
// 				fileLengths = fileData.TaskInput.GetClassification().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
// 			case *modelpb.TaskInputStream_Detection:
// 				fileLengths = fileData.TaskInput.GetDetection().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
// 			case *modelpb.TaskInputStream_Keypoint:
// 				fileLengths = fileData.TaskInput.GetKeypoint().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
// 			case *modelpb.TaskInputStream_Ocr:
// 				fileLengths = fileData.TaskInput.GetOcr().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
// 			case *modelpb.TaskInputStream_InstanceSegmentation:
// 				fileLengths = fileData.TaskInput.GetInstanceSegmentation().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
// 			case *modelpb.TaskInputStream_SemanticSegmentation:
// 				fileLengths = fileData.TaskInput.GetSemanticSegmentation().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
// 			case *modelpb.TaskInputStream_TextToImage:
// 				extraParams := ""
// 				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
// 					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
// 					if err != nil {
// 						log.Fatalf("Error marshaling to JSON: %v", err)
// 					} else {
// 						extraParams = string(jsonData)
// 					}
// 				}
// 				textToImageInput = &ray.TextToImageInput{
// 					Prompt:      fileData.TaskInput.GetTextToImage().Prompt,
// 					PromptImage: "", // TODO: support streaming image generation
// 					Steps:       *fileData.TaskInput.GetTextToImage().Steps,
// 					CfgScale:    *fileData.TaskInput.GetTextToImage().CfgScale,
// 					Seed:        *fileData.TaskInput.GetTextToImage().Seed,
// 					Samples:     *fileData.TaskInput.GetTextToImage().Samples,
// 					ExtraParams: extraParams, // *fileData.TaskInput.GetTextToImage().ExtraParams
// 				}
// 			case *modelpb.TaskInputStream_TextGeneration:
// 				extraParams := ""
// 				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
// 					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
// 					if err != nil {
// 						log.Fatalf("Error marshaling to JSON: %v", err)
// 					} else {
// 						extraParams = string(jsonData)
// 					}
// 				}
// 				textGeneration = &ray.TextGenerationInput{
// 					Prompt: fileData.TaskInput.GetTextGeneration().Prompt,
// 					// PromptImage:  "", // TODO: support streaming image generation
// 					MaxNewTokens: *fileData.TaskInput.GetTextGeneration().MaxNewTokens,
// 					// StopWordsList: *fileData.TaskInput.GetTextGeneration().StopWordsList,
// 					Temperature: *fileData.TaskInput.GetTextGeneration().Temperature,
// 					TopK:        *fileData.TaskInput.GetTextGeneration().TopK,
// 					Seed:        *fileData.TaskInput.GetTextGeneration().Seed,
// 					ExtraParams: extraParams, // *fileData.TaskInput.GetTextGeneration().ExtraParams,
// 				}
// 			default:
// 				return nil, "", "", "", errors.New("unsupported task input type")
// 			}
// 		} else {
// 			switch fileData.TaskInput.Input.(type) {
// 			case *modelpb.TaskInputStream_Classification:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
// 			case *modelpb.TaskInputStream_Detection:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
// 			case *modelpb.TaskInputStream_Keypoint:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
// 			case *modelpb.TaskInputStream_Ocr:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
// 			case *modelpb.TaskInputStream_InstanceSegmentation:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
// 			case *modelpb.TaskInputStream_SemanticSegmentation:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
// 			default:
// 				return nil, "", "", "", errors.New("unsupported task input type")
// 			}
// 		}
// 	}

// 	switch task.Input.(type) {
// 	case *modelpb.TaskInputStream_Classification,
// 		*modelpb.TaskInputStream_Detection,
// 		*modelpb.TaskInputStream_Keypoint,
// 		*modelpb.TaskInputStream_Ocr,
// 		*modelpb.TaskInputStream_InstanceSegmentation,
// 		*modelpb.TaskInputStream_SemanticSegmentation:
// 		if len(fileLengths) == 0 {
// 			return nil, "", "", "", errors.New("wrong parameter length of files")
// 		}
// 		imageBytes := make([][]byte, len(fileLengths))
// 		start := uint32(0)
// 		for i := 0; i < len(fileLengths); i++ {
// 			buff := new(bytes.Buffer)
// 			img, _, err := image.Decode(bytes.NewReader(allContentFiles[start : start+fileLengths[i]]))
// 			if err != nil {
// 				return nil, "", "", "", err
// 			}
// 			err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
// 			if err != nil {
// 				return nil, "", "", "", err
// 			}
// 			imageBytes[i] = buff.Bytes()
// 			start += fileLengths[i]
// 		}
// 		return imageBytes, namespaceID, modelID, version, nil
// 	case *modelpb.TaskInputStream_TextToImage:
// 		return textToImageInput, namespaceID, modelID, version, nil
// 	case *modelpb.TaskInputStream_TextGeneration:
// 		return textGeneration, namespaceID, modelID, version, nil
// 	}
// 	return nil, "", "", "", errors.New("unsupported task input type")
// }

// func triggerNamespaceLatestModelBinaryFileUploadParser(stream modelpb.ModelPublicService_TriggerModelBinaryFileUploadServer) (triggerInput any, namespaceID string, modelID string, err error) {

// 	var firstChunk = true

// 	var fileData *modelpb.TriggerModelBinaryFileUploadRequest

// 	var allContentFiles []byte
// 	var fileLengths []uint32

// 	var textToImageInput *ray.TextToImageInput
// 	var textGeneration *ray.TextGenerationInput

// 	var task *modelpb.TaskInputStream
// 	for {
// 		fileData, err = stream.Recv()
// 		if errors.Is(err, io.EOF) {
// 			break
// 		} else if err != nil {
// 			err = errors.Wrapf(err,
// 				"failed while reading chunks from stream")
// 			return nil, "", "", err
// 		}

// 		if firstChunk { // first chunk contains model instance name
// 			firstChunk = false
// 			namespaceID = fileData.GetNamespaceId()
// 			modelID = fileData.GetModelId()
// 			task = fileData.TaskInput
// 			switch fileData.TaskInput.Input.(type) {
// 			case *modelpb.TaskInputStream_Classification:
// 				fileLengths = fileData.TaskInput.GetClassification().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
// 			case *modelpb.TaskInputStream_Detection:
// 				fileLengths = fileData.TaskInput.GetDetection().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
// 			case *modelpb.TaskInputStream_Keypoint:
// 				fileLengths = fileData.TaskInput.GetKeypoint().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
// 			case *modelpb.TaskInputStream_Ocr:
// 				fileLengths = fileData.TaskInput.GetOcr().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
// 			case *modelpb.TaskInputStream_InstanceSegmentation:
// 				fileLengths = fileData.TaskInput.GetInstanceSegmentation().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
// 			case *modelpb.TaskInputStream_SemanticSegmentation:
// 				fileLengths = fileData.TaskInput.GetSemanticSegmentation().FileLengths
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
// 			case *modelpb.TaskInputStream_TextToImage:
// 				extraParams := ""
// 				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
// 					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
// 					if err != nil {
// 						log.Fatalf("Error marshaling to JSON: %v", err)
// 					} else {
// 						extraParams = string(jsonData)
// 					}
// 				}
// 				textToImageInput = &ray.TextToImageInput{
// 					Prompt:      fileData.TaskInput.GetTextToImage().Prompt,
// 					PromptImage: "", // TODO: support streaming image generation
// 					Steps:       *fileData.TaskInput.GetTextToImage().Steps,
// 					CfgScale:    *fileData.TaskInput.GetTextToImage().CfgScale,
// 					Seed:        *fileData.TaskInput.GetTextToImage().Seed,
// 					Samples:     *fileData.TaskInput.GetTextToImage().Samples,
// 					ExtraParams: extraParams, // *fileData.TaskInput.GetTextToImage().ExtraParams
// 				}
// 			case *modelpb.TaskInputStream_TextGeneration:
// 				extraParams := ""
// 				if fileData.TaskInput.GetTextGeneration().ExtraParams != nil {
// 					jsonData, err := json.Marshal(fileData.TaskInput.GetTextGeneration().ExtraParams)
// 					if err != nil {
// 						log.Fatalf("Error marshaling to JSON: %v", err)
// 					} else {
// 						extraParams = string(jsonData)
// 					}
// 				}
// 				textGeneration = &ray.TextGenerationInput{
// 					Prompt: fileData.TaskInput.GetTextGeneration().Prompt,
// 					// PromptImage:  "", // TODO: support streaming image generation
// 					MaxNewTokens: *fileData.TaskInput.GetTextGeneration().MaxNewTokens,
// 					// StopWordsList: *fileData.TaskInput.GetTextGeneration().StopWordsList,
// 					Temperature: *fileData.TaskInput.GetTextGeneration().Temperature,
// 					TopK:        *fileData.TaskInput.GetTextGeneration().TopK,
// 					Seed:        *fileData.TaskInput.GetTextGeneration().Seed,
// 					ExtraParams: extraParams, // *fileData.TaskInput.GetTextGeneration().ExtraParams,
// 				}
// 			default:
// 				return nil, "", "", errors.New("unsupported task input type")
// 			}
// 		} else {
// 			switch fileData.TaskInput.Input.(type) {
// 			case *modelpb.TaskInputStream_Classification:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetClassification().Content...)
// 			case *modelpb.TaskInputStream_Detection:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetDetection().Content...)
// 			case *modelpb.TaskInputStream_Keypoint:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetKeypoint().Content...)
// 			case *modelpb.TaskInputStream_Ocr:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetOcr().Content...)
// 			case *modelpb.TaskInputStream_InstanceSegmentation:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetInstanceSegmentation().Content...)
// 			case *modelpb.TaskInputStream_SemanticSegmentation:
// 				allContentFiles = append(allContentFiles, fileData.TaskInput.GetSemanticSegmentation().Content...)
// 			default:
// 				return nil, "", "", errors.New("unsupported task input type")
// 			}
// 		}
// 	}

// 	switch task.Input.(type) {
// 	case *modelpb.TaskInputStream_Classification,
// 		*modelpb.TaskInputStream_Detection,
// 		*modelpb.TaskInputStream_Keypoint,
// 		*modelpb.TaskInputStream_Ocr,
// 		*modelpb.TaskInputStream_InstanceSegmentation,
// 		*modelpb.TaskInputStream_SemanticSegmentation:
// 		if len(fileLengths) == 0 {
// 			return nil, "", "", errors.New("wrong parameter length of files")
// 		}
// 		imageBytes := make([][]byte, len(fileLengths))
// 		start := uint32(0)
// 		for i := 0; i < len(fileLengths); i++ {
// 			buff := new(bytes.Buffer)
// 			img, _, err := image.Decode(bytes.NewReader(allContentFiles[start : start+fileLengths[i]]))
// 			if err != nil {
// 				return nil, "", "", err
// 			}
// 			err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
// 			if err != nil {
// 				return nil, "", "", err
// 			}
// 			imageBytes[i] = buff.Bytes()
// 			start += fileLengths[i]
// 		}
// 		return imageBytes, namespaceID, modelID, nil
// 	case *modelpb.TaskInputStream_TextToImage:
// 		return textToImageInput, namespaceID, modelID, nil
// 	case *modelpb.TaskInputStream_TextGeneration:
// 		return textGeneration, namespaceID, modelID, nil
// 	}
// 	return nil, "", "", errors.New("unsupported task input type")
// }

// func (h *PublicHandler) TriggerUserModelBinaryFileUpload(stream modelpb.ModelPublicService_TriggerUserModelBinaryFileUploadServer) error {
// 	triggerInput, path, versionID, err := triggerUserModelBinaryFileUploadParser(stream)
// 	if err != nil {
// 		return status.Error(codes.Internal, err.Error())
// 	}

// 	namespaceID := strings.Split(path, "/")[1]
// 	modelID := strings.Split(path, "/")[3]

// 	response, task, err := h.triggerModelBinaryFileUpload(stream.Context(), triggerInput, namespaceID, modelID, versionID)
// 	if err != nil {
// 		return err
// 	}

// 	err = stream.SendAndClose(&modelpb.TriggerUserModelBinaryFileUploadResponse{
// 		Task:        *task,
// 		TaskOutputs: response,
// 	})

// 	return err
// }

// func (h *PublicHandler) TriggerOrganizationModelBinaryFileUpload(stream modelpb.ModelPublicService_TriggerOrganizationModelBinaryFileUploadServer) error {
// 	triggerInput, path, versionID, err := triggerOrganizationModelBinaryFileUploadParser(stream)
// 	if err != nil {
// 		return status.Error(codes.Internal, err.Error())
// 	}

// 	namespaceID := strings.Split(path, "/")[1]
// 	modelID := strings.Split(path, "/")[3]

// 	response, task, err := h.triggerModelBinaryFileUpload(stream.Context(), triggerInput, namespaceID, modelID, versionID)
// 	if err != nil {
// 		return err
// 	}

// 	err = stream.SendAndClose(&modelpb.TriggerOrganizationModelBinaryFileUploadResponse{
// 		Task:        *task,
// 		TaskOutputs: response,
// 	})

// 	return err
// }

// func (h *PublicHandler) TriggerModelVersionBinaryFileUpload(stream modelpb.ModelPublicService_TriggerModelVersionBinaryFileUploadServer) error {
// 	triggerInput, namespaceID, modelID, versionID, err := triggerModelBinaryFileUploadParser(stream)
// 	if err != nil {
// 		return status.Error(codes.Internal, err.Error())
// 	}

// 	response, task, err := h.triggerModelBinaryFileUpload(stream.Context(), triggerInput, namespaceID, modelID, versionID)
// 	if err != nil {
// 		return err
// 	}

// 	err = stream.SendAndClose(&modelpb.TriggerModelVersionBinaryFileUploadResponse{
// 		Task:        *task,
// 		TaskOutputs: response,
// 	})

// 	return err
// }

// func (h *PublicHandler) TriggerModelBinaryFileUpload(stream modelpb.ModelPublicService_TriggerModelBinaryFileUploadServer) error {
// 	triggerInput, namespaceID, modelID, err := triggerNamespaceLatestModelBinaryFileUploadParser(stream)
// 	if err != nil {
// 		return status.Error(codes.Internal, err.Error())
// 	}

// 	response, task, err := h.triggerModelBinaryFileUpload(stream.Context(), triggerInput, namespaceID, modelID, "")
// 	if err != nil {
// 		return err
// 	}

// 	err = stream.SendAndClose(&modelpb.TriggerModelBinaryFileUploadResponse{
// 		Task:        *task,
// 		TaskOutputs: response,
// 	})

// 	return err
// }

// func (h *PublicHandler) triggerModelBinaryFileUpload(ctx context.Context, triggerInput any, namespaceID string, modelID string, versionID string) ([]*modelpb.TaskOutput, *commonpb.Task, error) {

// 	startTime := time.Now()
// 	eventName := "TriggerUserModelBinaryFileUpload"

// 	ctx, span := tracer.Start(ctx, eventName,
// 		trace.WithSpanKind(trace.SpanKindServer))
// 	defer span.End()

// 	logUUID, _ := uuid.NewV4()

// 	logger, _ := custom_logger.GetZapLogger(ctx)

// 	ns, err := h.service.GetRscNamespace(ctx, namespaceID)
// 	if err != nil {
// 		span.SetStatus(1, err.Error())
// 		return nil, nil, err
// 	}
// 	if err := authenticateUser(ctx, false); err != nil {
// 		span.SetStatus(1, err.Error())
// 		return nil, nil, err
// 	}

// 	pbModel, err := h.service.GetModelByID(ctx, ns, modelID, modelpb.View_VIEW_FULL)
// 	if err != nil {
// 		span.SetStatus(1, err.Error())
// 		return nil, nil, err
// 	}

// 	modelDefID, err := resource.GetDefinitionID(pbModel.ModelDefinition)
// 	if err != nil {
// 		span.SetStatus(1, err.Error())
// 		return nil, nil, err
// 	}

// 	modelDef, err := h.service.GetRepository().GetModelDefinition(modelDefID)
// 	if err != nil {
// 		span.SetStatus(1, err.Error())
// 		return nil, nil, status.Error(codes.InvalidArgument, err.Error())
// 	}

// 	var version *datamodel.ModelVersion
// 	modelUID := uuid.FromStringOrNil(pbModel.Uid)
// 	if versionID == "" {
// 		version, err = h.service.GetRepository().GetLatestModelVersionByModelUID(ctx, modelUID)
// 		if err != nil {
// 			return nil, nil, status.Error(codes.NotFound, err.Error())
// 		}
// 	} else {
// 		version, err = h.service.GetModelVersionAdmin(ctx, modelUID, versionID)
// 		if err != nil {
// 			return nil, nil, status.Error(codes.NotFound, err.Error())
// 		}
// 	}

// 	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

// 	usageData := &utils.UsageMetricData{
// 		OwnerUID:           ns.NsUID.String(),
// 		OwnerType:          mgmtpb.OwnerType_OWNER_TYPE_USER,
// 		UserUID:            userUID,
// 		UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
// 		ModelUID:           pbModel.Uid,
// 		Mode:               mgmtpb.Mode_MODE_SYNC,
// 		TriggerUID:         logUUID.String(),
// 		TriggerTime:        startTime.Format(time.RFC3339Nano),
// 		ModelDefinitionUID: modelDef.UID.String(),
// 		ModelTask:          pbModel.Task,
// 	}

// 	// write usage/metric datapoint and prediction record
// 	defer func(u *utils.UsageMetricData, startTime time.Time) {
// 		// TODO: prediction feature not ready
// 		// pred.ComputeTimeDuration = time.Since(startTime).Seconds()
// 		// if err := h.service.CreateModelPrediction(ctx, pred); err != nil {
// 		// 	logger.Warn("model prediction write failed")
// 		// }
// 		u.ComputeTimeDuration = time.Since(startTime).Seconds()
// 		if err := h.service.WriteNewDataPoint(ctx, usageData); err != nil {
// 			logger.Warn("usage/metric write failed")
// 		}
// 	}(usageData, startTime)

// 	// check whether model support batching or not. If not, raise an error
// 	numberOfInferences := 1
// 	switch pbModel.Task {
// 	case commonpb.Task_TASK_CLASSIFICATION,
// 		commonpb.Task_TASK_DETECTION,
// 		commonpb.Task_TASK_INSTANCE_SEGMENTATION,
// 		commonpb.Task_TASK_SEMANTIC_SEGMENTATION,
// 		commonpb.Task_TASK_OCR,
// 		commonpb.Task_TASK_KEYPOINT:
// 		numberOfInferences = len(triggerInput.([][]byte))
// 	}
// 	if numberOfInferences > 1 {
// 		doSupportBatch, err := utils.DoSupportBatch()
// 		if err != nil {
// 			span.SetStatus(1, err.Error())
// 			usageData.Status = mgmtpb.Status_STATUS_ERRORED
// 			return nil, nil, status.Error(codes.InvalidArgument, err.Error())
// 		}
// 		if !doSupportBatch {
// 			span.SetStatus(1, "The model do not support batching, so could not make inference with multiple images")
// 			usageData.Status = mgmtpb.Status_STATUS_ERRORED
// 			return nil, nil, status.Error(codes.InvalidArgument, "The model do not support batching, so could not make inference with multiple images")
// 		}
// 	}

// 	parsedInputJSON, err := json.Marshal(triggerInput)
// 	if err != nil {
// 		span.SetStatus(1, err.Error())
// 		usageData.Status = mgmtpb.Status_STATUS_ERRORED
// 		return nil, nil, status.Error(codes.InvalidArgument, err.Error())
// 	}

// 	response, err := h.service.TriggerModelVersionByID(ctx, ns, modelID, version, parsedInputJSON, pbModel.Task, logUUID.String())
// 	if err != nil {
// 		st, e := sterr.CreateErrorResourceInfo(
// 			codes.FailedPrecondition,
// 			fmt.Sprintf("[handler] inference model error: %s", err.Error()),
// 			"Ray inference server",
// 			"",
// 			"",
// 			err.Error(),
// 		)
// 		if strings.Contains(err.Error(), "Failed to allocate memory") {
// 			st, e = sterr.CreateErrorResourceInfo(
// 				codes.ResourceExhausted,
// 				"[handler] inference model error",
// 				"Ray inference server OOM",
// 				"Out of memory for running the model, maybe try with smaller batch size",
// 				"",
// 				err.Error(),
// 			)
// 		}

// 		if e != nil {
// 			logger.Error(e.Error())
// 		}
// 		span.SetStatus(1, st.Err().Error())
// 		usageData.Status = mgmtpb.Status_STATUS_ERRORED
// 		return nil, nil, st.Err()
// 	}

// 	usageData.Status = mgmtpb.Status_STATUS_COMPLETED

// 	logger.Info(string(custom_otel.NewLogMessage(
// 		ctx,
// 		span,
// 		logUUID.String(),
// 		eventName,
// 		custom_otel.SetEventResource(pbModel.Name),
// 		custom_otel.SetEventMessage(fmt.Sprintf("%s done", eventName)),
// 	)))

// 	return response, &pbModel.Task, err
// }
