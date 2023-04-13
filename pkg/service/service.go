package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"go.temporal.io/sdk/client"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/triton"
	"github.com/instill-ai/model-backend/pkg/util"
	"github.com/instill-ai/model-backend/pkg/worker"
	"github.com/instill-ai/x/sterr"

	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type InferInput interface{}

type Service interface {
	CreateModelAsync(owner string, model *datamodel.Model) (string, error)
	GetModelById(owner string, modelID string, view modelPB.View) (datamodel.Model, error)
	GetModelByUid(owner string, modelUID uuid.UUID, view modelPB.View) (datamodel.Model, error)
	DeleteModel(owner string, modelID string) error
	RenameModel(owner string, modelID string, newModelId string) (datamodel.Model, error)
	PublishModel(owner string, modelID string) (datamodel.Model, error)
	UnpublishModel(owner string, modelID string) (datamodel.Model, error)
	UpdateModel(modelUID uuid.UUID, model *datamodel.Model) (datamodel.Model, error)
	UpdateModelState(modelUID uuid.UUID, model *datamodel.Model, state datamodel.ModelState) (datamodel.Model, error)
	ListModels(owner string, view modelPB.View, pageSize int, pageToken string) ([]datamodel.Model, string, int64, error)
	WatchModel(name string) (*controllerPB.GetResourceResponse, error)
	CheckModel(modelUID uuid.UUID) (*modelPB.Model_State, error)

	ModelInfer(modelUID uuid.UUID, inferInput InferInput, task modelPB.Model_Task) ([]*modelPB.TaskOutput, error)
	ModelInferTestMode(owner string, modelUID uuid.UUID, inferInput InferInput, task modelPB.Model_Task) ([]*modelPB.TaskOutput, error)

	DeployModelAsync(owner string, modelUID uuid.UUID) (string, error)
	UndeployModelAsync(owner string, modelUID uuid.UUID) (string, error)

	GetModelDefinition(id string) (datamodel.ModelDefinition, error)
	GetModelDefinitionByUid(uid uuid.UUID) (datamodel.ModelDefinition, error)
	ListModelDefinitions(view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelDefinition, string, int64, error)

	GetTritonEnsembleModel(modelUID uuid.UUID) (datamodel.TritonModel, error)
	GetTritonModels(modelUID uuid.UUID) ([]datamodel.TritonModel, error)

	GetOperation(workflowId string) (*longrunningpb.Operation, *worker.ModelParams, string, error)
	ListOperation(pageSize int, pageToken string) ([]*longrunningpb.Operation, []*worker.ModelParams, string, int64, error)
	CancelOperation(workflowId string) error

	SearchAttributeReady() error
	GetModelByIdAdmin(modelID string, view modelPB.View) (datamodel.Model, error)
	GetModelByUidAdmin(modelUID uuid.UUID, view modelPB.View) (datamodel.Model, error)
	ListModelsAdmin(view modelPB.View, pageSize int, pageToken string) ([]datamodel.Model, string, int64, error)

	GetResourceState(modelID string) (*modelPB.Model_State, error)
	UpdateResourceState(modelID string, state modelPB.Model_State, progress *int32, workflowId *string) error
	DeleteResourceState(modelID string) error
}

type service struct {
	repository                  repository.Repository
	triton                      triton.Triton
	redisClient                 *redis.Client
	pipelinePublicServiceClient pipelinePB.PipelinePublicServiceClient
	temporalClient              client.Client
	controllerClient            controllerPB.ControllerPrivateServiceClient
}

func NewService(r repository.Repository, t triton.Triton, p pipelinePB.PipelinePublicServiceClient, rc *redis.Client, tc client.Client, cs controllerPB.ControllerPrivateServiceClient) Service {
	return &service{
		repository:                  r,
		triton:                      t,
		pipelinePublicServiceClient: p,
		redisClient:                 rc,
		temporalClient:              tc,
		controllerClient:            cs,
	}
}

