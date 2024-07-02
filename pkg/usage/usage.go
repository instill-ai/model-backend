package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	usagepb "github.com/instill-ai/protogen-go/core/usage/v1beta"
	usageClient "github.com/instill-ai/usage-client/client"
	usageReporter "github.com/instill-ai/usage-client/reporter"
	"github.com/instill-ai/x/repo"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/config"
	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/utils"
)

// Usage interface
type Usage interface {
	RetrieveUsageData() any
	StartReporter(ctx context.Context)
	TriggerSingleReporter(ctx context.Context)
}

type usage struct {
	repository               repository.Repository
	mgmtPrivateServiceClient mgmtpb.MgmtPrivateServiceClient
	redisClient              *redis.Client
	reporter                 usageReporter.Reporter
	version                  string
	userUID                  string
}

// NewUsage initiates a usage instance
func NewUsage(ctx context.Context, r repository.Repository, u mgmtpb.MgmtPrivateServiceClient, rc *redis.Client, usc usagepb.UsageServiceClient, userUID string) Usage {
	logger, _ := custom_logger.GetZapLogger(ctx)

	version, err := repo.ReadReleaseManifest("release-please/manifest.json")
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	reporter, err := usageClient.InitReporter(ctx, usc, usagepb.Session_SERVICE_MODEL, config.Config.Server.Edition, version, userUID)
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
		userUID:                  userUID,
	}
}

