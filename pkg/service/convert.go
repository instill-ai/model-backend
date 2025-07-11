package service

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"slices"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid"
	"golang.org/x/image/draw"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/x/constant"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	logx "github.com/instill-ai/x/log"
	resourcex "github.com/instill-ai/x/resource"
)

func (s *service) compressProfileImage(profileImage string) (string, error) {

	// Due to the local env, we don't set the `InstillCoreHost` config, the avatar path is not working.
	// As a workaround, if the profileAvatar is not a base64 string, we ignore the avatar.
	if !strings.HasPrefix(profileImage, "data:") {
		return "", nil
	}

	profileImageStr := strings.Split(profileImage, ",")
	b, err := base64.StdEncoding.DecodeString(profileImageStr[len(profileImageStr)-1])
	if err != nil {
		return "", err
	}
	if len(b) > 200*1024 {
		mimeType := strings.Split(mimetype.Detect(b).String(), ";")[0]

		var src image.Image
		switch mimeType {
		case "image/png":
			src, _ = png.Decode(bytes.NewReader(b))
		case "image/jpeg":
			src, _ = jpeg.Decode(bytes.NewReader(b))
		default:
			return "", status.Errorf(codes.InvalidArgument, "only support profile image in jpeg and png formats")
		}

		// Set the expected size that you want:
		dst := image.NewRGBA(image.Rect(0, 0, 256, 256*src.Bounds().Max.Y/src.Bounds().Max.X))

		// Resize:
		draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

		var buf bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		err = encoder.Encode(bufio.NewWriter(&buf), dst)
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, "profile image error")
		}
		profileImage = fmt.Sprintf("data:%s;base64,%s", "image/png", base64.StdEncoding.EncodeToString(buf.Bytes()))
	}
	return profileImage, nil
}

func (s *service) PBToDBModel(ctx context.Context, ns resource.Namespace, pbModel *modelpb.Model) (*datamodel.Model, error) {
	logger, _ := logx.GetZapLogger(ctx)

	profileImage, err := s.compressProfileImage(pbModel.GetProfileImage())
	if err != nil {
		return nil, err
	}

	return &datamodel.Model{
		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbModel.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbModel.GetUid())
				if err != nil {
					logger.Fatal(err.Error())
				}
				return id
			}(),
			CreateTime: func() time.Time {
				if pbModel.GetCreateTime() != nil {
					return pbModel.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbModel.GetUpdateTime() != nil {
					return pbModel.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},
		Owner:         ns.Permalink(),
		ID:            pbModel.GetId(),
		NamespaceID:   ns.NsID,
		NamespaceType: string(ns.NsType),
		Description: sql.NullString{
			String: pbModel.GetDescription(),
			Valid:  true,
		},
		Task:       datamodel.ModelTask(pbModel.GetTask()),
		Visibility: datamodel.ModelVisibility(pbModel.GetVisibility()),
		Region:     pbModel.GetRegion(),
		Hardware:   pbModel.GetHardware(),
		Readme: sql.NullString{
			String: pbModel.GetReadme(),
			Valid:  true,
		},
		SourceURL: sql.NullString{
			String: pbModel.GetSourceUrl(),
			Valid:  true,
		},
		DocumentationURL: sql.NullString{
			String: pbModel.GetDocumentationUrl(),
			Valid:  true,
		},
		License: sql.NullString{
			String: pbModel.GetLicense(),
			Valid:  true,
		},
		ProfileImage: sql.NullString{
			String: profileImage,
			Valid:  profileImage != "",
		},
	}, nil
}