func (s *service) DeployModel(modelUID uuid.UUID) error {
	var tEnsembleModel datamodel.TritonModel
	var err error

	if tEnsembleModel, err = s.repository.GetTritonEnsembleModel(modelUID); err != nil {
		return err
	}
	// Load one ensemble model, which will also load all its dependent models
	if _, err = s.triton.LoadModelRequest(tEnsembleModel.Name); err != nil {
		if err1 := s.repository.UpdateModel(modelUID, datamodel.Model{
			State: datamodel.ModelState(modelPB.Model_STATE_ERROR),
		}); err1 != nil {
			return err1
		}
		return err
	}

	if err = s.repository.UpdateModel(modelUID, datamodel.Model{
		State: datamodel.ModelState(modelPB.Model_STATE_ONLINE),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) UndeployModel(modelUID uuid.UUID) error {

	var tritonModels []datamodel.TritonModel
	var err error

	if tritonModels, err = s.repository.GetTritonModels(modelUID); err != nil {
		return err
	}

	for _, tm := range tritonModels {
		// Unload all models composing the ensemble model
		if _, err = s.triton.UnloadModelRequest(tm.Name); err != nil {
			// If any models unloaded with error, we set the ensemble model status with ERROR and return
			if err1 := s.repository.UpdateModel(modelUID, datamodel.Model{
				State: datamodel.ModelState(modelPB.Model_STATE_ERROR),
			}); err1 != nil {
				return err1
			}
			return err
		}
	}

	if err := s.repository.UpdateModel(modelUID, datamodel.Model{
		State: datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) GetModelById(owner string, modelID string, view modelPB.View) (datamodel.Model, error) {
	return s.repository.GetModelById(owner, modelID, view)
}

func (s *service) GetModelByIdAdmin(modelID string, view modelPB.View) (datamodel.Model, error) {
	return s.repository.GetModelByIdAdmin(modelID, view)
}

func (s *service) GetModelByUid(owner string, uid uuid.UUID, view modelPB.View) (datamodel.Model, error) {
	return s.repository.GetModelByUid(owner, uid, view)
}

func (s *service) GetModelByUidAdmin(uid uuid.UUID, view modelPB.View) (datamodel.Model, error) {
	return s.repository.GetModelByUidAdmin(uid, view)
}

func (s *service) ModelInferTestMode(owner string, modelUID uuid.UUID, inferInput InferInput, task modelPB.Model_Task) ([]*modelPB.TaskOutput, error) {
	// Increment trigger image numbers
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	uid, _ := resource.GetPermalinkUID(owner)
	switch task {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION,
		modelPB.Model_TASK_UNSPECIFIED:

		if strings.HasPrefix(owner, "users/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:test.num", uid), int64(len(inferInput.([][]byte))))
		} else if strings.HasPrefix(owner, "orgs/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:test.num", uid), int64(len(inferInput.([][]byte))))
		}
	case modelPB.Model_TASK_TEXT_TO_IMAGE:
		if strings.HasPrefix(owner, "users/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:test.num", uid), 1)
		} else if strings.HasPrefix(owner, "orgs/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:test.num", uid), 1)
		}
	case modelPB.Model_TASK_TEXT_GENERATION:
		if strings.HasPrefix(owner, "users/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:test.num", uid), 1)
		} else if strings.HasPrefix(owner, "orgs/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:test.num", uid), 1)
		}
	default:
		return nil, fmt.Errorf("unknown task input type")
	}

	return s.ModelInfer(modelUID, inferInput, task)
}

func (s *service) CheckModel(modelUID uuid.UUID) (*modelPB.Model_State, error) {
	ensembleModel, err := s.repository.GetTritonEnsembleModel(modelUID)
	if err != nil {
		return nil, fmt.Errorf("triton model not found")
	}

	ensembleModelName := ensembleModel.Name
	ensembleModelVersion := ensembleModel.Version
	modelReadyResponse := s.triton.ModelReadyRequest(ensembleModelName, fmt.Sprint(ensembleModelVersion))

	state := modelPB.Model_STATE_UNSPECIFIED
	if modelReadyResponse == nil {
		state = modelPB.Model_STATE_ERROR
	} else if modelReadyResponse.Ready {
		state = modelPB.Model_STATE_ONLINE
	} else {
		state = modelPB.Model_STATE_OFFLINE
	}

	return &state, nil
}

