package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/x/repo"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/base/usage/v1alpha"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
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

			triggerCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:trigger.trigger_uid", user.GetUid())).Val()

			if triggerCount != 0 {
				for i := int64(0); i < triggerCount; i++ {
					timeStr, _ := time.Parse(time.RFC3339Nano, u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:trigger.trigger_time", user.GetUid())).Val())
					triggerData := &usagePB.ModelUsageData_UserUsageData_ModelTriggerData{}
					triggerData.ModelUid = u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:trigger.model_uid", user.GetUid())).Val()
					triggerData.TriggerUid = u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:trigger.trigger_uid", user.GetUid())).Val()
					triggerData.TriggerTime = timestamppb.New(timeStr)
					triggerData.ModelDefinitionUid = u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:trigger.model_definition_uid", user.GetUid())).Val()
					triggerData.ModelTask = commonPB.Task(commonPB.Task_value[u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:trigger.model_task", user.GetUid())).Val()])
					triggerDataList = append(triggerDataList, triggerData)
				}
			}

			u.redisClient.Unlink(
				ctx,
				fmt.Sprintf("user:%s:trigger.trigger_uid", user.GetUid()),
				fmt.Sprintf("user:%s:trigger.trigger_time", user.GetUid()),
				fmt.Sprintf("user:%s:trigger.model_uid", user.GetUid()),
				fmt.Sprintf("user:%s:trigger.model_definition_uid", user.GetUid()),
				fmt.Sprintf("user:%s:trigger.model_task", user.GetUid()),
				fmt.Sprintf("user:%s:trigger.status", user.GetUid()),
			)

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

	fmt.Println("\n========================================================================")
	fmt.Println(pbModelUsageData)
	fmt.Println("========================================================================")
	fmt.Println()

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
