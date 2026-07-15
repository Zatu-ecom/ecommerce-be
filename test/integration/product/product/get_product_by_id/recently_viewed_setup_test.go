package get_product_by_id

import (
	"net/http"
	"testing"
	"time"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// API endpoints for recently viewed feature.
const (
	ProductAPIEndpoint             = "/api/product"
	ProductByIDAPIEndpoint         = "/api/product/%d"
	RecentlyViewedAPIEndpoint      = "/api/product/recently-viewed"
	RecentlyViewedLimitAPIEndpoint = "/api/product/recently-viewed?limit=%d"
)

// RecentlyViewedTestSuite is the test suite for recently viewed product recording.
// Uses testify/suite pattern as mandated by the project constitution.
type RecentlyViewedTestSuite struct {
	suite.Suite
	container *setup.TestContainer
	server    http.Handler

	customerClient  *helpers.APIClient
	customer2Client *helpers.APIClient

	// One seller client reused across tests.
	sellerClient *helpers.APIClient
	adminClient  *helpers.APIClient
}

// SetupSuite initialises the test container, runs migrations/seeds, and creates
// authenticated API clients for each role.
func (s *RecentlyViewedTestSuite) SetupSuite() {
	s.container = setup.SetupTestContainers(s.T())
	s.container.RunAllMigrations(s.T())
	s.container.RunAllCoreSeeds(s.T())
	s.container.RunSeeds(s.T(), "migrations/seeds/mock/001_seed_users.sql")
	s.container.RunSeeds(s.T(), "migrations/seeds/mock/002_seed_products.sql")
	s.container.RunSeeds(s.T(), "test/integration/data/get_product_by_id_seed_data.sql")

	s.server = setup.SetupTestServer(s.T(), s.container.DB, s.container.RedisClient)

	// Customer (Alice) — recently viewed should be recorded for this role.
	s.customerClient = helpers.NewAPIClient(s.server)
	customerToken := helpers.Login(
		s.T(),
		s.customerClient,
		helpers.CustomerEmail,
		helpers.CustomerPassword,
	)
	s.customerClient.SetToken(customerToken)

	// Customer 2 (Michael) — used for user-isolation tests.
	s.customer2Client = helpers.NewAPIClient(s.server)
	customer2Token := helpers.Login(
		s.T(),
		s.customer2Client,
		helpers.Customer2Email,
		helpers.Customer2Password,
	)
	s.customer2Client.SetToken(customer2Token)

	// Seller (John Seller) — recently viewed should NOT be recorded.
	s.sellerClient = helpers.NewAPIClient(s.server)
	sellerToken := helpers.Login(
		s.T(),
		s.sellerClient,
		helpers.Seller2Email,
		helpers.Seller2Password,
	)
	s.sellerClient.SetToken(sellerToken)

	// Admin — recently viewed should NOT be recorded.
	s.adminClient = helpers.NewAPIClient(s.server)
	adminToken := helpers.Login(s.T(), s.adminClient, helpers.AdminEmail, helpers.AdminPassword)
	s.adminClient.SetToken(adminToken)
}

// TearDownSuite cleans up the test container.
func (s *RecentlyViewedTestSuite) TearDownSuite() {
	if s.container != nil {
		s.container.Cleanup(s.T())
	}
}

// SetupTest clears recently viewed data before every test for isolation.
func (s *RecentlyViewedTestSuite) SetupTest() {
	s.clearRecentlyViewed()
}

// TestRecentlyViewedSuite is the single entry point that runs all recently viewed tests.
func TestRecentlyViewedSuite(t *testing.T) {
	suite.Run(t, new(RecentlyViewedTestSuite))
}

// ============================================================================
// DB helper methods (attached to suite for reuse across test files)
// ============================================================================

// seedRecentlyViewed inserts a recently viewed row directly for test setup.
func (s *RecentlyViewedTestSuite) seedRecentlyViewed(
	userID, sellerID, productID uint,
	viewedAt time.Time,
) {
	result := s.container.DB.Exec(
		`INSERT INTO user_recently_viewed (user_id, seller_id, product_id, viewed_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (user_id, product_id)
		 DO UPDATE SET viewed_at = EXCLUDED.viewed_at, seller_id = EXCLUDED.seller_id, updated_at = NOW()`,
		userID,
		sellerID,
		productID,
		viewedAt,
		viewedAt,
		viewedAt,
	)
	s.Require().NoError(result.Error, "Failed to seed recently viewed record")
}

// countByUser returns the number of recently viewed rows for the given user.
func (s *RecentlyViewedTestSuite) countByUser(userID uint) int64 {
	var count int64
	err := s.container.DB.Raw(
		"SELECT COUNT(*) FROM user_recently_viewed WHERE user_id = ?", userID,
	).Scan(&count).Error
	s.Require().NoError(err, "Failed to count recently viewed records")
	return count
}

// countByUserAndProduct returns 1 if the user-product pair exists, 0 otherwise.
func (s *RecentlyViewedTestSuite) countByUserAndProduct(userID, productID uint) int64 {
	var count int64
	err := s.container.DB.Raw(
		"SELECT COUNT(*) FROM user_recently_viewed WHERE user_id = ? AND product_id = ?",
		userID, productID,
	).Scan(&count).Error
	s.Require().NoError(err, "Failed to check recently viewed record")
	return count
}

// clearRecentlyViewed deletes all rows from the recently viewed table.
func (s *RecentlyViewedTestSuite) clearRecentlyViewed() {
	err := s.container.DB.Exec("DELETE FROM user_recently_viewed").Error
	s.Require().NoError(err, "Failed to clear recently viewed records")
}

// countAllRecentlyViewed returns the total row count across all users.
func (s *RecentlyViewedTestSuite) countAllRecentlyViewed() int64 {
	var count int64
	err := s.container.DB.Raw("SELECT COUNT(*) FROM user_recently_viewed").Scan(&count).Error
	s.Require().NoError(err, "Failed to count all recently viewed records")
	return count
}

// DB is exposed for direct gorm.DB access in test assertions (convenience).
func (s *RecentlyViewedTestSuite) DB() *gorm.DB {
	return s.container.DB
}
