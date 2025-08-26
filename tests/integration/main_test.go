package integration

import (
	"context"
	"fmt"
	ctn "github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"io"
	"log"
	"os"
	"postService/internal/bootstrap"
	"postService/internal/routes"
	"postService/tests/testutil" // ✅ import testutil for JWKS + token
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testApp   *gin.Engine
	container *bootstrap.Container
	jwksURL   string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// --- Create shared Docker network ---
	net, err := network.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create network: %v", err)
	}
	defer func() {
		if err := net.Remove(ctx); err != nil {
			log.Printf("Failed to remove network: %v", err)
		}
	}()

	// --- MongoDB ---
	mongoReq := testcontainers.ContainerRequest{
		Image:        "mongo:6.0",
		ExposedPorts: []string{"27017/tcp"},
		Networks:     []string{net.Name},
		WaitingFor:   wait.ForListeningPort("27017/tcp"),
		NetworkAliases: map[string][]string{
			net.Name: {"mongo"},
		},
	}
	mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: mongoReq, Started: true})
	require.NoError(nil, err)
	mongoHost, _ := mongoC.Host(ctx)
	mongoPort, _ := mongoC.MappedPort(ctx, "27017")
	mongoURI := fmt.Sprintf("mongodb://%s:%s", mongoHost, mongoPort.Port())
	// --- Redis ---
	redisReq := testcontainers.ContainerRequest{
		Image:        "redis:7.0",
		ExposedPorts: []string{"6379/tcp"},
		Networks:     []string{net.Name},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
		NetworkAliases: map[string][]string{
			net.Name: {"redis"},
		},
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: redisReq, Started: true})
	require.NoError(nil, err)
	redisHost, _ := redisC.Host(ctx)
	redisPort, _ := redisC.MappedPort(ctx, "6379")
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort.Port())

	// --- MinIO ---
	minioReq := testcontainers.ContainerRequest{
		Image:        "minio/minio:RELEASE.2025-04-22T22-12-26Z",
		ExposedPorts: []string{"9000/tcp"},
		Cmd:          []string{"server", "/data", "--console-address", ":9090"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "admin",
			"MINIO_ROOT_PASSWORD": "admin123",
		},
		WaitingFor: wait.ForLog("API: http://"),
		NetworkAliases: map[string][]string{
			net.Name: {"minio"},
		},
	}

	minioC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: minioReq,
		Started:          true,
	})
	require.NoError(nil, err)

	// get dynamically mapped host/port
	minioHost, _ := minioC.Host(ctx)
	minioPort, _ := minioC.MappedPort(ctx, "9000/tcp")

	endpoint := fmt.Sprintf("%s:%s", minioHost, minioPort.Port())
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("admin", "admin123", ""),
		Secure: false,
	})
	require.NoError(nil, err)

	// --- Create bucket ---
	bucketName := "test-bucket"
	err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: "us-east-1"})
	if err != nil {
		// If bucket already exists, ignore
		exists, errBucketExists := client.BucketExists(ctx, bucketName)
		require.NoError(nil, errBucketExists)
		if !exists {
			require.NoError(nil, err) // fail only if bucket really missing
		}
	}

	kafkaReq := testcontainers.ContainerRequest{
		Name:         "kafka",
		Image:        "bitnami/kafka:3.6.1",
		ExposedPorts: []string{"9092/tcp"},
		Networks:     []string{net.Name},
		NetworkAliases: map[string][]string{
			net.Name: {"kafka"},
		},
		Env: map[string]string{
			"KAFKA_ENABLE_KRAFT":                       "yes",
			"KAFKA_CFG_PROCESS_ROLES":                  "broker,controller",
			"KAFKA_CFG_NODE_ID":                        "1",
			"KAFKA_CFG_LISTENERS":                      "PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093",
			"KAFKA_CFG_ADVERTISED_LISTENERS":           "PLAINTEXT://localhost:9092", // client connects here
			"KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP": "PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT",
			"KAFKA_CFG_CONTROLLER_QUORUM_VOTERS":       "1@kafka:9093",
			"KAFKA_CFG_CONTROLLER_LISTENER_NAMES":      "CONTROLLER",
			"KAFKA_CFG_INTER_BROKER_LISTENER_NAME":     "PLAINTEXT",
			"KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE":      "true",
		},
		WaitingFor: wait.ForListeningPort("9092/tcp").WithStartupTimeout(120 * time.Second),
		HostConfigModifier: func(hc *ctn.HostConfig) {
			hc.PortBindings = nat.PortMap{
				"9092/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "9092"}},
			}
		},
	}

	kafkaC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: kafkaReq,
		Started:          true,
	})
	require.NoError(nil, err)

	kafkaAddr := "localhost:9092"

	execCmd := []string{
		"/opt/bitnami/kafka/bin/kafka-topics.sh",
		"--create",
		"--topic", "post-events",
		"--bootstrap-server", "localhost:9092",
		"--partitions", "1",
		"--replication-factor", "1",
	}

	exitCode, reader, err := kafkaC.Exec(ctx, execCmd)
	require.NoError(nil, err)
	if exitCode != 0 {
		body, _ := io.ReadAll(reader)
		log.Fatalf("Failed to create topic: %s", string(body))
	}

	// --- JWKS mock ---
	jwksURL = "http://localhost:9095/certs"
	testutil.StartMockJWKS(":9095")

	// --- Bootstrap Application ---
	container = bootstrap.NewTestContainer(mongoURI, kafkaAddr, minioHost, minioPort.Port(), redisAddr, jwksURL)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	//logger := logging.GetLogger()
	//logger.SetLevel(logrus.PanicLevel)

	routes.SetupPostRoutes(r, container)
	testApp = r

	defer func() {
		if err := mongoC.Terminate(ctx); err != nil {
			log.Printf("failed to terminate mongo container: %v", err)
		}
	}()
	defer func() {
		if err := redisC.Terminate(ctx); err != nil {
			log.Printf("failed to terminate redis container: %v", err)
		}
	}()
	defer func() {
		if err := minioC.Terminate(ctx); err != nil {
			log.Printf("failed to terminate minio container: %v", err)
		}
	}()
	defer func() {
		if err := kafkaC.Terminate(ctx); err != nil {
			log.Printf("failed to terminate kafka container: %v", err)
		}
	}()
	code := m.Run()
	os.Exit(code)
}
