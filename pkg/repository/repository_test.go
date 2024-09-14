//go:build dbtest
// +build dbtest

package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/repository"

	database "github.com/instill-ai/model-backend/pkg/db"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	if err := config.Init("../../config/config.local.yaml"); err != nil {
		panic(err)
	}

	db = database.GetSharedConnection()
	db.Logger = logger.Default.LogMode(logger.Info)
	exitCode := m.Run()
	database.Close(db)

	os.Exit(exitCode)
}

func TestRepository(t *testing.T) {
	c := qt.New(t)

	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	rc := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	tx := db.Begin()
	c.Cleanup(func() { tx.Rollback() })

	repo := repository.NewRepository(tx, rc)
	recordUUID, _ := uuid.NewV4()
	triggerUUID, _ := uuid.NewV4()
	userUUID, _ := uuid.NewV4()

	defs, _, _, err := repo.ListModelDefinitions(modelpb.View_VIEW_FULL, 100, "")
	require.NoError(t, err)
	require.NotEmpty(t, defs)

	ctx := context.Background()
	err = repo.CreateNamespaceModel(ctx, recordUUID.String(), &datamodel.Model{
		ID:                 recordUUID.String(),
		ModelDefinitionUID: defs[0].UID,
		Visibility:         datamodel.ModelVisibility(modelpb.Model_VISIBILITY_PRIVATE),
		Owner:              recordUUID.String(),
		Task:               datamodel.ModelTask(commonpb.Task_TASK_CLASSIFICATION),
	})
	require.NoError(t, err)
	visibility := modelpb.Model_VISIBILITY_PRIVATE
	models, _, _, err := repo.ListNamespaceModels(ctx, recordUUID.String(), 10, "",
		true, filtering.Filter{}, nil, false, ordering.OrderBy{}, &visibility)
	require.NoError(t, err)
	require.NotEmpty(t, models)
	require.Len(t, models, 1)
	require.Equal(t, defs[0].UID, models[0].ModelDefinitionUID)
	require.Equal(t, recordUUID.String(), models[0].Owner)

	res, err := repo.CreateModelRun(ctx, &datamodel.ModelRun{
		BaseStaticHardDelete: datamodel.BaseStaticHardDelete{
			UID: triggerUUID,
		},
		ModelUID:     models[0].UID,
		ModelVersion: "latest",
		Status:       datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
		Source:       datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API),
		RequesterUID: userUUID,
	})
	require.NoError(t, err)

	runLog := &datamodel.ModelRun{}
	runLog.UID = res.UID
	require.NoError(t, tx.First(runLog).Error)
	require.Equal(t, models[0].UID, runLog.ModelUID)
	require.Equal(t, triggerUUID, runLog.UID)
	require.Equal(t, userUUID, runLog.RequesterUID)

	time.Sleep(2 * time.Second)
	now := time.Now()
	runLog.EndTime = null.TimeFrom(now)
	duration := now.Sub(runLog.CreateTime)
	runLog.TotalDuration = null.IntFrom(duration.Milliseconds())
	err = repo.UpdateModelRun(ctx, runLog)
	require.NoError(t, err)

	runLog = &datamodel.ModelRun{}
	runLog.UID = res.UID
	require.NoError(t, tx.First(runLog).Error)
	require.True(t, runLog.EndTime.Valid)
	require.True(t, runLog.TotalDuration.Valid)
	require.GreaterOrEqual(t, runLog.TotalDuration.Int64, int64(2000))

}
