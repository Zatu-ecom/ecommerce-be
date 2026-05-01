package file_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/config"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/helper"
	"ecommerce-be/common/messaging"
	"ecommerce-be/common/scheduler"
	"ecommerce-be/file/entity"
	fileSingleton "ecommerce-be/file/factory/singleton"
	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/suite"
)

const (
	uploadInitEndpoint            = "/api/files/init-upload"
	uploadCompleteEndpoint        = "/api/files/complete-upload"
	uploadVariantQueueName        = "file.image.process.requested.q"
	uploadTestBucket              = "upload-suite-bucket"
	uploadTestSellerID     uint64 = 3 // jane.merchant@example.com
)

var schedulerOnce sync.Once

type VariantEnvelopeMessage struct {
	Envelope messaging.Envelope
	Payload  map[string]interface{}
}

// UploadSuite is the integration harness for upload init/complete happy-path behavior.
type UploadSuite struct {
	suite.Suite

	container *setup.TestContainer
	minio     *setup.MinioContainer
	rabbit    *setup.RabbitMQContainer

	server        http.Handler
	sellerToken   string
	seller2Token  string
	customerToken string
	adminToken    string

	variantMessages chan VariantEnvelopeMessage
}

func TestUploadSuite(t *testing.T) {
	suite.Run(t, new(UploadSuite))
}

func (s *UploadSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllSeeds(s.T())

	s.minio = setup.SetupMinioContainer(s.T())
	if err := s.minio.CreateBucket(context.Background(), uploadTestBucket); err != nil {
		s.T().Fatalf("failed to create minio bucket: %v", err)
	}

	s.rabbit = setup.SetupRabbitMQContainer(s.T())
	s.configureUploadEnv()

	fileSingleton.ResetInstance()
	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)
	s.seedUploadStorageConfig()

	s.variantMessages = make(chan VariantEnvelopeMessage, 32)
	s.setupVariantQueueConsumer()

	// Scheduler worker loop is global and should only be started once.
	schedulerOnce.Do(func() {
		go scheduler.StartRedisWorkerPool()
	})

	client := helpers.NewAPIClient(s.server)
	s.sellerToken = helpers.Login(s.T(), client, helpers.SellerEmail, helpers.SellerPassword)
	s.seller2Token = helpers.Login(s.T(), client, helpers.Seller2Email, helpers.Seller2Password)
	s.customerToken = helpers.Login(s.T(), client, helpers.CustomerEmail, helpers.CustomerPassword)
	s.adminToken = helpers.Login(s.T(), client, helpers.AdminEmail, helpers.AdminPassword)
}

