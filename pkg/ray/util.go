package ray

import (
	"errors"
	"fmt"
	"strings"
)

func GenerateScalingConfig(modelID string) []string {
	if strings.HasPrefix(modelID, DummyModelPrefix) {
		return []string{
			fmt.Sprintf("-e %s=%v", EnvIsTestModel, "true"),
		}
	}

	return []string{}
}

func GetApplicationMetadaValue(modelName string, version string) (applicationMetadataValue string, err error) {
	nameParts := strings.Split(modelName, "/") // {owner_type}/{owner_uid}/{model_id}

	if len(nameParts) != 3 {
		return "", errors.New("modelName format error")
	}

	nameParts = append(nameParts, version)

	return strings.Join(nameParts, "_"), nil
}
