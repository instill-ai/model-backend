package utils

const MaxBatchSize int = 32

const DefaultPageSize = 10

const (
	ToImageSteps    = int32(10)
	ToImageCFGScale = float32(7)
	ToImageSeed     = int32(1024)
	ToImageSamples  = int32(1)
)

const (
	TextGenerationMaxNewTokens = int32(100)
	TextGenerationTemperature  = float32(0.7)
	TextGenerationTopK         = int32(10)
	TextGenerationSeed         = int32(0)
)

type Flag string

const Testing Flag = "TESTING"
