package ray

import (
	"fmt"
	"strings"

	"github.com/instill-ai/model-backend/pkg/ray/rayserver"
)

func GetOutputFromInferResponse(name string, response *rayserver.ModelInferResponse) (*rayserver.InferTensor, []byte, error) {
	for idx, output := range response.Outputs {
		if output.Name == name {
			if len(response.RawOutputContents) > 0 {
				return output, response.RawOutputContents[idx], nil
			} else {
				return output, nil, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("unable to find output named %v", name)
}

func GetApplicationMetadaValue(inferenceModelName string) (applicationMetadataValue string, err error) {
	nameParts := strings.Split(inferenceModelName, "/")

	if len(nameParts) != 2 {
		return "", fmt.Errorf("inferenceModelName format error")
	}

	applicationNameParts := strings.Split(nameParts[1], "#")

	if len(applicationNameParts) != 4 {
		return "", fmt.Errorf("inferenceModelName format error")
	}

	return strings.Join(applicationNameParts[:2], "_"), nil
}
