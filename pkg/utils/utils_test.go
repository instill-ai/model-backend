package utils

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetModelMetaFromReadme_Normal(t *testing.T) {
	txt := `---
Task: Detection
Tags:
  - Detection
  - YoloV4
---
# Body
Hello, it is my body
`
	testFilePath := fmt.Sprintf("/tmp/%v", rand.Int())
	_ = os.WriteFile(testFilePath, []byte(txt), 0644)
	meta, err := GetModelMetaFromReadme(testFilePath)

	assert.Equal(t, err, nil)
	assert.Equal(t, meta.Task, "Detection")
	assert.Equal(t, len(meta.Tags), 2)

	_ = os.Remove(testFilePath)
}

func TestGetModelMetaFromReadme_TaskIsNil(t *testing.T) {
	txt := `---
Tags:
  - Detection
  - YoloV4
---
# Body
Hello, it is my body
`
	testFilePath := fmt.Sprintf("/tmp/%v", rand.Int())
	_ = os.WriteFile(testFilePath, []byte(txt), 0644)
	meta, err := GetModelMetaFromReadme(testFilePath)
	assert.Equal(t, err, nil)
	assert.Equal(t, meta.Task, "")
	assert.Equal(t, len(meta.Tags), 2)

	_ = os.Remove(testFilePath)
}

func Test_GetModelMetaFromReadme_MetaIsEmpty(t *testing.T) {
	txt := `
# Body
Hello, it is my body
`
	testFilePath := fmt.Sprintf("/tmp/%v", rand.Int())
	_ = os.WriteFile(testFilePath, []byte(txt), 0644)
	_, err := GetModelMetaFromReadme(testFilePath)

	assert.Error(t, err)

	_ = os.Remove(testFilePath)
}

// func TestGitHubClone(t *testing.T) {
// 	tmpDir := fmt.Sprintf("/tmp/%v", uuid.New().String())
// 	assert.Nil(t, GitHubClone(tmpDir, datamodel.GitHub{
// 		RepoUrl: "https://github.com/instill-ai/model-dummy-cls.git",
// 		GitRef:  datamodel.GitRef{},
// 	}))
// 	os.RemoveAll(tmpDir)

// 	tmpDir = fmt.Sprintf("/tmp/%v", uuid.New().String())
// 	assert.Nil(t, GitHubClone(tmpDir, datamodel.GitHub{
// 		RepoUrl: "https://github.com/instill-ai/model-dummy-cls.git",
// 		GitRef: datamodel.GitRef{
// 			Branch: "main",
// 		},
// 	}))
// 	os.RemoveAll(tmpDir)

// 	tmpDir = fmt.Sprintf("/tmp/%v", uuid.New().String())
// 	assert.Nil(t, GitHubClone(tmpDir, datamodel.GitHub{
// 		RepoUrl: "https://github.com/instill-ai/model-dummy-cls.git",
// 		GitRef: datamodel.GitRef{
// 			Branch: "feat-a",
// 		},
// 	}))
// 	os.RemoveAll(tmpDir)

// 	tmpDir = fmt.Sprintf("/tmp/%v", uuid.New().String())
// 	assert.Nil(t, GitHubClone(tmpDir, datamodel.GitHub{
// 		RepoUrl: "https://github.com/instill-ai/model-dummy-cls.git",
// 		GitRef: datamodel.GitRef{
// 			Tag: "v1.0",
// 		},
// 	}))
// 	os.RemoveAll(tmpDir)

// 	tmpDir = fmt.Sprintf("/tmp/%v", uuid.New().String())
// 	assert.Nil(t, GitHubClone(tmpDir, datamodel.GitHub{
// 		RepoUrl: "https://github.com/instill-ai/model-dummy-cls.git",
// 		GitRef: datamodel.GitRef{
// 			Commit: "c2843d5af0f5316c60aafb9d2548811132076e28",
// 		},
// 	}))
// 	os.RemoveAll(tmpDir)

// 	assert.NotNil(t, GitHubClone(tmpDir, datamodel.GitHub{
// 		RepoUrl: "https://github.com/Phelan164/non-existed-repo.git",
// 		GitRef: datamodel.GitRef{
// 			Commit: "c2843d5af0f5316c60aafb9d2548811132076e28",
// 		},
// 	}))
// 	assert.NotNil(t, GitHubClone(tmpDir, datamodel.GitHub{
// 		RepoUrl: "https://github.com/instill-ai/model-dummy-cls.git",
// 		GitRef: datamodel.GitRef{
// 			Commit: "non-existed-commit-hash",
// 		},
// 	}))
// 	assert.NotNil(t, GitHubClone(tmpDir, datamodel.GitHub{
// 		RepoUrl: "https://github.com/instill-ai/model-dummy-cls.git",
// 		GitRef: datamodel.GitRef{
// 			Tag: "v10.0",
// 		},
// 	}))
// 	assert.NotNil(t, GitHubClone(tmpDir, datamodel.GitHub{
// 		RepoUrl: "https://github.com/instill-ai/model-dummy-cls.git",
// 		GitRef: datamodel.GitRef{
// 			Branch: "non-existed-branch",
// 		},
// 	}))
// }
