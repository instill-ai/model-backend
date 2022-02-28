package test

import (
	"context"
	"database/sql/driver"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	"github.com/instill-ai/model-backend/configs"
	database "github.com/instill-ai/model-backend/internal/db"
	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/services"
	"github.com/instill-ai/protogen-go/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/instill-ai/model-backend/rpc"
	"github.com/stretchr/testify/suite"
)

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

type UploadModelTestSuite struct {
	suite.Suite
	mock sqlmock.Sqlmock
}

func dialer() func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)

	server := grpc.NewServer()

	db := database.GetConnection()
	modelRepository := repository.NewModelRepository(db)
	modelService := services.NewModelService(modelRepository)
	modelServiceHandler := rpc.NewServiceHandlers(modelService)

	model.RegisterModelServer(server, modelServiceHandler)

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

// This will run right before the test starts
// and receives the suite and test names as input
func (suite *UploadModelTestSuite) BeforeTest(suiteName, testName string) {}

// This will run after test finishes
// and receives the suite and test names as input
func (suite *UploadModelTestSuite) AfterTest(suiteName, testName string) {
	require.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// This will run before before the tests in the suite are run
func (suite *UploadModelTestSuite) SetupSuite() {
	_ = configs.Init()
}

// This will run before each test in the suite
func (suite *UploadModelTestSuite) SetupTest() {}

type AnyTime struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func (suite *UploadModelTestSuite) TestUploadModelNormal() {

	modelName := RandomString(6)
	description := RandomString(15)

	suite.mock.ExpectBegin()
	suite.mock.ExpectExec("INSERT INTO `models` (.+) VALUES (.+)").
		WithArgs("densenet_onnx", modelName, false, description, "tensorrt", "pytorch", AnyTime{}, AnyTime{}, "domain@instill.tech", "", "public", "local-user@instill.tech").
		WillReturnResult(sqlmock.NewResult(1, 1))
	suite.mock.ExpectCommit()

	suite.mock.ExpectBegin()
	suite.mock.ExpectExec("INSERT INTO `versions` (.+) VALUES (.+)").
		WithArgs("densenet_onnx", 1, description, AnyTime{}, AnyTime{}, "offline", "{}").
		WillReturnResult(sqlmock.NewResult(1, 1))
	suite.mock.ExpectCommit()

	suite.mock.ExpectQuery("SELECT(.*)").
		WillReturnRows(
			sqlmock.NewRows([]string{"name", "id", "version", "optimized", "type", "framework", "status", "created_at", "modified_at"}).
				AddRow(modelName, "densenet_onnx", 1, false, "tensorrt", "pytorch", "offline", time.Now(), time.Now()))

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := model.NewModelClient(conn)

	streamUploader, _ := c.CreateModelByUpload(ctx)
	defer streamUploader.CloseSend()

	const chunkSize = 64 * 1024 // 64 KiB
	buf := make([]byte, chunkSize)
	firstChunk := true

	file, _ := os.Open("data/densenet_onnx.zip")

	defer file.Close()

	for {
		n, errRead := file.Read(buf)
		if errRead != nil {
			if errRead == io.EOF {
				break
			}

			break
		}
		if firstChunk {
			_ = streamUploader.Send(&model.CreateModelRequest{
				Description: description,
				Content:     buf[:n],
			})
			firstChunk = false
		} else {
			_ = streamUploader.Send(&model.CreateModelRequest{
				Content: buf[:n],
			})
		}
	}

	response, _ := streamUploader.CloseAndRecv()
	suite.T().Run("TestUploadModelNormal", func(t *testing.T) {
		assert.Equal(t, response, nil)
	})
}

func TestUploadModelTestSuite(t *testing.T) {
	suite.Run(t, new(UploadModelTestSuite))
}