func (s *service) DBToPBModel(ctx context.Context, modelDef *datamodel.ModelDefinition, dbModel *datamodel.Model, view modelpb.View, checkPermission bool) (*modelpb.Model, error) {
	logger, _ := logx.GetZapLogger(ctx)

	ownerName := fmt.Sprintf("%s/%s", dbModel.NamespaceType, dbModel.NamespaceID)

	ctxUserUID := resourcex.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	profileImage := fmt.Sprintf("%s/v1alpha/%s/models/%s/image", s.instillCoreHost, ownerName, dbModel.ID)

	pbModel := &modelpb.Model{
		Name:       fmt.Sprintf("%s/models/%s", ownerName, dbModel.ID),
		Uid:        dbModel.UID.String(),
		Id:         dbModel.ID,
		CreateTime: timestamppb.New(dbModel.CreateTime),
		UpdateTime: timestamppb.New(dbModel.UpdateTime),
		DeleteTime: func() *timestamppb.Timestamp {
			if dbModel.DeleteTime.Time.IsZero() {
				return nil
			} else {
				return timestamppb.New(dbModel.DeleteTime.Time)
			}
		}(),
		Description:     &dbModel.Description.String,
		ModelDefinition: fmt.Sprintf("model-definitions/%s", modelDef.ID),
		Visibility:      modelpb.Model_Visibility(dbModel.Visibility),
		Task:            commonpb.Task(dbModel.Task),
		Configuration: func() *structpb.Struct {
			if dbModel.Configuration != nil {
				str := structpb.Struct{}
				err := str.UnmarshalJSON(dbModel.Configuration)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return &str
			}
			return nil
		}(),
		OwnerName:        ownerName,
		Region:           dbModel.Region,
		Hardware:         dbModel.Hardware,
		SourceUrl:        &dbModel.SourceURL.String,
		DocumentationUrl: &dbModel.DocumentationURL.String,
		License:          &dbModel.License.String,
		Readme:           &dbModel.Readme.String,
		ProfileImage:     &profileImage,
		Tags: func() []string {
			tagNames := make([]string, len(dbModel.TagNames()))
			for i, tag := range dbModel.TagNames() {
				if slices.Contains(preserveTags, tag) {
					tagNames[i] = cases.Title(language.English).String(tag)
				} else {
					tagNames[i] = tag
				}
			}
			return tagNames
		}(),
		Versions: dbModel.VersionNames(),
		Stats: &modelpb.Model_Stats{
			NumberOfRuns: int32(dbModel.NumberOfRuns),
			LastRunTime:  timestamppb.New(dbModel.LastRunTime),
		},
	}

	pbModel.Permission = &modelpb.Permission{}
	if checkPermission {
		if strings.Split(dbModel.Owner, "/")[1] == ctxUserUID {
			pbModel.Permission.CanEdit = true
			pbModel.Permission.CanTrigger = true
		} else {
			canEdit, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "writer")
			if err != nil {
				return nil, err
			}
			pbModel.Permission.CanEdit = canEdit

			canTrigger, err := s.aclClient.CheckPermission(ctx, "model_", dbModel.UID, "executor")
			if err != nil {
				return nil, err
			}
			pbModel.Permission.CanTrigger = canTrigger
		}
	}

	var owner *mgmtpb.Owner
	owner, err := s.FetchOwnerWithPermalink(ctx, dbModel.Owner)
	if err != nil {
		return nil, err
	}
	pbModel.Owner = owner

	if view > modelpb.View_VIEW_BASIC {

		inputTaskSchema, ok := datamodel.TasksJSONMap[pbModel.Task.String()]["input"]
		if !ok {
			return nil, fmt.Errorf("value for key %s is not a JSON object", pbModel.Task.String())
		}
		outputTaskSchema, ok := datamodel.TasksJSONMap[pbModel.Task.String()]["output"]
		if !ok {
			return nil, fmt.Errorf("value for key %s is not a JSON object", pbModel.Task.String())
		}
		inputSchemaStruct, _ := structpb.NewStruct(inputTaskSchema.(map[string]any))
		outputSchemaStruct, _ := structpb.NewStruct(outputTaskSchema.(map[string]any))
		pbModel.InputSchema = inputSchemaStruct
		pbModel.OutputSchema = outputSchemaStruct

		// appendSampleInputOutput(&pbModel)
	}

	return pbModel, nil
}

func (s *service) DBToPBModels(ctx context.Context, dbModels []*datamodel.Model, view modelpb.View, checkPermission bool) ([]*modelpb.Model, error) {

	pbModels := make([]*modelpb.Model, len(dbModels))

	for idx := range dbModels {
		modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModels[idx].ModelDefinitionUID)
		if err != nil {
			return nil, err
		}

		pbModels[idx], err = s.DBToPBModel(
			ctx,
			modelDef,
			dbModels[idx],
			view,
			checkPermission,
		)
		if err != nil {
			return nil, err
		}
	}

	return pbModels, nil
}

