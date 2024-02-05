package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/service"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/sterr"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func createGitHubModel(s service.Service, ctx context.Context, req CreateNamespaceModelRequestInterface, ns resource.Namespace, authUser *service.AuthUser, modelDefinition *datamodel.ModelDefinition) (*longrunningpb.Operation, error) {

	eventName := "CreateGitHubModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	var modelConfig datamodel.GitHubModelConfiguration
	b, err := req.GetModel().Configuration.MarshalJSON()
	if err != nil {
		span.SetStatus(1, err.Error())
		return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		span.SetStatus(1, err.Error())
		return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if modelConfig.Repository == "" {
		span.SetStatus(1, "Invalid GitHub URL")
		return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, "Invalid GitHub URL")
	}
	var githubInfo *utils.GitHubInfo
	if config.Config.Server.ItMode.Enabled {
		githubInfo = &utils.GitHubInfo{
			Description: "This is a test model",
			Visibility:  "public",
			Tags:        []utils.Tag{{Name: "v1.0-cpu"}, {Name: "v1.1-cpu"}},
		}
	} else {
		githubInfo, err = utils.GetGitHubRepoInfo(modelConfig.Repository)
		if err != nil {
			span.SetStatus(1, "Invalid GitHub Info")
			return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid Github info: %s", err))
		}
		if len(githubInfo.Tags) == 0 {
			span.SetStatus(1, "There is no tag in GitHub repository")
			return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, "There is no tag in GitHub repository")
		}
	}
	visibility := utils.Visibility[githubInfo.Visibility]
	if req.GetModel().Visibility == modelPB.Model_VISIBILITY_PUBLIC {
		visibility = modelPB.Model_VISIBILITY_PUBLIC
	} else if req.GetModel().Visibility == modelPB.Model_VISIBILITY_PRIVATE {
		visibility = modelPB.Model_VISIBILITY_PRIVATE
	}
	bModelConfig, _ := json.Marshal(datamodel.GitHubModelConfiguration{
		Repository: modelConfig.Repository,
		HTMLURL:    "https://github.com/" + modelConfig.Repository,
		Tag:        modelConfig.Tag,
	})
	description := ""
	if req.GetModel().Description != nil {
		description = *req.GetModel().Description
	}

	githubModel := datamodel.Model{
		ID:                 req.GetModel().Id,
		ModelDefinitionUID: modelDefinition.UID,
		Owner:              authUser.Permalink(),
		Visibility:         datamodel.ModelVisibility(visibility),
		State:              datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		Configuration: bModelConfig,
	}

	rdid, _ := uuid.NewV4()
	modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String()) + ""
	if config.Config.Cache.Model.Enabled { // cache model into ~/.cache/instill/models
		modelSrcDir = config.Config.Cache.Model.CacheDir + "/" + fmt.Sprintf("%s_%s", modelConfig.Repository, modelConfig.Tag)
	}

	if config.Config.Server.ItMode.Enabled { // use local model for testing to remove internet connection issue while testing
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mkdir -p %s > /dev/null; cp -rf assets/model-dummy-cls/* %s", modelSrcDir, modelSrcDir))
		if err := cmd.Run(); err != nil {
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), githubModel.ID, modelConfig.Tag)
			span.SetStatus(1, err.Error())
			return &longrunningpb.Operation{}, err
		}
	} else {
		err = utils.GitHubClone(modelSrcDir, modelConfig, false, s.GetRedisClient())
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"GitHub",
				"Clone repository",
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), githubModel.ID, modelConfig.Tag)
			span.SetStatus(1, err.Error())
			return &longrunningpb.Operation{}, st.Err()
		}
	}
	readmeFilePath, ensembleFilePath, err := utils.UpdateModelPath(modelSrcDir, config.Config.TritonServer.ModelStore, authUser.Permalink(), &githubModel)
	if !config.Config.Cache.Model.Enabled {
		_ = os.RemoveAll(modelSrcDir) // remove uploaded temporary files
	}

	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Model folder structure",
			"",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), githubModel.ID, modelConfig.Tag)
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := utils.GetModelMetaFromReadme(readmeFilePath)
		if err != nil || modelMeta.Task == "" {
			st, err := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"README.md file",
				"Could not get meta data from README.md file",
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), githubModel.ID, modelConfig.Tag)
			span.SetStatus(1, st.Err().Error())
			return &longrunningpb.Operation{}, st.Err()
		}
		if val, ok := utils.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
			githubModel.Task = datamodel.ModelTask(val)
		} else {
			if modelMeta.Task != "" {
				st, err := sterr.CreateErrorResourceInfo(
					codes.FailedPrecondition,
					fmt.Sprintf("[handler] create a model error: %s", err.Error()),
					"README.md file",
					"README.md contains unsupported task",
					"",
					err.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), githubModel.ID, modelConfig.Tag)
				span.SetStatus(1, st.Err().Error())
				return &longrunningpb.Operation{}, st.Err()
			} else {
				githubModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
			}
		}
	} else {
		githubModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
	}

	maxBatchSize := 0
	if ensembleFilePath != "" {
		maxBatchSize, err = utils.GetMaxBatchSize(ensembleFilePath)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"GitHub model",
				"Missing ensemble model",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			span.SetStatus(1, st.Err().Error())
			return &longrunningpb.Operation{}, st.Err()
		}
	}

	allowedMaxBatchSize := utils.GetSupportedBatchSize(githubModel.Task)

	if maxBatchSize > allowedMaxBatchSize {
		st, e := sterr.CreateErrorPreconditionFailure(
			"[handler] create a model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "MAX BATCH SIZE LIMITATION",
					Subject:     "Create a model error",
					Description: fmt.Sprintf("The max_batch_size in config.pbtxt exceeded the limitation %v, please try with a smaller max_batch_size", allowedMaxBatchSize),
				},
			})
		if e != nil {
			logger.Error(e.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}

	wfID, err := s.CreateNamespaceModelAsync(ctx, ns, authUser, &githubModel)
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Model service",
			"",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		for _, tag := range githubInfo.Tags {
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, authUser.Permalink(), githubModel.ID, tag.Name)
		}
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(githubModel),
		custom_otel.SetEventResult(&longrunningpb.Operation_Response{
			Response: &anypb.Any{
				Value: []byte(wfID),
			},
		}),
	)))

	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfID),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}, nil
}

func createHuggingFaceModel(s service.Service, ctx context.Context, req CreateNamespaceModelRequestInterface, ns resource.Namespace, authUser *service.AuthUser, modelDefinition *datamodel.ModelDefinition) (*longrunningpb.Operation, error) {

	eventName := "CreateHuggingFaceModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ownerPermalink := authUser.Permalink()

	var modelConfig datamodel.HuggingFaceModelConfiguration
	b, err := req.GetModel().GetConfiguration().MarshalJSON()
	if err != nil {
		span.SetStatus(1, err.Error())
		return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		span.SetStatus(1, err.Error())
		return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if modelConfig.RepoID == "" {
		span.SetStatus(1, "Invalid model ID")
		return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, "Invalid model ID")
	}
	modelConfig.HTMLURL = "https://huggingface.co/" + modelConfig.RepoID
	modelConfig.Tag = "latest"

	visibility := modelPB.Model_VISIBILITY_PRIVATE
	if req.GetModel().Visibility == modelPB.Model_VISIBILITY_PUBLIC {
		visibility = modelPB.Model_VISIBILITY_PUBLIC
	}
	bModelConfig, _ := json.Marshal(modelConfig)
	description := ""
	if req.GetModel().Description != nil {
		description = *req.GetModel().Description
	}
	huggingfaceModel := datamodel.Model{
		ID:                 req.GetModel().Id,
		ModelDefinitionUID: modelDefinition.UID,
		Owner:              ownerPermalink,
		Visibility:         datamodel.ModelVisibility(visibility),
		State:              datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		Configuration: bModelConfig,
	}
	rdid, _ := uuid.NewV4()
	configTmpDir := fmt.Sprintf("/tmp/%s", rdid.String())
	if config.Config.Server.ItMode.Enabled { // use local model for testing to remove internet connection issue while testing
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mkdir -p %s > /dev/null; cp -rf assets/tiny-vit-random/* %s", configTmpDir, configTmpDir))
		if err := cmd.Run(); err != nil {
			_ = os.RemoveAll(configTmpDir)
			span.SetStatus(1, err.Error())
			return &longrunningpb.Operation{}, err
		}
	} else {
		if err := utils.HuggingFaceClone(configTmpDir, modelConfig); err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"Huggingface",
				"Clone model repository",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			_ = os.RemoveAll(configTmpDir)
			span.SetStatus(1, err.Error())
			return &longrunningpb.Operation{}, st.Err()
		}
	}
	rdid, _ = uuid.NewV4()
	modelDir := fmt.Sprintf("/tmp/%s", rdid.String())
	if err := utils.GenerateHuggingFaceModel(configTmpDir, modelDir, req.GetModel().Id); err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Huggingface",
			"Generate HuggingFace model",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
		}
		_ = os.RemoveAll(modelDir)
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}
	_ = os.RemoveAll(configTmpDir)

	readmeFilePath, ensembleFilePath, err := utils.UpdateModelPath(modelDir, config.Config.TritonServer.ModelStore, ownerPermalink, &huggingfaceModel)

	_ = os.RemoveAll(modelDir) // remove uploaded temporary files
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Model folder structure",
			"",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
		}
		utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, huggingfaceModel.ID, "latest")
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := utils.GetModelMetaFromReadme(readmeFilePath)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"README.md file",
				"Could not get meta data from README.md file",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, huggingfaceModel.ID, "latest")
			span.SetStatus(1, st.Err().Error())
			return &longrunningpb.Operation{}, st.Err()
		}
		if modelMeta.Task != "" {

			if val, ok := utils.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
				huggingfaceModel.Task = datamodel.ModelTask(val)
			} else {
				st, err := sterr.CreateErrorResourceInfo(
					codes.FailedPrecondition,
					"[handler] create a model error",
					"README.md file",
					"README.md contains unsupported task",
					"",
					err.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, huggingfaceModel.ID, modelConfig.Tag)
				span.SetStatus(1, st.Err().Error())
				return &longrunningpb.Operation{}, st.Err()
			}
		} else {
			if len(modelMeta.Tags) == 0 {
				huggingfaceModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
			} else { // check in tags also for HuggingFace model card README.md
				huggingfaceModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
				for _, tag := range modelMeta.Tags {
					if val, ok := utils.Tags[strings.ToUpper(tag)]; ok {
						huggingfaceModel.Task = datamodel.ModelTask(val)
						break
					}
				}
			}
		}
	} else {
		huggingfaceModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
	}

	maxBatchSize := 0
	if ensembleFilePath != "" {
		maxBatchSize, err = utils.GetMaxBatchSize(ensembleFilePath)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"HuggingFace model",
				"Missing ensemble model",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			span.SetStatus(1, st.Err().Error())
			return &longrunningpb.Operation{}, st.Err()
		}
	}

	allowedMaxBatchSize := utils.GetSupportedBatchSize(huggingfaceModel.Task)

	if maxBatchSize > allowedMaxBatchSize {
		st, e := sterr.CreateErrorPreconditionFailure(
			"[handler] create a model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "MAX BATCH SIZE LIMITATION",
					Subject:     "Create a model error",
					Description: fmt.Sprintf("The max_batch_size in config.pbtxt exceeded the limitation %v, please try with a smaller max_batch_size", allowedMaxBatchSize),
				},
			})
		if e != nil {
			logger.Error(e.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}

	wfID, err := s.CreateNamespaceModelAsync(ctx, ns, authUser, &huggingfaceModel)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Model service",
			"",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
		}
		utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, huggingfaceModel.ID, "latest")
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(huggingfaceModel),
		custom_otel.SetEventResult(&longrunningpb.Operation_Response{
			Response: &anypb.Any{
				Value: []byte(wfID),
			},
		}),
	)))

	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfID),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}, nil
}

func createArtiVCModel(s service.Service, ctx context.Context, req CreateNamespaceModelRequestInterface, ns resource.Namespace, authUser *service.AuthUser, modelDefinition *datamodel.ModelDefinition) (*longrunningpb.Operation, error) {

	eventName := "CreateArtiVCModel"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := custom_logger.GetZapLogger(ctx)

	ownerPermalink := authUser.Permalink()

	var modelConfig datamodel.ArtiVCModelConfiguration
	b, err := req.GetModel().GetConfiguration().MarshalJSON()
	if err != nil {
		span.SetStatus(1, err.Error())
		return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := json.Unmarshal(b, &modelConfig); err != nil {
		span.SetStatus(1, err.Error())
		return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if modelConfig.URL == "" {
		span.SetStatus(1, "Invalid GitHub URL")
		return &longrunningpb.Operation{}, status.Errorf(codes.InvalidArgument, "Invalid GitHub URL")
	}

	visibility := modelPB.Model_VISIBILITY_PRIVATE
	if req.GetModel().Visibility == modelPB.Model_VISIBILITY_PUBLIC {
		visibility = modelPB.Model_VISIBILITY_PUBLIC
	}
	bModelConfig, _ := json.Marshal(modelConfig)
	description := ""
	if req.GetModel().Description != nil {
		description = *req.GetModel().Description
	}
	artivcModel := datamodel.Model{
		ID:                 req.GetModel().Id,
		ModelDefinitionUID: modelDefinition.UID,
		Owner:              ownerPermalink,
		Visibility:         datamodel.ModelVisibility(visibility),
		State:              datamodel.ModelState(modelPB.Model_STATE_OFFLINE),
		Description: sql.NullString{
			String: description,
			Valid:  true,
		},
		Configuration: bModelConfig,
	}

	rdid, _ := uuid.NewV4()
	modelSrcDir := fmt.Sprintf("/tmp/%v", rdid.String())
	if config.Config.Server.ItMode.Enabled { // use local model for testing to remove internet connection issue while testing
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("mkdir -p %s > /dev/null; cp -rf assets/model-dummy-cls/* %s", modelSrcDir, modelSrcDir))
		if err := cmd.Run(); err != nil {
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, artivcModel.ID, modelConfig.Tag)
			span.SetStatus(1, err.Error())
			return &longrunningpb.Operation{}, err
		}
	} else {
		err = utils.ArtiVCClone(modelSrcDir, modelConfig, false)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"ArtiVC",
				"Clone repository",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			_ = os.RemoveAll(modelSrcDir)
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, artivcModel.ID, modelConfig.Tag)
			span.SetStatus(1, st.Err().Error())
			return &longrunningpb.Operation{}, st.Err()
		}
		utils.AddMissingTritonModelFolder(ctx, modelSrcDir) // large files not pull then need to create triton model folder
	}

	readmeFilePath, ensembleFilePath, err := utils.UpdateModelPath(modelSrcDir, config.Config.TritonServer.ModelStore, ownerPermalink, &artivcModel)
	_ = os.RemoveAll(modelSrcDir) // remove uploaded temporary files
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.FailedPrecondition,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Model folder structure",
			"",
			"",
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, artivcModel.ID, modelConfig.Tag)
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}
	if _, err := os.Stat(readmeFilePath); err == nil {
		modelMeta, err := utils.GetModelMetaFromReadme(readmeFilePath)
		if err != nil || modelMeta.Task == "" {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"README.md file",
				"Could not get meta data from README.md file",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, artivcModel.ID, modelConfig.Tag)
			span.SetStatus(1, st.Err().Error())
			return &longrunningpb.Operation{}, st.Err()
		}
		if val, ok := utils.Tasks[fmt.Sprintf("TASK_%v", strings.ToUpper(modelMeta.Task))]; ok {
			artivcModel.Task = datamodel.ModelTask(val)
		} else {
			if modelMeta.Task != "" {
				st, err := sterr.CreateErrorResourceInfo(
					codes.FailedPrecondition,
					"[handler] create a model error",
					"README.md file",
					"README.md contains unsupported task",
					"",
					err.Error(),
				)
				if err != nil {
					logger.Error(err.Error())
				}
				utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, artivcModel.ID, modelConfig.Tag)
				span.SetStatus(1, st.Err().Error())
				return &longrunningpb.Operation{}, st.Err()
			} else {
				artivcModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
			}
		}
	} else {
		artivcModel.Task = datamodel.ModelTask(commonPB.Task_TASK_UNSPECIFIED)
	}

	maxBatchSize := 0
	if ensembleFilePath != "" {
		maxBatchSize, err = utils.GetMaxBatchSize(ensembleFilePath)
		if err != nil {
			st, e := sterr.CreateErrorResourceInfo(
				codes.FailedPrecondition,
				fmt.Sprintf("[handler] create a model error: %s", err.Error()),
				"ArtiVC model",
				"Missing ensemble model",
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			span.SetStatus(1, st.Err().Error())
			return &longrunningpb.Operation{}, st.Err()
		}
	}

	allowedMaxBatchSize := utils.GetSupportedBatchSize(artivcModel.Task)

	if maxBatchSize > allowedMaxBatchSize {
		st, e := sterr.CreateErrorPreconditionFailure(
			"[handler] create a model",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "MAX BATCH SIZE LIMITATION",
					Subject:     "Create a model error",
					Description: fmt.Sprintf("The max_batch_size in config.pbtxt exceeded the limitation %v, please try with a smaller max_batch_size", allowedMaxBatchSize),
				},
			})

		if e != nil {
			logger.Error(e.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}

	wfID, err := s.CreateNamespaceModelAsync(ctx, ns, authUser, &artivcModel)
	if err != nil {
		st, e := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[handler] create a model error: %s", err.Error()),
			"Model service",
			"",
			"",
			err.Error(),
		)
		if e != nil {
			logger.Error(e.Error())
		}
		utils.RemoveModelRepository(config.Config.TritonServer.ModelStore, ownerPermalink, artivcModel.ID, modelConfig.Tag)
		span.SetStatus(1, st.Err().Error())
		return &longrunningpb.Operation{}, st.Err()
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(artivcModel),
		custom_otel.SetEventResult(&longrunningpb.Operation_Response{
			Response: &anypb.Any{
				Value: []byte(wfID),
			},
		}),
	)))

	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", wfID),
		Done: false,
		Result: &longrunningpb.Operation_Response{
			Response: &anypb.Any{},
		},
	}, nil
}
