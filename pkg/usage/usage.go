package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/x/repo"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/base/usage/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
	usageClient "github.com/instill-ai/usage-client/client"
	usageReporter "github.com/instill-ai/usage-client/reporter"
)

// Usage interface
type Usage interface {
	RetrieveUsageData() interface{}
	StartReporter(ctx context.Context)
	TriggerSingleReporter(ctx context.Context)
}

type usage struct {
	repository               repository.Repository
	mgmtPrivateServiceClient mgmtPB.MgmtPrivateServiceClient
	redisClient              *redis.Client
	reporter                 usageReporter.Reporter
	version                  string
}

// NewUsage initiates a usage instance
func NewUsage(ctx context.Context, r repository.Repository, u mgmtPB.MgmtPrivateServiceClient, rc *redis.Client, usc usagePB.UsageServiceClient) Usage {
	logger, _ := logger.GetZapLogger(ctx)

	version, err := repo.ReadReleaseManifest("release-please/manifest.json")
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	reporter, err := usageClient.InitReporter(ctx, usc, usagePB.Session_SERVICE_MODEL, config.Config.Server.Edition, version)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	return &usage{
		repository:               r,
		mgmtPrivateServiceClient: u,
		redisClient:              rc,
		reporter:                 reporter,
		version:                  version,
	}
}

func (u *usage) RetrieveUsageData() interface{} {

	ctx := context.Background()
	logger, _ := logger.GetZapLogger(ctx)

	logger.Debug("Retrieve usage data...")

	pbModelUsageData := []*usagePB.ModelUsageData_UserUsageData{}

	// Roll over all users and update the metrics with the cached uuid
	userPageToken := ""
	userPageSizeMax := int64(repository.MaxPageSize)
	for {
		userResp, err := u.mgmtPrivateServiceClient.ListUsersAdmin(ctx, &mgmtPB.ListUsersAdminRequest{
			PageSize:  &userPageSizeMax,
			PageToken: &userPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUser] %s", err))
			break
		}

		// Roll all model resources on a user
		for _, user := range userResp.Users {
			modelPageToken := ""

			modelOnlineStateNum := int64(0)  // Number of models that have at least one 'online' instance
			modelOfflineStateNum := int64(0) // Number of models that have no 'online' instances
			modelDefinitionIds := []string{} // Definition IDs of the model instances. Element in the list should not be duplicated.
			tasks := []modelPB.Model_Task{}
			testImageNum := int64(0)         // Number of processed images via model instance testing operations
			var mDef = make(map[string]bool) // use for creating unique model definition list
			for {
				dbModels, modelNextPageToken, _, err := u.repository.ListModels(fmt.Sprintf("users/%s", user.GetUid()), modelPB.View_VIEW_BASIC, repository.MaxPageSize, modelPageToken)
				if err != nil {
					logger.Error(fmt.Sprintf("%s", err))
				}

				for _, model := range dbModels {
					isModelOnline := false
					modelDef, err := u.repository.GetModelDefinitionByUID(model.ModelDefinitionUid)
					if err != nil {
						logger.Error(fmt.Sprintf("%s", err))
					} else {
						if !mDef[modelDef.ID] {
							mDef[modelDef.ID] = true
							modelDefinitionIds = append(modelDefinitionIds, modelDef.ID)
						}
					}

					if model.State == datamodel.ModelState(modelPB.Model_STATE_ONLINE) {
						isModelOnline = true
					}

					if isModelOnline {
						modelOnlineStateNum++
					} else {
						modelOfflineStateNum++
					}

					tasks = append(tasks, modelPB.Model_Task(model.Task))
				}

				if modelNextPageToken == "" {
					break
				} else {
					modelPageToken = modelNextPageToken
				}
			}

			testImageNum, err := u.redisClient.Get(ctx, fmt.Sprintf("user:%s:test.num", user.GetUid())).Int64()
			if err == redis.Nil {
				testImageNum = 0
			} else if err != nil {
				logger.Error(fmt.Sprintf("%s", err))
			}

			pbModelUsageData = append(pbModelUsageData, &usagePB.ModelUsageData_UserUsageData{
				UserUid:              user.GetUid(),
				ModelOnlineStateNum:  modelOnlineStateNum,
				ModelOfflineStateNum: modelOfflineStateNum,
				ModelDefinitionIds:   modelDefinitionIds,
				Tasks:                tasks,
				TestNum:              testImageNum,
			})
		}

		if userResp.NextPageToken == "" {
			break
		} else {
			userPageToken = userResp.NextPageToken
		}
	}

	logger.Debug("Send retrieved usage data...")
	return &usagePB.SessionReport_ModelUsageData{
		ModelUsageData: &usagePB.ModelUsageData{
			Usages: pbModelUsageData,
		},
	}
}

func (u *usage) StartReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}

	logger, _ := logger.GetZapLogger(ctx)
	go func() {
		time.Sleep(5 * time.Second)
		err := usageClient.StartReporter(ctx, u.reporter, usagePB.Session_SERVICE_MODEL, config.Config.Server.Edition, u.version, u.RetrieveUsageData)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to start reporter: %v\n", err))
		}
	}()
}

func (u *usage) TriggerSingleReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}
	logger, _ := logger.GetZapLogger(ctx)
	err := usageClient.SingleReporter(ctx, u.reporter, usagePB.Session_SERVICE_MODEL, config.Config.Server.Edition, u.version, u.RetrieveUsageData())
	if err != nil {
		logger.Fatal(err.Error())
	}
}
