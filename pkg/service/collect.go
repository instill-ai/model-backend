package service

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/model-backend/pkg/util"
)

func (s *service) WriteNewDataPoint(ctx context.Context, data util.UsageMetricData) {
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.trigger_time", data.OwnerUID), data.TriggerTime.Format(time.RFC3339Nano))
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.trigger_uid", data.OwnerUID), data.TriggerUID)
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.model_uid", data.OwnerUID), data.ModelUID)
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.model_definition_uid", data.OwnerUID), data.ModelDefinitionUID)
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.model_task", data.OwnerUID), data.ModelTask)
	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:trigger.status", data.OwnerUID), data.Status)
}
