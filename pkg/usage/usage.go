package usage

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v9"

	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
)

// Usage interface
type Usage interface {
	RetrieveUsageData() interface{}
}

type usage struct {
	repository        repository.Repository
	userServiceClient mgmtPB.UserServiceClient
	redisClient       *redis.Client
}

// NewUsage initiates a usage instance
func NewUsage(r repository.Repository, mu mgmtPB.UserServiceClient, rc *redis.Client) Usage {
	return &usage{
		repository:        r,
		userServiceClient: mu,
		redisClient:       rc,
	}
}

func (u *usage) RetrieveUsageData() interface{} {

	logger, _ := logger.GetZapLogger()
	ctx := context.Background()

	logger.Debug("Retrieve usage data...")

	pbModelUsageData := []*usagePB.ModelUsageData_UserUsageData{}

	// Roll over all users and update the metrics with the cached uuid
	userPageToken := ""
	userPageSizeMax := int64(repository.MaxPageSize)
	for {
		userResp, err := u.userServiceClient.ListUser(ctx, &mgmtPB.ListUserRequest{
			PageSize:  &userPageSizeMax,
			PageToken: &userPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUser] %s", err))
		}

		// Roll all model resources on a user
		for _, user := range userResp.Users {
			modelPageToken := ""

			modelOnlineStateNum := int64(0)     // Number of models that have at least one 'online' instance
			modelOfflineStateNum := int64(0)    // Number of models that have no 'online' instances
			instanceOnlineStateNum := int64(0)  // Number of model instances with 'online' state
			instanceOfflineStateNum := int64(0) // Number of model instances with 'offline' state
			modelDefinitionIds := []string{}    // Definition IDs of the model instances. Element in the list should not be duplicated.
			tasks := []modelPB.ModelInstance_Task{}
			testImageNum := int64(0)                               // Number of processed images via model instance testing operations
			var mTask = make(map[datamodel.ModelInstanceTask]bool) // use for creating unique task list
			var mDef = make(map[string]bool)                       // use for creating unique model definition list
			for {
				dbModels, modelNextPageToken, _, err := u.repository.ListModel(fmt.Sprintf("users/%s", user.GetUid()), modelPB.View_VIEW_BASIC, repository.MaxPageSize, modelPageToken)
				if err != nil {
					logger.Error(fmt.Sprintf("%s", err))
				}

				instancePageToken := ""
				for _, model := range dbModels {
					isModelOnline := false
					modelDef, err := u.repository.GetModelDefinitionByUid(model.ModelDefinitionUid)
					if err != nil {
						logger.Error(fmt.Sprintf("%s", err))
					} else {
						if !mDef[modelDef.ID] {
							mDef[modelDef.ID] = true
							modelDefinitionIds = append(modelDefinitionIds, modelDef.ID)
						}
					}

					for {
						dbInstances, instanceNextPageToken, _, err := u.repository.ListModelInstance(model.UID, modelPB.View_VIEW_BASIC, repository.MaxPageSize, instancePageToken)
						if err != nil {
							logger.Error(fmt.Sprintf("%s", err))
						}

						for _, instance := range dbInstances {
							if instance.State == datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_ONLINE) {
								instanceOnlineStateNum++
								isModelOnline = true
							}
							if instance.State == datamodel.ModelInstanceState(modelPB.ModelInstance_STATE_OFFLINE) {
								instanceOfflineStateNum++
							}

							if !mTask[instance.Task] {
								mTask[instance.Task] = true
								tasks = append(tasks, modelPB.ModelInstance_Task(instance.Task))
							}
						}

						if instanceNextPageToken == "" {
							break
						} else {
							instancePageToken = instanceNextPageToken
						}
					}
					if isModelOnline {
						modelOnlineStateNum++
					} else {
						modelOfflineStateNum++
					}
				}

				if modelNextPageToken == "" {
					break
				} else {
					modelPageToken = modelNextPageToken
				}
			}

			testImageNum, err := u.redisClient.Get(ctx, fmt.Sprintf("user:%s:test.image.num", user.GetUid())).Int64()
			if err == redis.Nil {
				testImageNum = 0
			} else if err != nil {
				logger.Error(fmt.Sprintf("%s", err))
			}

			pbModelUsageData = append(pbModelUsageData, &usagePB.ModelUsageData_UserUsageData{
				UserUid:                 user.GetUid(),
				ModelOnlineStateNum:     modelOnlineStateNum,
				ModelOfflineStateNum:    modelOfflineStateNum,
				InstanceOnlineStateNum:  instanceOnlineStateNum,
				InstanceOfflineStateNum: instanceOfflineStateNum,
				ModelDefinitionIds:      modelDefinitionIds,
				Tasks:                   tasks,
				TestImageNum:            testImageNum,
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