func (s *service) WatchModel(name string) (*controllerPB.GetResourceResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *service) ModelInfer(modelUID uuid.UUID, inferInput InferInput, task modelPB.Model_Task) ([]*modelPB.TaskOutput, error) {

	ensembleModel, err := s.repository.GetTritonEnsembleModel(modelUID)
	if err != nil {
		return nil, fmt.Errorf("triton model not found")
	}

	ensembleModelName := ensembleModel.Name
	ensembleModelVersion := ensembleModel.Version
	modelMetadataResponse := s.triton.ModelMetadataRequest(ensembleModelName, fmt.Sprint(ensembleModelVersion))
	if modelMetadataResponse == nil {
		return nil, fmt.Errorf("model is offline")
	}
	modelConfigResponse := s.triton.ModelConfigRequest(ensembleModelName, fmt.Sprint(ensembleModelVersion))
	if modelMetadataResponse == nil {
		return nil, err
	}

	// We use a simple model that takes 2 input tensors of 16 integers
	// each and returns 2 output tensors of 16 integers each. One
	// output tensor is the element-wise sum of the inputs and one
	// output is the element-wise difference.
	inferResponse, err := s.triton.ModelInferRequest(task, inferInput, ensembleModelName, fmt.Sprint(ensembleModelVersion), modelMetadataResponse, modelConfigResponse)
	if err != nil {
		return nil, err
	}

	// We expect there to be 2 results (each with batch-size 1). Walk
	// over all 16 result elements and print the sum and difference
	// calculated by the modelPB.
	postprocessResponse, err := s.triton.PostProcess(inferResponse, modelMetadataResponse, task)
	if err != nil {
		return nil, err
	}

	switch task {
	case modelPB.Model_TASK_CLASSIFICATION:
		clsResponses := postprocessResponse.([]string)
		var clsOutputs []*modelPB.TaskOutput
		for _, clsRes := range clsResponses {
			clsResSplit := strings.Split(clsRes, ":")
			if len(clsResSplit) == 2 {
				score, err := strconv.ParseFloat(clsResSplit[0], 32)
				if err != nil {
					return nil, fmt.Errorf("unable to decode inference output")
				}
				clsOutput := modelPB.TaskOutput{
					Output: &modelPB.TaskOutput_Classification{
						Classification: &modelPB.ClassificationOutput{
							Category: clsResSplit[1],
							Score:    float32(score),
						},
					},
				}
				clsOutputs = append(clsOutputs, &clsOutput)
			} else if len(clsResSplit) == 3 {
				score, err := strconv.ParseFloat(clsResSplit[0], 32)
				if err != nil {
					return nil, fmt.Errorf("unable to decode inference output")
				}
				clsOutput := modelPB.TaskOutput{
					Output: &modelPB.TaskOutput_Classification{
						Classification: &modelPB.ClassificationOutput{
							Category: clsResSplit[2],
							Score:    float32(score),
						},
					},
				}
				clsOutputs = append(clsOutputs, &clsOutput)
			} else {
				return nil, fmt.Errorf("unable to decode inference output")
			}
		}
		if len(clsOutputs) == 0 {
			clsOutputs = append(clsOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Classification{
					Classification: &modelPB.ClassificationOutput{},
				},
			})
		}
		return clsOutputs, nil
	case modelPB.Model_TASK_DETECTION:
		detResponses := postprocessResponse.(triton.DetectionOutput)
		batchedOutputDataBboxes := detResponses.Boxes
		batchedOutputDataLabels := detResponses.Labels
		var detOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataBboxes {
			var detOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Detection{
					Detection: &modelPB.DetectionOutput{
						Objects: []*modelPB.DetectionObject{},
					},
				},
			}
			for j := range batchedOutputDataBboxes[i] {
				box := batchedOutputDataBboxes[i][j]
				label := batchedOutputDataLabels[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and label "0" for Triton to be able to batch Tensors
				if label != "0" {
					bbObj := &modelPB.DetectionObject{
						Category: label,
						Score:    box[4],
						// Convert x1y1x2y2 to xywh where xy is top-left corner
						BoundingBox: &modelPB.BoundingBox{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2] - box[0],
							Height: box[3] - box[1],
						},
					}
					detOutput.GetDetection().Objects = append(detOutput.GetDetection().Objects, bbObj)
				}
			}
			detOutputs = append(detOutputs, &detOutput)
		}
		if len(detOutputs) == 0 {
			detOutputs = append(detOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Detection{
					Detection: &modelPB.DetectionOutput{
						Objects: []*modelPB.DetectionObject{},
					},
				},
			})
		}
		return detOutputs, nil
	case modelPB.Model_TASK_KEYPOINT:
		keypointResponse := postprocessResponse.(triton.KeypointOutput)
		var keypointOutputs []*modelPB.TaskOutput
		for i := range keypointResponse.Keypoints { // batch size
			var keypointObjects []*modelPB.KeypointObject
			for j := range keypointResponse.Keypoints[i] { // n keypoints in one image
				if keypointResponse.Scores[i][j] == -1 { // dummy object for batching to make sure every images have same output shape
					continue
				}
				var keypoints []*modelPB.Keypoint
				points := keypointResponse.Keypoints[i][j]
				for k := range points { // 17 point for each keypoint
					if points[k][0] == -1 && points[k][1] == -1 && points[k][2] == -1 { // dummy output for batching to make sure every images have same output shape
						continue
					}
					keypoints = append(keypoints, &modelPB.Keypoint{
						X: points[k][0],
						Y: points[k][1],
						V: points[k][2],
					})
				}
				keypointObjects = append(keypointObjects, &modelPB.KeypointObject{
					Keypoints: keypoints,
					BoundingBox: &modelPB.BoundingBox{
						Left:   keypointResponse.Boxes[i][j][0],
						Top:    keypointResponse.Boxes[i][j][1],
						Width:  keypointResponse.Boxes[i][j][2],
						Height: keypointResponse.Boxes[i][j][3],
					},
					Score: keypointResponse.Scores[i][j],
				})
			}
			keypointOutputs = append(keypointOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Keypoint{
					Keypoint: &modelPB.KeypointOutput{
						Objects: keypointObjects,
					},
				},
			})
		}
		if len(keypointOutputs) == 0 {
			keypointOutputs = append(keypointOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Keypoint{
					Keypoint: &modelPB.KeypointOutput{
						Objects: []*modelPB.KeypointObject{},
					},
				},
			})
		}
		return keypointOutputs, nil
	case modelPB.Model_TASK_OCR:
		ocrResponses := postprocessResponse.(triton.OcrOutput)
		batchedOutputDataBboxes := ocrResponses.Boxes
		batchedOutputDataTexts := ocrResponses.Texts
		batchedOutputDataScores := ocrResponses.Scores
		var ocrOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataBboxes {
			var ocrOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Ocr{
					Ocr: &modelPB.OcrOutput{
						Objects: []*modelPB.OcrObject{},
					},
				},
			}
			for j := range batchedOutputDataBboxes[i] {
				box := batchedOutputDataBboxes[i][j]
				text := batchedOutputDataTexts[i][j]
				score := batchedOutputDataScores[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Triton to be able to batch Tensors
				if text != "" && box[0] != -1 {
					ocrOutput.GetOcr().Objects = append(ocrOutput.GetOcr().Objects, &modelPB.OcrObject{
						BoundingBox: &modelPB.BoundingBox{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2],
							Height: box[3],
						},
						Score: score,
						Text:  text,
					})
				}
			}
			ocrOutputs = append(ocrOutputs, &ocrOutput)
		}
		if len(ocrOutputs) == 0 {
			ocrOutputs = append(ocrOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Ocr{
					Ocr: &modelPB.OcrOutput{
						Objects: []*modelPB.OcrObject{},
					},
				},
			})
		}
		return ocrOutputs, nil

	case modelPB.Model_TASK_INSTANCE_SEGMENTATION:
		instanceSegmentationResponses := postprocessResponse.(triton.InstanceSegmentationOutput)
		batchedOutputDataRles := instanceSegmentationResponses.Rles
		batchedOutputDataBboxes := instanceSegmentationResponses.Boxes
		batchedOutputDataLabels := instanceSegmentationResponses.Labels
		batchedOutputDataScores := instanceSegmentationResponses.Scores
		var instanceSegmentationOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataBboxes {
			var instanceSegmentationOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_InstanceSegmentation{
					InstanceSegmentation: &modelPB.InstanceSegmentationOutput{
						Objects: []*modelPB.InstanceSegmentationObject{},
					},
				},
			}
			for j := range batchedOutputDataBboxes[i] {
				rle := batchedOutputDataRles[i][j]
				box := batchedOutputDataBboxes[i][j]
				label := batchedOutputDataLabels[i][j]
				score := batchedOutputDataScores[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Triton to be able to batch Tensors
				if label != "" && rle != "" {
					instanceSegmentationOutput.GetInstanceSegmentation().Objects = append(instanceSegmentationOutput.GetInstanceSegmentation().Objects, &modelPB.InstanceSegmentationObject{
						Rle: rle,
						BoundingBox: &modelPB.BoundingBox{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2],
							Height: box[3],
						},
						Score:    score,
						Category: label,
					})
				}
			}
			instanceSegmentationOutputs = append(instanceSegmentationOutputs, &instanceSegmentationOutput)
		}
		if len(instanceSegmentationOutputs) == 0 {
			instanceSegmentationOutputs = append(instanceSegmentationOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_InstanceSegmentation{
					InstanceSegmentation: &modelPB.InstanceSegmentationOutput{
						Objects: []*modelPB.InstanceSegmentationObject{},
					},
				},
			})
		}
		return instanceSegmentationOutputs, nil

	case modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
		semanticSegmentationResponses := postprocessResponse.(triton.SemanticSegmentationOutput)
		batchedOutputDataRles := semanticSegmentationResponses.Rles
		batchedOutputDataCategories := semanticSegmentationResponses.Categories
		var semanticSegmentationOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataCategories { // loop over images
			var semanticSegmentationOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_SemanticSegmentation{
					SemanticSegmentation: &modelPB.SemanticSegmentationOutput{
						Stuffs: []*modelPB.SemanticSegmentationStuff{},
					},
				},
			}
			for j := range batchedOutputDataCategories[i] { // single image
				rle := batchedOutputDataRles[i][j]
				category := batchedOutputDataCategories[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Triton to be able to batch Tensors
				if category != "" && rle != "" {
					semanticSegmentationOutput.GetSemanticSegmentation().Stuffs = append(semanticSegmentationOutput.GetSemanticSegmentation().Stuffs, &modelPB.SemanticSegmentationStuff{
						Rle:      rle,
						Category: category,
					})
				}
			}
			semanticSegmentationOutputs = append(semanticSegmentationOutputs, &semanticSegmentationOutput)
		}
		if len(semanticSegmentationOutputs) == 0 {
			semanticSegmentationOutputs = append(semanticSegmentationOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_SemanticSegmentation{
					SemanticSegmentation: &modelPB.SemanticSegmentationOutput{
						Stuffs: []*modelPB.SemanticSegmentationStuff{},
					},
				},
			})
		}
		return semanticSegmentationOutputs, nil
	case modelPB.Model_TASK_TEXT_TO_IMAGE:
		textToImageResponses := postprocessResponse.(triton.TextToImageOutput)
		batchedOutputDataImages := textToImageResponses.Images
		var textToImageOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataImages { // loop over images
			var textToImageOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextToImage{
					TextToImage: &modelPB.TextToImageOutput{
						Images: batchedOutputDataImages[i],
					},
				},
			}

			textToImageOutputs = append(textToImageOutputs, &textToImageOutput)
		}
		if len(textToImageOutputs) == 0 {
			textToImageOutputs = append(textToImageOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextToImage{
					TextToImage: &modelPB.TextToImageOutput{
						Images: []string{},
					},
				},
			})
		}
		return textToImageOutputs, nil
	case modelPB.Model_TASK_TEXT_GENERATION:
		textGenerationResponses := postprocessResponse.(triton.TextGenerationOutput)
		batchedOutputDataTexts := textGenerationResponses.Text
		var textGenerationOutputs []*modelPB.TaskOutput
		for i := range batchedOutputDataTexts {
			var textGenerationOutput = modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextGeneration{
					TextGeneration: &modelPB.TextGenerationOutput{
						Text: batchedOutputDataTexts[i],
					},
				},
			}

			textGenerationOutputs = append(textGenerationOutputs, &textGenerationOutput)
		}
		if len(textGenerationOutputs) == 0 {
			textGenerationOutputs = append(textGenerationOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_TextGeneration{
					TextGeneration: &modelPB.TextGenerationOutput{
						Text: "",
					},
				},
			})
		}
		return textGenerationOutputs, nil
	default:
		outputs := postprocessResponse.([]triton.BatchUnspecifiedTaskOutputs)
		var rawOutputs []*modelPB.TaskOutput
		if len(outputs) == 0 {
			return []*modelPB.TaskOutput{}, nil
		}
		deserializedOutputs := outputs[0].SerializedOutputs
		for i := range deserializedOutputs {
			var singleImageOutput []*structpb.Struct

			for _, output := range outputs {
				unspecifiedOutput := triton.SingleOutputUnspecifiedTaskOutput{
					Name:     output.Name,
					Shape:    output.Shape,
					DataType: output.DataType,
					Data:     output.SerializedOutputs[i],
				}

				var mapOutput map[string]interface{}
				b, err := json.Marshal(unspecifiedOutput)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(b, &mapOutput); err != nil {
					return nil, err
				}
				util.ConvertAllJSONKeySnakeCase(mapOutput)

				b, err = json.Marshal(mapOutput)
				if err != nil {
					return nil, err
				}
				var structData = &structpb.Struct{}
				err = protojson.Unmarshal(b, structData)

				if err != nil {
					return nil, err
				}
				singleImageOutput = append(singleImageOutput, structData)
			}

			rawOutputs = append(rawOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Unspecified{
					Unspecified: &modelPB.UnspecifiedOutput{
						RawOutputs: singleImageOutput,
					},
				},
			})
		}
		if len(rawOutputs) == 0 {
			rawOutputs = append(rawOutputs, &modelPB.TaskOutput{
				Output: &modelPB.TaskOutput_Unspecified{
					Unspecified: &modelPB.UnspecifiedOutput{
						RawOutputs: []*structpb.Struct{},
					},
				},
			})
		}
		return rawOutputs, nil
	}
}

