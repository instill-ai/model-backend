package ray

import (
	"errors"
	"regexp"
	"strings"
)

// GenerateHardwareConfig generates the hardware config for the model
// It is used to generate the hardware config for the model
// from {model_id}-{num_of_gpu}g to {num_of_gpu}
func GenerateHardwareConfig(modelID string) string {
	// TODO: refactor this
	// match suffix `-{int}g`
	re := regexp.MustCompile(`-(\d+)g$`)

	matches := re.FindStringSubmatch(modelID)
	if len(matches) == 2 {
		return matches[1]
	}

	return "0"
}

// GetApplicationMetadataValue gets the application metadata value
// It is used to get the application metadata value to name the Ray application
// from {owner_type}/{owner_uid}/{model_id} to {owner_type}_{owner_uid}_{model_id}_{version}
func GetApplicationMetadataValue(modelName string, version string) (applicationMetadataValue string, err error) {
	nameParts := strings.Split(modelName, "/") // {owner_type}/{owner_uid}/{model_id}

	if len(nameParts) != 3 {
		return "", errors.New("modelName format error")
	}

	nameParts = append(nameParts, version)

	return strings.Join(nameParts, "_"), nil
}

// IsDummyModel checks if the model is a dummy model
// Dummy model is a model that is used for testing and development
// It is not a real model and does not have a real owner
// It is used to test the model deployment and scaling
func IsDummyModel(modelName string) bool {
	nameParts := strings.Split(modelName, "/") // {owner_type}/{owner_uid}/{model_id}

	if len(nameParts) != 3 {
		return false
	}

	return strings.HasPrefix(nameParts[2], DummyModelPrefix)
}
