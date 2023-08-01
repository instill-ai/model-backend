package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/repo"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/base/usage/v1alpha"
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

			triggerDataList := []*usagePB.ModelUsageData_UserUsageData_ModelTriggerData{}

			triggerCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:model.trigger_data", user.GetUid())).Val() // O(1)

			if triggerCount != 0 {
				for i := int64(0); i < triggerCount; i++ {

					strData := u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:model.trigger_data", user.GetUid())).Val()

					triggerData := &utils.UsageMetricData{}
					if err := json.Unmarshal([]byte(strData), triggerData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					triggerTime, _ := time.Parse(time.RFC3339Nano, triggerData.TriggerTime)

					triggerDataList = append(
						triggerDataList,
						&usagePB.ModelUsageData_UserUsageData_ModelTriggerData{
							TriggerUid:         triggerData.TriggerUID,
							TriggerTime:        timestamppb.New(triggerTime),
							ModelUid:           triggerData.ModelUID,
							ModelDefinitionUid: triggerData.ModelDefinitionUID,
							ModelTask:          triggerData.ModelTask,
							Status:             triggerData.Status,
						},
					)
				}
			}

			pbModelUsageData = append(pbModelUsageData, &usagePB.ModelUsageData_UserUsageData{
				UserUid:          user.GetUid(),
				ModelTriggerData: triggerDataList,
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