func (s *service) ListModels(owner string, view modelPB.View, pageSize int, pageToken string) ([]datamodel.Model, string, int64, error) {
	return s.repository.ListModels(owner, view, pageSize, pageToken)
}

func (s *service) ListModelsAdmin(view modelPB.View, pageSize int, pageToken string) ([]datamodel.Model, string, int64, error) {
	return s.repository.ListModelsAdmin(view, pageSize, pageToken)
}

func (s *service) DeleteModel(owner string, modelID string) error {
	logger, _ := logger.GetZapLogger()

	modelInDB, err := s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return err
	}

	filter := fmt.Sprintf("recipe.models:\"models/%s\"", modelInDB.UID)

	pipeResp, err := s.pipelinePublicServiceClient.ListPipelines(context.Background(), &pipelinePB.ListPipelinesRequest{
		Filter: &filter,
	})
	if err != nil {
		return err
	}
	if len(pipeResp.Pipelines) > 0 {
		var pipeIDs []string
		for _, pipe := range pipeResp.Pipelines {
			pipeIDs = append(pipeIDs, pipe.GetId())
		}
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] delete model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "DELETE",
					Subject:     fmt.Sprintf("id %s", modelInDB.ID),
					Description: fmt.Sprintf("The model is still in use by pipeline: %s ", strings.Join(pipeIDs, " ")),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	state, err := s.GetResourceState(modelID)
	if err != nil {
		return err
	}

	if *state == modelPB.Model_STATE_UNSPECIFIED {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] delete model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "DELETE",
					Subject:     fmt.Sprintf("id %s", modelInDB.ID),
					Description: "The model is still in operations, please wait the operation finish and try it again",
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	if err := s.UndeployModel(modelInDB.UID); err != nil {
		s.UpdateResourceState(
			modelID,
			modelPB.Model_STATE_ERROR,
			nil,
			nil,
		)
		return err
	}

	// remove README.md
	_ = os.RemoveAll(fmt.Sprintf("%v/%v#%v#README.md", config.Config.TritonServer.ModelStore, owner, modelInDB.ID))
	tritonModels, err := s.repository.GetTritonModels(modelInDB.UID)
	if err == nil {
		// remove model folders
		for i := 0; i < len(tritonModels); i++ {
			modelDir := filepath.Join(config.Config.TritonServer.ModelStore, tritonModels[i].Name)
			_ = os.RemoveAll(modelDir)
		}
	}

	if err := s.DeleteResourceState(modelID); err != nil {
		return err
	}

	return s.repository.DeleteModel(modelInDB.UID)
}

