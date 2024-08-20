package utils

import (
	"google.golang.org/protobuf/encoding/protojson"
)

const MaxBatchSize int = 32

const DefaultGCPServiceAccountFile = "https://artifacts.instill.tech/default-service-account.json"

var MarshalOptions protojson.MarshalOptions = protojson.MarshalOptions{
	EmitUnpopulated: true,
	UseEnumNumbers:  false,
}

var UnmarshalOptions protojson.UnmarshalOptions = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}

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
