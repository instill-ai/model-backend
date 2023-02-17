package triton

type DetectionOutput struct {
	Boxes  [][][]float32
	Labels [][]string
}

type OcrOutput struct {
	Boxes  [][][]float32
	Texts  [][]string
	Scores [][]float32
}

type KeypointOutput struct {
	Keypoints [][][][]float32
	Boxes     [][][]float32
	Scores    [][]float32
}

type BatchUnspecifiedTaskOutputs struct {
	Name              string
	Shape             []int64
	DataType          string
	SerializedOutputs []interface{} // batching output
}

type SingleOutputUnspecifiedTaskOutput struct {
	Name     string
	Shape    []int64
	DataType string
	Data     interface{} // batching output
}

type UnspecifiedTaskOutput struct {
	RawOutput []SingleOutputUnspecifiedTaskOutput
}

type InstanceSegmentationOutput struct {
	Rles   [][]string
	Boxes  [][][]float32
	Scores [][]float32
	Labels [][]string
}

type SemanticSegmentationOutput struct {
	Rles       [][]string
	Categories [][]string
}

type TextToImageOutput struct {
	Images [][]string
}

type TextGenerationOutput struct {
	Text []string
}
