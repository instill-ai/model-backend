package triton

type DetectionOutput struct {
	Boxes  [][][]float32
	Labels [][]string
}

type KeypointOutput struct {
	Keypoints [][][]float32
	Scores    []float32
}
