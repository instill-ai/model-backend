package ray

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
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

func GenerateHardwareConfig(modelID string) int {
	// TODO: proper support for multi-gpu
	// match suffix `-{int}g`
	re := regexp.MustCompile(`-(\d+)g$`)

	matches := re.FindStringSubmatch(modelID)
	if len(matches) == 2 {
		numOfGPU, err := strconv.Atoi(matches[1])
		if err != nil {
			return 1
		}
		return numOfGPU
	}

	return 1
}

func GetApplicationMetadaValue(modelName string, version string) (applicationMetadataValue string, err error) {
	nameParts := strings.Split(modelName, "/") // {owner_type}/{owner_uid}/{model_id}

	if len(nameParts) != 3 {
		return "", errors.New("modelName format error")
	}

	nameParts = append(nameParts, version)

	return strings.Join(nameParts, "_"), nil
}