func (s *service) RenameModel(owner string, modelID string, newModelId string) (datamodel.Model, error) {
	modelInDB, err := s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return datamodel.Model{}, err
	}

	if err := s.DeleteResourceState(modelID); err != nil {
		return datamodel.Model{}, err
	}

	if err := s.UpdateResourceState(newModelId, modelPB.Model_State(modelInDB.State), nil, nil); err != nil {
		return datamodel.Model{}, err
	}

	err = s.repository.UpdateModel(modelInDB.UID, datamodel.Model{
		ID: newModelId,
	})
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(owner, newModelId, modelPB.View_VIEW_FULL)
}

func (s *service) PublishModel(owner string, modelID string) (datamodel.Model, error) {
	modelInDB, err := s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return datamodel.Model{}, err
	}

	err = s.repository.UpdateModel(modelInDB.UID, datamodel.Model{
		ID:         modelInDB.ID,
		Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PUBLIC),
	})
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
}

func (s *service) UnpublishModel(owner string, modelID string) (datamodel.Model, error) {
	modelInDB, err := s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
	if err != nil {
		return datamodel.Model{}, err
	}

	err = s.repository.UpdateModel(modelInDB.UID, datamodel.Model{
		ID:         modelInDB.ID,
		Visibility: datamodel.ModelVisibility(modelPB.Model_VISIBILITY_PRIVATE),
	})
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(owner, modelID, modelPB.View_VIEW_FULL)
}

