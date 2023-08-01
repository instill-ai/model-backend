package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/instill-ai/model-backend/pkg/utils"
)

func (s *service) WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData) error {

	bData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	s.redisClient.RPush(ctx, fmt.Sprintf("user:%s:model.trigger_data", data.OwnerUID), string(bData))

	return nil
}
