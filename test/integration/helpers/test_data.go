package helpers

// Test user credentials from seed data
const (
	// Admin credentials
	AdminEmail    = "admin@ecommerce.com"
	AdminPassword = "admin123"
	AdminUserID   = 1

	// Seller credentials
	SellerEmail    = "jane.merchant@example.com"
	SellerPassword = "seller123"
	SellerUserID   = 3 // Jane Merchant's user ID from seed data

	// Additional Seller (for products with multiple options)
	Seller2Email    = "john.seller@example.com"
	Seller2Password = "seller123"
	Seller2UserID   = 2 // John Seller's user ID from seed data

	// Another Seller (for Home & Living products)
	Seller4Email    = "bob.store@example.com"
	Seller4Password = "seller123"
	Seller4UserID   = 4 // Bob Store's user ID from seed data

	// Customer credentials
	CustomerEmail    = "alice.j@example.com"
	CustomerPassword = "customer123"
	CustomerUserID   = 5 // Alice Johnson's user ID from seed data (has seller_id = 2)

	// Additional Customers
	Customer2Email    = "michael.s@example.com"
	Customer2Password = "customer123"
	Customer2UserID   = 6 // Michael Smith's user ID from seed data (has seller_id = 3)

	Customer3Email    = "sarah.w@example.com"
	Customer3Password = "customer123"
	Customer3UserID   = 7 // Sarah Williams's user ID from seed data (has seller_id = 4)
)
