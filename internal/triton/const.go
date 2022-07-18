package triton

type DetectionOutput struct {
	Boxes  [][][]float32
	Labels [][]string
}

type KeypointOutput struct {
	Keypoints [][][]float32
	Scores    []float32
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