func (s *service) UpdateModel(modelUID uuid.UUID, model *datamodel.Model) (datamodel.Model, error) {
	err := s.repository.UpdateModel(modelUID, *model)
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(model.Owner, model.ID, modelPB.View_VIEW_FULL)
}

// TODO: gorm do not update the zero value with struct, so we need to update the state manually.
func (s *service) UpdateModelState(modelUID uuid.UUID, model *datamodel.Model, state datamodel.ModelState) (datamodel.Model, error) {
	err := s.repository.UpdateModelState(modelUID, state)
	if err != nil {
		return datamodel.Model{}, err
	}

	return s.GetModelById(model.Owner, model.ID, modelPB.View_VIEW_FULL)
}

func (s *service) GetModelDefinition(id string) (datamodel.ModelDefinition, error) {
	return s.repository.GetModelDefinition(id)
}

func (s *service) GetModelDefinitionByUid(uid uuid.UUID) (datamodel.ModelDefinition, error) {
	return s.repository.GetModelDefinitionByUid(uid)
}

func (s *service) ListModelDefinitions(view modelPB.View, pageSize int, pageToken string) ([]datamodel.ModelDefinition, string, int64, error) {
	return s.repository.ListModelDefinitions(view, pageSize, pageToken)
}

func (s *service) GetTritonEnsembleModel(modelUID uuid.UUID) (datamodel.TritonModel, error) {
	return s.repository.GetTritonEnsembleModel(modelUID)
}

func (s *service) GetTritonModels(modelUID uuid.UUID) ([]datamodel.TritonModel, error) {
	return s.repository.GetTritonModels(modelUID)
}
