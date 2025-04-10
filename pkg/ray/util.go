package ray

import (
	"errors"
	"fmt"
	"regexp"
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

func GenerateHardwareConfig(modelID string) string {
	// TODO: proper support for multi-gpu
	// match suffix `-{int}g`
	re := regexp.MustCompile(`-(\d+)g$`)

	matches := re.FindStringSubmatch(modelID)
	if len(matches) == 2 {
		return matches[1]
	}

	return "0"
}

func GetApplicationMetadataValue(modelName string, version string) (applicationMetadataValue string, err error) {
	nameParts := strings.Split(modelName, "/") // {owner_type}/{owner_uid}/{model_id}

	if len(nameParts) != 3 {
		return "", errors.New("modelName format error")
	}

	nameParts = append(nameParts, version)

	return strings.Join(nameParts, "_"), nil
}
