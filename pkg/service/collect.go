package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/utils"
)

func (s *service) WriteNewDataPoint(ctx context.Context, data *utils.UsageMetricData) error {
	if config.Config.Server.Usage.Enabled {

		bData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		s.redisClient.RPush(ctx, fmt.Sprintf("owner:%s:model.trigger_data", data.OwnerUID), string(bData))
	}

	return nil
}
