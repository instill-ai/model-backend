package minio_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/minio"
)

func TestMinio(t *testing.T) {
	t.Skipf("only for testing on local")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mc, err := minio.NewMinioClientAndInitBucket(&config.MinioConfig{
		Host:       "localhost",
		Port:       "19000",
		RootUser:   "minioadmin",
		RootPwd:    "minioadmin",
		BucketName: "instill-ai-model",
	})
	require.NoError(t, err)

	fileName, _ := uuid.NewV4()
	uid, _ := uuid.NewV4()
	fileContent := base64.StdEncoding.EncodeToString([]byte(uid.String()))
	err = mc.UploadBase64File(ctx, fileName.String(), fileContent, "text/plain")
	require.NoError(t, err)

	fileBytes, err := mc.GetFile(ctx, fileName.String())
	require.NoError(t, err)
	require.Equal(t, uid.String(), string(fileBytes))
}