func (s *service) DBToPBModelDefinition(ctx context.Context, dbModelDefinition *datamodel.ModelDefinition) (*modelpb.ModelDefinition, error) {
	logger, _ := logx.GetZapLogger(ctx)

	pbModelDefinition := modelpb.ModelDefinition{
		Name:             fmt.Sprintf("model-definitions/%s", dbModelDefinition.ID),
		Id:               dbModelDefinition.ID,
		Uid:              dbModelDefinition.UID.String(),
		Title:            dbModelDefinition.Title,
		DocumentationUrl: dbModelDefinition.DocumentationURL,
		Icon:             dbModelDefinition.Icon,
		CreateTime:       timestamppb.New(dbModelDefinition.CreateTime),
		UpdateTime:       timestamppb.New(dbModelDefinition.UpdateTime),
		ReleaseStage:     modelpb.ReleaseStage(dbModelDefinition.ReleaseStage),
		ModelSpec: func() *structpb.Struct {
			if dbModelDefinition.ModelSpec != nil {
				var specification = &structpb.Struct{}
				if err := protojson.Unmarshal([]byte(dbModelDefinition.ModelSpec.String()), specification); err != nil {
					logger.Fatal(err.Error())
				}
				return specification
			} else {
				return nil
			}
		}(),
	}

	return &pbModelDefinition, nil
}

func (s *service) DBToPBModelDefinitions(ctx context.Context, dbModelDefinitions []*datamodel.ModelDefinition) ([]*modelpb.ModelDefinition, error) {

	var err error
	pbModelDefinitions := make([]*modelpb.ModelDefinition, len(dbModelDefinitions))

	for idx := range dbModelDefinitions {
		pbModelDefinitions[idx], err = s.DBToPBModelDefinition(
			ctx,
			dbModelDefinitions[idx],
		)
		if err != nil {
			return nil, err
		}
	}

	return pbModelDefinitions, nil
}

// func appendSampleInputOutput(pbModel *modelpb.Model) {
// 	steps := int32(10)
// 	cfgScale := float32(7)
// 	samples := int32(1)
// 	maxNewTokens := int32(512)
// 	topK := int32(10)
// 	temperature := float32(0.7)
// 	seed := int32(1024)

