package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/utils"
)

func (w *worker) writeNewDataPoint(ctx context.Context, data *utils.UsageMetricData) error {

	if config.Config.Server.Usage.Enabled {

		bData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		w.redisClient.RPush(ctx, fmt.Sprintf("owner:%s:model.trigger_data", data.OwnerUID), string(bData))
	}

	return nil
}

func (w *worker) writePrediction(ctx context.Context, pred *datamodel.ModelPrediction) error {
	return w.repository.CreateModelPrediction(ctx, pred)
}