func (s *UploadSuite) TearDownSuite() {
	fileSingleton.ResetInstance()

	if s.rabbit != nil {
		s.rabbit.Cleanup(s.T())
	}
	if s.minio != nil {
		s.minio.Cleanup(s.T())
	}
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

func (s *UploadSuite) configureUploadEnv() {
	_ = os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	_ = os.Setenv("MESSAGING_ENABLED", "true")
	_ = os.Setenv("MESSAGING_QUEUE_TYPE", "rabbitmq")
	_ = os.Setenv("RABBITMQ_URL", s.rabbit.AMQPURL)
	_ = os.Setenv("RABBITMQ_HOST", "")
	_ = os.Setenv("RABBITMQ_USER", "")
	_ = os.Setenv("RABBITMQ_PASSWORD", "")

	// Ensure fresh config singleton picks up messaging/rabbit values.
	config.Reset()
	cache.SetRedisClient(s.container.RedisClient)
}

func (s *UploadSuite) seedUploadStorageConfig() {
	key := os.Getenv("ENCRYPTION_KEY")
	rawCreds := `{"access_key_id":"` + s.minio.AccessKey + `","secret_access_key":"` + s.minio.SecretKey + `"}`
	encrypted, err := helper.Encrypt(rawCreds, key)
	s.Require().NoError(err)

	type providerRow struct {
		ID uint
	}
	var provider providerRow
	err = s.container.DB.
		Table("storage_provider").
		Select("id").
		Where("adapter_type = ? AND is_active = ?", "s3_compatible", true).
		Order("id ASC").
		First(&provider).Error
	s.Require().NoError(err)

	// Platform default.
	var platformConfigID uint
	err = s.container.DB.Raw(`
		INSERT INTO storage_config (
			owner_type, owner_id, provider_id, display_name, bucket_or_container, region, endpoint,
			base_path, force_path_style, credentials_encrypted, config_json, is_default, is_active,
			validation_status, created_at, updated_at
		) VALUES (
			'PLATFORM', NULL, ?, 'Upload Platform Default', ?, ?, ?, '', true, ?, '{}'::jsonb, true, true, 'SUCCESS', NOW(), NOW()
		) RETURNING id
	`, provider.ID, uploadTestBucket, s.minio.Region, s.minio.Endpoint, []byte(encrypted)).
		Scan(&platformConfigID).Error
	s.Require().NoError(err)

	// Seller config + active binding.
	var sellerConfigID uint
	err = s.container.DB.Raw(`
		INSERT INTO storage_config (
			owner_type, owner_id, provider_id, display_name, bucket_or_container, region, endpoint,
			base_path, force_path_style, credentials_encrypted, config_json, is_default, is_active,
			validation_status, created_at, updated_at
		) VALUES (
			'SELLER', ?, ?, 'Upload Seller Config', ?, ?, ?, '', true, ?, '{}'::jsonb, false, true, 'SUCCESS', NOW(), NOW()
		) RETURNING id
	`, uploadTestSellerID, provider.ID, uploadTestBucket, s.minio.Region, s.minio.Endpoint, []byte(encrypted)).
		Scan(&sellerConfigID).Error
	s.Require().NoError(err)

	err = s.container.DB.Exec(`DELETE FROM seller_storage_binding WHERE seller_id = ?`, uploadTestSellerID).Error
	s.Require().NoError(err)

	err = s.container.DB.Exec(`
		INSERT INTO seller_storage_binding (
			seller_id, storage_config_id, is_active, created_at, updated_at
		) VALUES (?, ?, true, NOW(), NOW())
	`, uploadTestSellerID, sellerConfigID).Error
	s.Require().NoError(err)
}

func (s *UploadSuite) setupVariantQueueConsumer() {
	ch, err := s.rabbit.Connection.Channel()
	s.Require().NoError(err)

	err = ch.ExchangeDeclare(
		constants.DEFAULT_COMMANDS_EXCHANGE,
		constants.DEFAULT_EXCHANGE_TYPE,
		true,
		false,
		false,
		false,
		nil,
	)
	s.Require().NoError(err)

	_, err = ch.QueueDeclare(uploadVariantQueueName, true, false, false, false, nil)
	s.Require().NoError(err)

	err = ch.QueueBind(
		uploadVariantQueueName,
		constants.ROUTING_KEY_FILE_IMAGE_PROCESS_REQUESTED,
		constants.DEFAULT_COMMANDS_EXCHANGE,
		false,
		nil,
	)
	s.Require().NoError(err)

	deliveries, err := ch.Consume(uploadVariantQueueName, "", true, false, false, false, nil)
	s.Require().NoError(err)

	go func() {
		for d := range deliveries {
			var env messaging.Envelope
			if err := json.Unmarshal(d.Body, &env); err != nil {
				continue
			}

			payload := map[string]interface{}{}
			_ = json.Unmarshal(env.Payload, &payload)

			select {
			case s.variantMessages <- VariantEnvelopeMessage{Envelope: env, Payload: payload}:
			case <-time.After(50 * time.Millisecond):
			}
		}
	}()
}

func (s *UploadSuite) nextVariantMessage(timeout time.Duration) VariantEnvelopeMessage {
	select {
	case msg := <-s.variantMessages:
		return msg
	case <-time.After(timeout):
		s.T().Fatal("timed out waiting for variant message")
		return VariantEnvelopeMessage{}
	}
}

func (s *UploadSuite) assertNoVariantMessage(within time.Duration) {
	select {
	case msg := <-s.variantMessages:
		s.T().Fatalf("unexpected variant message: %#v", msg.Payload)
	case <-time.After(within):
	}
}

func (s *UploadSuite) assertFileStatus(fileID string, status entity.FileStatus) {
	type row struct {
		Status string
	}
	var r row
	err := s.container.DB.Raw("SELECT status FROM file_object WHERE file_id = ?", fileID).Scan(&r).Error
	s.Require().NoError(err)
	s.Require().Equal(string(status), r.Status)
}

func (s *UploadSuite) countMinioObjects() int {
	cfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(s.minio.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s.minio.AccessKey, s.minio.SecretKey, ""),
		),
		awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...any) (aws.Endpoint, error) {
					if service == s3.ServiceID {
						return aws.Endpoint{
							URL:               s.minio.Endpoint,
							SigningRegion:     s.minio.Region,
							HostnameImmutable: true,
						}, nil
					}
					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				},
			),
		),
	)
	s.Require().NoError(err)

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	out, err := client.ListObjectsV2(
		context.Background(),
		&s3.ListObjectsV2Input{Bucket: aws.String(uploadTestBucket)},
	)
	s.Require().NoError(err)
	return len(out.Contents)
}