// 	sampleInput := modelpb.TaskInput{}
// 	sampleOutput := modelpb.TaskOutput{}
// 	switch pbModel.Task {
// 	case commonpb.Task_TASK_CLASSIFICATION:
// 		sampleInput.Input = &modelpb.TaskInput_Classification{
// 			Classification: &modelpb.ClassificationInput{
// 				Type: &modelpb.ClassificationInput_ImageUrl{
// 					ImageUrl: "https://artifacts.instill-ai.com/imgs/dog.jpg",
// 				},
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_Classification{
// 			Classification: &modelpb.ClassificationOutput{
// 				Category: "golden retriever",
// 				Score:    0.98,
// 			},
// 		}
// 	case commonpb.Task_TASK_DETECTION:
// 		sampleInput.Input = &modelpb.TaskInput_Detection{
// 			Detection: &modelpb.DetectionInput{
// 				Type: &modelpb.DetectionInput_ImageUrl{
// 					ImageUrl: "https://artifacts.instill-ai.com/imgs/dog.jpg",
// 				},
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_Detection{
// 			Detection: &modelpb.DetectionOutput{
// 				Objects: []*modelpb.DetectionObject{
// 					{
// 						Category: "dog",
// 						Score:    0.9582795,
// 						BoundingBox: &modelpb.BoundingBox{
// 							Top:    102,
// 							Left:   324,
// 							Width:  208,
// 							Height: 403,
// 						},
// 					},
// 					{
// 						Category: "dog",
// 						Score:    0.9457829,
// 						BoundingBox: &modelpb.BoundingBox{
// 							Top:    198,
// 							Left:   130,
// 							Width:  198,
// 							Height: 236,
// 						},
// 					},
// 				},
// 			},
// 		}
// 	case commonpb.Task_TASK_KEYPOINT:
// 		sampleInput.Input = &modelpb.TaskInput_Keypoint{
// 			Keypoint: &modelpb.KeypointInput{
// 				Type: &modelpb.KeypointInput_ImageUrl{
// 					ImageUrl: "https://artifacts.instill-ai.com/imgs/dance.jpg",
// 				},
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_Keypoint{
// 			Keypoint: &modelpb.KeypointOutput{
// 				Objects: []*modelpb.KeypointObject{
// 					{
// 						Keypoints: []*modelpb.Keypoint{
// 							{
// 								X: 542.82764,
// 								Y: 86.63817,
// 								V: 0.53722847,
// 							},
// 							{
// 								X: 553.0073,
// 								Y: 79.440636,
// 								V: 0.634061,
// 							},
// 						},
// 						Score: 0.94,
// 						BoundingBox: &modelpb.BoundingBox{
// 							Top:    86,
// 							Left:   185,
// 							Width:  571,
// 							Height: 203,
// 						},
// 					},
// 				},
// 			},
// 		}
// 	case commonpb.Task_TASK_OCR:
// 		sampleInput.Input = &modelpb.TaskInput_Ocr{
// 			Ocr: &modelpb.OcrInput{
// 				Type: &modelpb.OcrInput_ImageUrl{
// 					ImageUrl: "https://artifacts.instill-ai.com/imgs/sign-small.jpg",
// 				},
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_Ocr{
// 			Ocr: &modelpb.OcrOutput{
// 				Objects: []*modelpb.OcrObject{
// 					{
// 						Text:  "ENDS",
// 						Score: 0.99,
// 						BoundingBox: &modelpb.BoundingBox{
// 							Top:    298,
// 							Left:   279,
// 							Width:  134,
// 							Height: 59,
// 						},
// 					},
// 					{
// 						Text:  "PAVEMENT",
// 						Score: 0.99,
// 						BoundingBox: &modelpb.BoundingBox{
// 							Top:    228,
// 							Left:   216,
// 							Width:  255,
// 							Height: 65,
// 						},
// 					},
// 				},
// 			},
// 		}
// 	case commonpb.Task_TASK_INSTANCE_SEGMENTATION:
// 		sampleInput.Input = &modelpb.TaskInput_InstanceSegmentation{
// 			InstanceSegmentation: &modelpb.InstanceSegmentationInput{
// 				Type: &modelpb.InstanceSegmentationInput_ImageUrl{
// 					ImageUrl: "https://artifacts.instill-ai.com/imgs/dog.jpg",
// 				},
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_InstanceSegmentation{
// 			InstanceSegmentation: &modelpb.InstanceSegmentationOutput{
// 				Objects: []*modelpb.InstanceSegmentationObject{
// 					{
// 						Score: 0.99,
// 						BoundingBox: &modelpb.BoundingBox{
// 							Top:    95,
// 							Left:   320,
// 							Width:  215,
// 							Height: 406,
// 						},
// 						Category: "dog",
// 						Rle:      "472,26,35,31,31,34,28,35,27,36,25,37,25,37,24,37,24,38,23,39,23,40,22,40,22,41,21,41,21,41,21,40,22,39,22,40,22,39,23,39,23,39,24,38,25,37,26,35,28,32,31,29,34,27,36,26,37,25,38,25,38,24,39,23,40,21,42,16,47,11,53,8,55,7,50",
// 					},
// 					{
// 						Score: 0.97,
// 						BoundingBox: &modelpb.BoundingBox{
// 							Top:    194,
// 							Left:   130,
// 							Width:  197,
// 							Height: 248,
// 						},
// 						Category: "dog",
// 						Rle:      "158,43,22,45,20,47,19,48,18,49,17,50,16,51,13,54,9,58,6,60,6,60,6,60,7,59,8,59,8,58,9,57,9,56,11,48,19,45,22,44,23,43,25,41,26,40,28,38,30,35,34,25,168",
// 					},
// 				},
// 			},
// 		}
// 	case commonpb.Task_TASK_SEMANTIC_SEGMENTATION:
// 		sampleInput.Input = &modelpb.TaskInput_SemanticSegmentation{
// 			SemanticSegmentation: &modelpb.SemanticSegmentationInput{
// 				Type: &modelpb.SemanticSegmentationInput_ImageUrl{
// 					ImageUrl: "https://artifacts.instill-ai.com/imgs/dog.jpg",
// 				},
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_SemanticSegmentation{
// 			SemanticSegmentation: &modelpb.SemanticSegmentationOutput{
// 				Stuffs: []*modelpb.SemanticSegmentationStuff{
// 					{
// 						Rle:      "472,26,35,31,31,34,28,35,27,36,25,37,25,37,24,37,24,38,23,39,23,40,22,40,22,41,21,41,21,41,21,40,22,39,22,40,22,39,23,39,23,39,24,38,25,37,26,35,28,32,31,29,34,27,36,26,37,25,38,25,38,24,39,23,40,21,42,16,47,11,53,8,55,7,50",
// 						Category: "person",
// 					},
// 					{
// 						Rle:      "158,43,22,45,20,47,19,48,18,49,17,50,16,51,13,54,9,58,6,60,6,60,6,60,7,59,8,59,8,58,9,57,9,56,11,48,19,45,22,44,23,43,25,41,26,40,28,38,30,35,34,25,168",
// 						Category: "sky",
// 					},
// 				},
// 			},
// 		}
// 	case commonpb.Task_TASK_TEXT_TO_IMAGE:

// 		sampleInput.Input = &modelpb.TaskInput_TextToImage{
// 			TextToImage: &modelpb.TextToImageInput{
// 				Prompt:   "A stunning landscape with metropolitan view",
// 				CfgScale: &cfgScale,
// 				Steps:    &steps,
// 				Samples:  &samples,
// 				Seed:     &seed,
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_TextToImage{
// 			TextToImage: &modelpb.TextToImageOutput{
// 				Images: []string{
// 					"/9j/4AAQSkZJRgABAQAAAQABAAD/...",
// 				},
// 			},
// 		}
// 	case commonpb.Task_TASK_IMAGE_TO_IMAGE:
// 		prompt := "cute dog"
// 		sampleInput.Input = &modelpb.TaskInput_ImageToImage{
// 			ImageToImage: &modelpb.ImageToImageInput{
// 				Prompt: &prompt,
// 				Type: &modelpb.ImageToImageInput_PromptImageUrl{
// 					PromptImageUrl: "https://artifacts.instill-ai.com/imgs/dog.jpg",
// 				},
// 				CfgScale: &cfgScale,
// 				Steps:    &steps,
// 				Samples:  &samples,
// 				Seed:     &seed,
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_ImageToImage{
// 			ImageToImage: &modelpb.ImageToImageOutput{
// 				Images: []string{
// 					"/9j/4AAQSkZJRgABAQAAAQABAAD/...",
// 				},
// 			},
// 		}
// 	case commonpb.Task_TASK_TEXT_GENERATION:
// 		systemMessage := "You are a helpful assistant."
// 		sampleInput.Input = &modelpb.TaskInput_TextGeneration{
// 			TextGeneration: &modelpb.TextGenerationInput{
// 				Prompt:        "The winds of change",
// 				SystemMessage: &systemMessage,
// 				MaxNewTokens:  &maxNewTokens,
// 				TopK:          &topK,
// 				Temperature:   &temperature,
// 				Seed:          &seed,
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_TextGeneration{
// 			TextGeneration: &modelpb.TextGenerationOutput{
// 				Text: "The winds of change are blowing strong, bring new beginnings, righting wrongs. The world around us is constantly turning, and with each sunrise, our spirits are yearning.",
// 			},
// 		}
// 	case commonpb.Task_TASK_TEXT_GENERATION_CHAT:
// 		systemMessage := "You are a lovely cat, named Penguin."
// 		sampleInput.Input = &modelpb.TaskInput_TextGenerationChat{
// 			TextGenerationChat: &modelpb.TextGenerationChatInput{
// 				Prompt:        "Who are you?",
// 				SystemMessage: &systemMessage,
// 				MaxNewTokens:  &maxNewTokens,
// 				TopK:          &topK,
// 				Temperature:   &temperature,
// 				Seed:          &seed,
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_TextGenerationChat{
// 			TextGenerationChat: &modelpb.TextGenerationChatOutput{
// 				Text: "*rubs against leg* Oh, hello there! My name is Penguin, and I'm a lovely cat. I'm a bit of a gentle soul, with soft gray fur and bright green eyes. I love to lounge in the sunbeams that stream through the windows, chase the occasional fly, and purr contentedly as I watch the world go by. I'm a bit of a cuddlebug, too - I adore being petted and snuggled, and I'll often curl up in my human's lap for a good nap.",
// 			},
// 		}
// 	case commonpb.Task_TASK_VISUAL_QUESTION_ANSWERING:
// 		systemMessage := "You are a helpful assistant."
// 		sampleInput.Input = &modelpb.TaskInput_VisualQuestionAnswering{
// 			VisualQuestionAnswering: &modelpb.VisualQuestionAnsweringInput{
// 				Prompt: "What is in the picture?",
// 				PromptImages: []*modelpb.PromptImage{
// 					{
// 						Type: &modelpb.PromptImage_PromptImageUrl{
// 							PromptImageUrl: "https://artifacts.instill-ai.com/imgs/dog.jpg",
// 						},
// 					},
// 				},
// 				SystemMessage: &systemMessage,
// 				MaxNewTokens:  &maxNewTokens,
// 				TopK:          &topK,
// 				Temperature:   &temperature,
// 				Seed:          &seed,
// 			},
// 		}
// 		sampleOutput.Output = &modelpb.TaskOutput_VisualQuestionAnswering{
// 			VisualQuestionAnswering: &modelpb.VisualQuestionAnsweringOutput{
// 				Text: "The picture shows two dogs standing in a snowy outdoor setting. The dog on the left appears to be a young Labrador Retriever puppy with a light cream or yellowish coat, while the dog on the right is an adult",
// 			},
// 		}
// 	}
// 	pbModel.SampleInput = &sampleInput
// 	pbModel.SampleOutput = &sampleOutput
// }