func (u *usage) RetrieveUsageData() any {

	ctx := context.Background()
	logger, _ := custom_logger.GetZapLogger(ctx)

	logger.Debug("Retrieve usage data...")

	pbModelUsageData := []*usagepb.ModelUsageData_UserUsageData{}

	// Roll over all users and update the metrics with the cached uuid
	userPageToken := ""
	pageSizeMax := int32(repository.MaxPageSize)
	for {
		userResp, err := u.mgmtPrivateServiceClient.ListUsersAdmin(ctx, &mgmtpb.ListUsersAdminRequest{
			PageSize:  &pageSizeMax,
			PageToken: &userPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUser] %s", err))
			break
		}

		// Roll all model resources on a user
		for _, user := range userResp.GetUsers() {

			triggerDataList := []*usagepb.ModelUsageData_UserUsageData_ModelTriggerData{}

			triggerCount := u.redisClient.LLen(ctx, fmt.Sprintf("owner:%s:model.trigger_data", user.GetUid())).Val() // O(1)

			if triggerCount != 0 {
				for i := int64(0); i < triggerCount; i++ {

					strData := u.redisClient.LPop(ctx, fmt.Sprintf("owner:%s:model.trigger_data", user.GetUid())).Val()

					triggerData := &utils.UsageMetricData{}
					if err := json.Unmarshal([]byte(strData), triggerData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					triggerTime, _ := time.Parse(time.RFC3339Nano, triggerData.TriggerTime)

					triggerDataList = append(
						triggerDataList,
						&usagepb.ModelUsageData_UserUsageData_ModelTriggerData{
							TriggerUid:         triggerData.TriggerUID,
							TriggerTime:        timestamppb.New(triggerTime),
							ModelUid:           triggerData.ModelUID,
							ModelDefinitionUid: triggerData.ModelDefinitionUID,
							ModelTask:          triggerData.ModelTask,
							Status:             triggerData.Status,
							UserUid:            triggerData.UserUID,
							UserType:           triggerData.UserType,
						},
					)
				}
				pbModelUsageData = append(pbModelUsageData, &usagepb.ModelUsageData_UserUsageData{
					OwnerUid:         user.GetUid(),
					OwnerType:        mgmtpb.OwnerType_OWNER_TYPE_USER,
					ModelTriggerData: triggerDataList,
				})
			}
		}

		if userResp.NextPageToken == "" {
			break
		} else {
			userPageToken = userResp.NextPageToken
		}
	}

	// Roll over all orgs and update the metrics with the cached uuid
	orgPageToken := ""
	for {
		orgResp, err := u.mgmtPrivateServiceClient.ListOrganizationsAdmin(ctx, &mgmtpb.ListOrganizationsAdminRequest{
			PageSize:  &pageSizeMax,
			PageToken: &orgPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListOrganizationsAdmin] %s", err))
			break
		}

		// Roll all model resources on an org
		for _, org := range orgResp.GetOrganizations() {

			triggerDataList := []*usagepb.ModelUsageData_UserUsageData_ModelTriggerData{}

			triggerCount := u.redisClient.LLen(ctx, fmt.Sprintf("owner:%s:model.trigger_data", org.GetUid())).Val() // O(1)

			if triggerCount != 0 {
				for i := int64(0); i < triggerCount; i++ {

					strData := u.redisClient.LPop(ctx, fmt.Sprintf("owner:%s:model.trigger_data", org.GetUid())).Val()

					triggerData := &utils.UsageMetricData{}
					if err := json.Unmarshal([]byte(strData), triggerData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					triggerTime, _ := time.Parse(time.RFC3339Nano, triggerData.TriggerTime)

					triggerDataList = append(
						triggerDataList,
						&usagepb.ModelUsageData_UserUsageData_ModelTriggerData{
							TriggerUid:         triggerData.TriggerUID,
							TriggerTime:        timestamppb.New(triggerTime),
							ModelUid:           triggerData.ModelUID,
							ModelDefinitionUid: triggerData.ModelDefinitionUID,
							ModelTask:          triggerData.ModelTask,
							Status:             triggerData.Status,
							UserUid:            triggerData.UserUID,
							UserType:           triggerData.UserType,
						},
					)
				}
				pbModelUsageData = append(pbModelUsageData, &usagepb.ModelUsageData_UserUsageData{
					OwnerUid:         org.GetUid(),
					OwnerType:        mgmtpb.OwnerType_OWNER_TYPE_ORGANIZATION,
					ModelTriggerData: triggerDataList,
				})
			}
		}

		if orgResp.NextPageToken == "" {
			break
		} else {
			orgPageToken = orgResp.NextPageToken
		}
	}

	logger.Debug("Send retrieved usage data...")
	return &usagepb.SessionReport_ModelUsageData{
		ModelUsageData: &usagepb.ModelUsageData{
			Usages: pbModelUsageData,
		},
	}
}

func (u *usage) StartReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}

	logger, _ := custom_logger.GetZapLogger(ctx)

	go func() {
		time.Sleep(5 * time.Second)
		err := usageClient.StartReporter(ctx, u.reporter, usagepb.Session_SERVICE_MODEL, config.Config.Server.Edition, u.version, u.userUID, u.RetrieveUsageData)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to start reporter: %v\n", err))
		}
	}()
}

func (u *usage) TriggerSingleReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}

	logger, _ := custom_logger.GetZapLogger(ctx)

	err := usageClient.SingleReporter(ctx, u.reporter, usagepb.Session_SERVICE_MODEL, config.Config.Server.Edition, u.version, u.userUID, u.RetrieveUsageData())
	if err != nil {
		logger.Fatal(err.Error())
	}
}

type ModelUsageHandlerParams struct {
	UserUID, OwnerUID, ModelUID           uuid.UUID
	OwnerType                             resource.NamespaceType
	CreditAmount                          int
	ModelVersion, ModelTriggerID, ModelID string
}

type ModelUsageHandler interface {
	Check(ctx context.Context, usageHandlerParams *ModelUsageHandlerParams) error
	Collect(ctx context.Context, usageHandlerParams *ModelUsageHandlerParams) error
}

type noopModelUsageHandler struct{}

func (h *noopModelUsageHandler) Check(ctx context.Context, usageHandlerParams *ModelUsageHandlerParams) error {
	return nil
}

func (h *noopModelUsageHandler) Collect(ctx context.Context, usageHandlerParams *ModelUsageHandlerParams) error {
	return nil
}

// NewNoopModelUsageHandler is a no-op usage handler initializer.
func NewNoopModelUsageHandler() ModelUsageHandler {
	return new(noopModelUsageHandler)
}
