package ray

type InferInput any

type TextToImageInput struct {
	Prompt      string
	PromptImage string
	Steps       int32
	CfgScale    float32
	Seed        int32
	Samples     int32
	ExtraParams string
}

type ImageToImageInput struct {
	Prompt      string
	PromptImage string
	Steps       int32
	CfgScale    float32
	Seed        int32
	Samples     int32
	ExtraParams string
}

type TextGenerationInput struct {
	Prompt        string
	PromptImages  string
	ChatHistory   string
	SystemMessage string
	MaxNewTokens  int32
	Temperature   float32
	TopK          int32
	Seed          int32
	ExtraParams   string
}

type TextGenerationChatInput struct {
	Prompt        string
	PromptImages  string
	ChatHistory   string
	SystemMessage string
	MaxNewTokens  int32
	Temperature   float32
	TopK          int32
	Seed          int32
	ExtraParams   string
}

type VisualQuestionAnsweringInput struct {
	Prompt        string
	PromptImages  string
	ChatHistory   string
	SystemMessage string
	MaxNewTokens  int32
	Temperature   float32
	TopK          int32
	Seed          int32
	ExtraParams   string
}

type ImageInput struct {
	ImgURL    string
	ImgBase64 string
}

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
	SerializedOutputs []any // batching output
}

type SingleOutputUnspecifiedTaskOutput struct {
	Name     string
	Shape    []int64
	DataType string
	Data     any // batching output
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

type ImageToImageOutput struct {
	Images [][]string
}

type TextGenerationOutput struct {
	Text []string
}

type TextGenerationChatOutput struct {
	Text []string
}

type VisualQuestionAnsweringOutput struct {
	Text []string
}

type ModelDeploymentConfig struct {
	Applications []Application `yaml:"applications" json:"applications"`
}

type ApplicationWithAction struct {
	Application Application
	IsDeploy    bool
}

type Application struct {
	Name        string     `yaml:"name" json:"name"`
	ImportPath  string     `yaml:"import_path" json:"import_path"`
	RoutePrefix string     `yaml:"route_prefix" json:"route_prefix"`
	RuntimeEnv  RuntimeEnv `yaml:"runtime_env" json:"runtime_env"`
}

type RuntimeEnv struct {
	Container Container `yaml:"container" json:"container"`
}

type Container struct {
	Image      string   `yaml:"image" json:"image"`
	RunOptions []string `yaml:"run_options" json:"run_options"`
}
