package product

import (
	"fmt"
	"net/http"
	"testing"

	"ecommerce-be/test/integration/helpers"
	"ecommerce-be/test/integration/setup"

	"github.com/stretchr/testify/assert"
)

func TestCreateProduct(t *testing.T) {
	// Setup test containers
	containers := setup.SetupTestContainers(t)
	defer containers.Cleanup(t)

	// Run migrations and seeds
	containers.RunAllMigrations(t)
	containers.RunAllCoreSeeds(t)
	containers.RunSeeds(t, "migrations/seeds/mock/001_seed_users.sql")
	containers.RunSeeds(t, "migrations/seeds/mock/002_seed_products.sql")

	// Setup test server
	server := setup.SetupTestServer(t, containers.DB, containers.RedisClient)

	// Create API client
	client := helpers.NewAPIClient(server)

	// ============================================================================
	// SUCCESS SCENARIOS
	// ============================================================================

	t.Run("Success - Create product with single variant (minimal data)", func(t *testing.T) {
		// Login as seller
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product - Single Variant",
			"categoryId": 4, // Smartphones category
			"baseSku":    "TEST-SINGLE-001",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{
							"value":       "Black",
							"displayName": "Black",
						},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-SINGLE-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{
							"optionName": "color",
							"value":      "black",
						},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody) // Debug: Print response if not successful
		if w.Code != http.StatusCreated {
			t.Logf("Response Status: %d", w.Code)
			t.Logf("Response Body: %s", w.Body.String())
		}

		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify product created
		assert.NotNil(t, product["id"])
		assert.Equal(t, "Test Product - Single Variant", product["name"])
		assert.Equal(t, float64(4), product["categoryId"])
		assert.Equal(t, "TEST-SINGLE-001", product["sku"])
		assert.Equal(t, float64(helpers.SellerUserID), product["sellerId"])

		// Verify variant created
		assert.NotNil(t, product["variants"])
		variants, ok := product["variants"].([]any)
		assert.True(t, ok, "variants should be an array")
		assert.Len(t, variants, 1, "Should have 1 variant")

		variant := variants[0].(map[string]any)
		assert.NotNil(t, variant["id"])
		assert.Equal(t, "TEST-SINGLE-001-V1", variant["sku"])
		assert.Equal(t, 99.99, variant["price"])

		// Verify option created
		selectedOptions, ok := variant["selectedOptions"].([]any)
		assert.True(t, ok, "selectedOptions should be an array")
		assert.Len(t, selectedOptions, 1, "Should have 1 option")

		option := selectedOptions[0].(map[string]any)
		assert.NotNil(t, option["optionId"])
		assert.Equal(t, "Color", option["optionDisplayName"])
		assert.NotNil(t, option["valueId"])
		assert.Equal(t, "Black", option["valueDisplayName"])

		// Verify product options structure
		assert.NotNil(t, product["options"])
		productOptions, ok := product["options"].([]any)
		assert.True(t, ok, "options should be an array")
		assert.Len(t, productOptions, 1, "Should have 1 product option")
	})

	t.Run("Success - Create product with multiple variants (2 options)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test T-Shirt - Multiple Variants",
			"categoryId": 7, // Men's Clothing
			"baseSku":    "TEST-TSHIRT-001",
			"brand":      "TestBrand",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
						{"value": "white", "displayName": "White"},
					},
				},
				{
					"name":        "Size",
					"displayName": "Size",
					"values": []map[string]any{
						{"value": "m", "displayName": "Medium"},
						{"value": "l", "displayName": "Large"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-TSHIRT-BLK-M",
					"price": 29.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
						{"optionName": "size", "value": "m"},
					},
				},
				{
					"sku":   "TEST-TSHIRT-BLK-L",
					"price": 29.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
						{"optionName": "size", "value": "l"},
					},
				},
				{
					"sku":   "TEST-TSHIRT-WHT-M",
					"price": 29.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "white"},
						{"optionName": "size", "value": "m"},
					},
				},
				{
					"sku":   "TEST-TSHIRT-WHT-L",
					"price": 29.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "white"},
						{"optionName": "size", "value": "l"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify product created
		assert.NotNil(t, product["id"])
		assert.Equal(t, "Test T-Shirt - Multiple Variants", product["name"])
		assert.Equal(t, "TestBrand", product["brand"])
		assert.Equal(t, float64(7), product["categoryId"])

		// Verify all variants created
		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		assert.Len(t, variants, 4, "Should have 4 variants")

		// Verify each variant has correct options
		skuMap := make(map[string]bool)
		for _, v := range variants {
			variant := v.(map[string]any)
			sku := variant["sku"].(string)
			skuMap[sku] = true

			selectedOptions := variant["selectedOptions"].([]any)
			assert.Len(t, selectedOptions, 2, "Each variant should have 2 options")
		}

		// Verify all expected SKUs are present
		expectedSKUs := []string{
			"TEST-TSHIRT-BLK-M",
			"TEST-TSHIRT-BLK-L",
			"TEST-TSHIRT-WHT-M",
			"TEST-TSHIRT-WHT-L",
		}
		for _, sku := range expectedSKUs {
			assert.True(t, skuMap[sku], fmt.Sprintf("SKU %s should exist", sku))
		}

		// Verify product has 2 options (Color and Size)
		productOptions, ok := product["options"].([]any)
		assert.True(t, ok)
		assert.Len(t, productOptions, 2, "Should have 2 product options")

		// Verify option values are created
		optionNames := make(map[string]int)
		for _, opt := range productOptions {
			option := opt.(map[string]any)
			name := option["optionName"].(string)
			optionNames[name]++

			values := option["values"].([]any)
			switch name {
			case "color":
				assert.Len(t, values, 2, "Color should have 2 values (black, white)")
			case "size":
				assert.Len(t, values, 2, "Size should have 2 values (m, l)")
			}
		}

		assert.Equal(t, 1, optionNames["color"], "Should have Color option")
		assert.Equal(t, 1, optionNames["size"], "Should have Size option")
	})

	t.Run("Success - Create product with 3 options", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Laptop - Complex Variants",
			"categoryId": 5, // Laptops
			"baseSku":    "TEST-LAPTOP-001",
			"brand":      "TestTech",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "silver", "displayName": "Silver"},
						{"value": "black", "displayName": "Black"},
					},
				},
				{
					"name":        "Memory",
					"displayName": "RAM",
					"values": []map[string]any{
						{"value": "8gb", "displayName": "8GB"},
						{"value": "16gb", "displayName": "16GB"},
					},
				},
				{
					"name":        "Storage",
					"displayName": "Storage",
					"values": []map[string]any{
						{"value": "256gb", "displayName": "256GB"},
						{"value": "512gb", "displayName": "512GB"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-LAPTOP-SLV-8-256",
					"price": 1299.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "silver"},
						{"optionName": "memory", "value": "8gb"},
						{"optionName": "storage", "value": "256gb"},
					},
				},
				{
					"sku":   "TEST-LAPTOP-SLV-16-512",
					"price": 1799.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "silver"},
						{"optionName": "memory", "value": "16gb"},
						{"optionName": "storage", "value": "512gb"},
					},
				},
				{
					"sku":   "TEST-LAPTOP-BLK-8-256",
					"price": 1299.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
						{"optionName": "memory", "value": "8gb"},
						{"optionName": "storage", "value": "256gb"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify product created
		assert.NotNil(t, product["id"])
		assert.Equal(t, "Test Laptop - Complex Variants", product["name"])

		// Verify variants
		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		assert.Len(t, variants, 3, "Should have 3 variants")

		// Verify each variant has 3 options
		for _, v := range variants {
			variant := v.(map[string]any)
			selectedOptions := variant["selectedOptions"].([]any)
			assert.Len(t, selectedOptions, 3, "Each variant should have 3 options")
		}

		// Verify product options
		productOptions, ok := product["options"].([]any)
		assert.True(t, ok)
		assert.Len(t, productOptions, 3, "Should have 3 product options")

		// Verify price differences
		var foundLowPrice, foundHighPrice bool
		for _, v := range variants {
			variant := v.(map[string]any)
			price := variant["price"].(float64)
			if price == 1299.99 {
				foundLowPrice = true
			}
			if price == 1799.99 {
				foundHighPrice = true
			}
		}
		assert.True(t, foundLowPrice, "Should have variant with lower price")
		assert.True(t, foundHighPrice, "Should have variant with higher price")
	})

	t.Run("Success - Create product with default variant marked", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		isDefault := true
		requestBody := map[string]any{
			"name":       "Test Product - Default Variant",
			"categoryId": 7,
			"baseSku":    "TEST-DEFAULT-001",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
						{"value": "white", "displayName": "White"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":       "TEST-DEFAULT-001-BLK",
					"price":     39.99,
					"isDefault": isDefault,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
				{
					"sku":   "TEST-DEFAULT-001-WHT",
					"price": 39.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "white"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify variants
		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		assert.Len(t, variants, 2)

		// Verify default variant is marked
		var foundDefault bool
		for _, v := range variants {
			variant := v.(map[string]any)
			switch variant["sku"] {
			case "TEST-DEFAULT-001-BLK":
				assert.Equal(t, true, variant["isDefault"], "Black variant should be default")
				foundDefault = true
			case "TEST-DEFAULT-001-WHT":
				assert.Equal(t, false, variant["isDefault"], "White variant should not be default")
			}
		}
		assert.True(t, foundDefault, "Should have found default variant")
	})

	t.Run("Success - Create product with popular variant", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		isPopular := true
		requestBody := map[string]any{
			"name":       "Test Product - Popular Variant",
			"categoryId": 7,
			"baseSku":    "TEST-POPULAR-001",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
						{"value": "red", "displayName": "Red"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":       "TEST-POPULAR-001-BLK",
					"price":     49.99,
					"isPopular": isPopular,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
				{
					"sku":   "TEST-POPULAR-001-RED",
					"price": 49.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "red"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify variants
		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		assert.Len(t, variants, 2)

		// Verify popular variant is marked
		var foundPopular bool
		for _, v := range variants {
			variant := v.(map[string]any)
			switch variant["sku"] {
			case "TEST-POPULAR-001-BLK":
				assert.Equal(t, true, variant["isPopular"], "Black variant should be popular")
				foundPopular = true
			case "TEST-POPULAR-001-RED":
				assert.Equal(t, false, variant["isPopular"], "Red variant should not be popular")
			}
		}
		assert.True(t, foundPopular, "Should have found popular variant")
	})

	t.Run("Success - Create product with all optional fields", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":             "Test Product - Full Featured",
			"categoryId":       4,
			"baseSku":          "TEST-FULL-001",
			"brand":            "Premium Brand",
			"shortDescription": "This is a short description of the product",
			"longDescription":  "This is a much longer description with more details about the product features and specifications",
			"tags":             []string{"premium", "featured", "bestseller"},
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "gold", "displayName": "Gold"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-FULL-001-V1",
					"price": 199.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "gold"},
					},
				},
			},
			"attributes": []map[string]any{
				{
					"key":   "screen_size",
					"name":  "Screen Size",
					"value": "6.5",
					"unit":  "inches",
				},
				{
					"key":   "battery",
					"name":  "Battery",
					"value": "5000",
					"unit":  "mAh",
				},
			},
			"packageOptions": []map[string]any{
				{
					"name":        "Extended Warranty",
					"description": "2 year extended warranty",
					"price":       49.99,
					"quantity":    1,
				},
				{
					"name":        "Screen Protector",
					"description": "Tempered glass screen protector",
					"price":       9.99,
					"quantity":    1,
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify product created with all fields
		assert.NotNil(t, product["id"])
		assert.Equal(t, "Test Product - Full Featured", product["name"])
		assert.Equal(t, "Premium Brand", product["brand"])
		assert.Equal(t, "This is a short description of the product", product["shortDescription"])
		assert.Equal(
			t,
			"This is a much longer description with more details about the product features and specifications",
			product["longDescription"],
		)

		// Verify tags
		tags, ok := product["tags"].([]any)
		assert.True(t, ok)
		assert.Len(t, tags, 3)
		assert.Contains(t, tags, "premium")
		assert.Contains(t, tags, "featured")
		assert.Contains(t, tags, "bestseller")

		// Verify variant exists with media field (images are now managed via variant media endpoints)
		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		assert.Len(t, variants, 1)
		variant := variants[0].(map[string]any)
		_, ok = variant["media"].([]any)
		assert.True(t, ok, "media should be a JSON array")

		// Verify product attributes
		attributes, ok := product["attributes"].([]any)
		assert.True(t, ok)
		assert.Len(t, attributes, 2)

		attributeMap := make(map[string]map[string]any)
		for _, a := range attributes {
			attr := a.(map[string]any)
			key := attr["key"].(string)
			attributeMap[key] = attr
		}

		assert.Contains(t, attributeMap, "screen_size")
		assert.Equal(t, "6.5", attributeMap["screen_size"]["value"])
		assert.Equal(t, "inches", attributeMap["screen_size"]["unit"])

		assert.Contains(t, attributeMap, "battery")
		assert.Equal(t, "5000", attributeMap["battery"]["value"])
		assert.Equal(t, "mAh", attributeMap["battery"]["unit"])

		// Verify package options
		packageOptions, ok := product["packageOptions"].([]any)
		assert.True(t, ok)
		assert.Len(t, packageOptions, 2)

		packageMap := make(map[string]map[string]any)
		for _, p := range packageOptions {
			pkg := p.(map[string]any)
			name := pkg["name"].(string)
			packageMap[name] = pkg
		}

		assert.Contains(t, packageMap, "Extended Warranty")
		assert.Equal(t, 49.99, packageMap["Extended Warranty"]["price"])

		assert.Contains(t, packageMap, "Screen Protector")
		assert.Equal(t, 9.99, packageMap["Screen Protector"]["price"])
	})

	t.Run("Success - Create product with product attributes", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Smartphone - With Attributes",
			"categoryId": 4,
			"baseSku":    "TEST-ATTR-001",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "blue", "displayName": "Blue"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-ATTR-001-V1",
					"price": 699.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "blue"},
					},
				},
			},
			"attributes": []map[string]any{
				{
					"key":       "processor",
					"name":      "Processor",
					"value":     "Snapdragon 8 Gen 2",
					"sortOrder": 1,
				},
				{
					"key":       "ram",
					"name":      "RAM",
					"value":     "8",
					"unit":      "GB",
					"sortOrder": 2,
				},
				{
					"key":       "storage",
					"name":      "Storage",
					"value":     "128",
					"unit":      "GB",
					"sortOrder": 3,
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify product attributes created
		attributes, ok := product["attributes"].([]any)
		assert.True(t, ok)
		assert.Len(t, attributes, 3, "Should have 3 attributes")

		// Verify each attribute has required fields
		for _, a := range attributes {
			attr := a.(map[string]any)
			assert.NotNil(t, attr["id"])
			assert.NotNil(t, attr["key"])
			assert.NotNil(t, attr["name"])
			assert.NotNil(t, attr["value"])
		}

		// Verify attribute with unit
		var foundRAM bool
		for _, a := range attributes {
			attr := a.(map[string]any)
			if attr["key"] == "ram" {
				foundRAM = true
				assert.Equal(t, "8", attr["value"])
				assert.Equal(t, "GB", attr["unit"])
			}
		}
		assert.True(t, foundRAM, "Should find RAM attribute")
	})

	t.Run("Success - Create product with package options", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product - With Packages",
			"categoryId": 4,
			"baseSku":    "TEST-PKG-001",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "green", "displayName": "Green"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-PKG-001-V1",
					"price": 499.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "green"},
					},
				},
			},
			"packageOptions": []map[string]any{
				{
					"name":        "Care Package",
					"description": "Complete protection package",
					"price":       99.99,
					"quantity":    1,
				},
				{
					"name":        "Accessory Bundle",
					"description": "Case, charger, and earphones",
					"price":       149.99,
					"quantity":    1,
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify package options created
		packageOptions, ok := product["packageOptions"].([]any)
		assert.True(t, ok)
		assert.Len(t, packageOptions, 2, "Should have 2 package options")

		// Verify each package has required fields
		for _, p := range packageOptions {
			pkg := p.(map[string]any)
			assert.NotNil(t, pkg["id"])
			assert.NotNil(t, pkg["name"])
			assert.NotNil(t, pkg["price"])
			assert.NotNil(t, pkg["quantity"])
		}

		// Verify specific package option
		var foundCarePackage bool
		for _, p := range packageOptions {
			pkg := p.(map[string]any)
			if pkg["name"] == "Care Package" {
				foundCarePackage = true
				assert.Equal(t, "Complete protection package", pkg["description"])
				assert.Equal(t, 99.99, pkg["price"])
				assert.Equal(t, float64(1), pkg["quantity"])
			}
		}
		assert.True(t, foundCarePackage, "Should find Care Package option")
	})

	t.Run("Success - Create product with multiple variants each has empty media", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Images are now managed via POST /product/:id/variant/:id/media after variant creation
		requestBody := map[string]any{
			"name":       "Test Product - Multiple Variants",
			"categoryId": 7,
			"baseSku":    "TEST-IMG-001",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black", "displayName": "Black"},
						{"value": "red", "displayName": "Red"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-IMG-001-BLK",
					"price": 79.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
				{
					"sku":   "TEST-IMG-001-RED",
					"price": 79.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "red"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		assert.Len(t, variants, 2)

		// Each variant should have an empty media array; use variant media endpoints to attach files
		for _, v := range variants {
			variant := v.(map[string]any)
			_, ok := variant["media"].([]any)
			assert.True(t, ok, "media should always be a JSON array")
		}
	})

	t.Run("Success - Create product with Color option having colorCode", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product - With Color Codes",
			"categoryId": 7,
			"baseSku":    "TEST-COLOR-001",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{
							"value":       "Black",
							"displayName": "Black",
							"colorCode":   "#000000",
						},
						{
							"value":       "White",
							"displayName": "White",
							"colorCode":   "#FFFFFF",
						},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-COLOR-001-BLK",
					"price": 59.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
				{
					"sku":   "TEST-COLOR-001-WHT",
					"price": 59.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "white"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify product options with color codes
		productOptions, ok := product["options"].([]any)
		assert.True(t, ok)
		assert.Len(t, productOptions, 1)

		option := productOptions[0].(map[string]any)
		assert.Equal(t, "color", option["optionName"])

		values, ok := option["values"].([]any)
		assert.True(t, ok)
		assert.Len(t, values, 2)

		// Verify color codes
		colorCodeMap := make(map[string]string)
		for _, v := range values {
			val := v.(map[string]any)
			colorCodeMap[val["value"].(string)] = val["colorCode"].(string)
		}

		assert.Equal(t, "#000000", colorCodeMap["black"])
		assert.Equal(t, "#FFFFFF", colorCodeMap["white"])
	})

	t.Run("Success - Create product in different categories", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Test with Electronics category
		requestBody1 := map[string]any{
			"name":       "Test Electronics Product",
			"categoryId": 4, // Smartphones
			"baseSku":    "TEST-ELEC-001",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "silver", "displayName": "Silver"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-ELEC-001-V1",
					"price": 299.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "silver"},
					},
				},
			},
		}

		w1 := client.Post(t, "/api/product", requestBody1)
		response1 := helpers.AssertSuccessResponse(t, w1, http.StatusCreated)
		product1 := helpers.GetResponseData(t, response1, "product")
		assert.Equal(t, float64(4), product1["categoryId"])

		// Test with Fashion category
		requestBody2 := map[string]any{
			"name":       "Test Fashion Product",
			"categoryId": 7, // Men's Clothing
			"baseSku":    "TEST-FASH-001",
			"options": []map[string]any{
				{
					"name":        "Size",
					"displayName": "Size",
					"values": []map[string]any{
						{"value": "l", "displayName": "Large"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-FASH-001-V1",
					"price": 49.99,
					"options": []map[string]any{
						{"optionName": "size", "value": "l"},
					},
				},
			},
		}

		w2 := client.Post(t, "/api/product", requestBody2)
		response2 := helpers.AssertSuccessResponse(t, w2, http.StatusCreated)
		product2 := helpers.GetResponseData(t, response2, "product")
		assert.Equal(t, float64(7), product2["categoryId"])

		// Test with Home & Living category
		requestBody3 := map[string]any{
			"name":       "Test Furniture Product",
			"categoryId": 10, // Furniture
			"baseSku":    "TEST-FURN-001",
			"options": []map[string]any{
				{
					"name":        "Material",
					"displayName": "Material",
					"values": []map[string]any{
						{"value": "wood", "displayName": "Wood"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-FURN-001-V1",
					"price": 599.99,
					"options": []map[string]any{
						{"optionName": "material", "value": "wood"},
					},
				},
			},
		}

		w3 := client.Post(t, "/api/product", requestBody3)
		response3 := helpers.AssertSuccessResponse(t, w3, http.StatusCreated)
		product3 := helpers.GetResponseData(t, response3, "product")
		assert.Equal(t, float64(10), product3["categoryId"])
	})

	t.Run("Success - Create product with special characters in name", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Men's Premium T-Shirt & Jeans™",
			"categoryId": 7,
			"baseSku":    "TEST-SPECIAL-001",
			"brand":      "O'Reilly & Sons",
			"options": []map[string]any{
				{
					"name":        "Size",
					"displayName": "Size",
					"values": []map[string]any{
						{"value": "m", "displayName": "Medium"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-SPECIAL-001-V1",
					"price": 89.99,
					"options": []map[string]any{
						{"optionName": "size", "value": "m"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify special characters are preserved
		assert.Equal(t, "Men's Premium T-Shirt & Jeans™", product["name"])
		assert.Equal(t, "O'Reilly & Sons", product["brand"])
	})

	t.Run("Success - Create product with URL-encoded special characters", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product - Special Options",
			"categoryId": 9, // Footwear
			"baseSku":    "TEST-ENCODE-001",
			"options": []map[string]any{
				{
					"name":        "Color",
					"displayName": "Color",
					"values": []map[string]any{
						{"value": "black/white", "displayName": "Black/White"},
					},
				},
				{
					"name":        "Size",
					"displayName": "Size",
					"values": []map[string]any{
						{"value": "10", "displayName": "10"},
					},
				},
			},
			"variants": []map[string]any{
				{
					"sku":   "TEST-ENCODE-001-BW",
					"price": 129.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black/white"},
						{"optionName": "size", "value": "10"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		response := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		product := helpers.GetResponseData(t, response, "product")

		// Verify option value with slash is preserved
		variants, ok := product["variants"].([]any)
		assert.True(t, ok)
		assert.Len(t, variants, 1)

		variant := variants[0].(map[string]any)
		selectedOptions := variant["selectedOptions"].([]any)

		var foundSlashValue bool
		for _, opt := range selectedOptions {
			option := opt.(map[string]any)
			if option["optionDisplayName"] == "Color" {
				assert.Equal(t, "Black/White", option["valueDisplayName"])
				foundSlashValue = true
			}
		}
		assert.True(t, foundSlashValue, "Should find color option with slash")
	})

	// ============================================================================
	// VALIDATION ERROR SCENARIOS
	// ============================================================================

	t.Run("Error - Missing required field: name", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			// name is missing
			"categoryId": 4,
			"baseSku":    "TEST-NONAME-001",
			"variants": []map[string]any{
				{
					"sku":   "TEST-NONAME-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Missing required field: categoryId", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name": "Test Product",
			// categoryId is missing
			"baseSku": "TEST-NOCAT-001",
			"variants": []map[string]any{
				{
					"sku":   "TEST-NOCAT-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Missing required field: variants", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-NOVAR-001",
			// variants is missing
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Product name too short (< 3 chars)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "AB", // Only 2 characters
			"categoryId": 4,
			"baseSku":    "TEST-SHORT-001",
			"variants": []map[string]any{
				{
					"sku":   "TEST-SHORT-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Product name too long (> 200 chars)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		// Create a string with 201 characters
		longName := ""
		for i := 0; i < 201; i++ {
			longName += "A"
		}

		requestBody := map[string]any{
			"name":       longName,
			"categoryId": 4,
			"baseSku":    "TEST-LONG-001",
			"variants": []map[string]any{
				{
					"sku":   "TEST-LONG-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Invalid categoryId (non-existent)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product",
			"categoryId": 99999, // Non-existent category
			"baseSku":    "TEST-BADCAT-001",
			"variants": []map[string]any{
				{
					"sku":   "TEST-BADCAT-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Error - Invalid variant data (missing price)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-NOPRICE-001",
			"variants": []map[string]any{
				{
					"sku": "TEST-NOPRICE-001-V1",
					// price is missing
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Invalid variant price (zero)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-ZEROPRICE-001",
			"variants": []map[string]any{
				{
					"sku":   "TEST-ZEROPRICE-001-V1",
					"price": 0, // Zero price
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Invalid variant price (negative)", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-NEGPRICE-001",
			"variants": []map[string]any{
				{
					"sku":   "TEST-NEGPRICE-001-V1",
					"price": -50.00, // Negative price
					"options": []map[string]any{
						{"optionName": "color", "value": "black"},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run(
		"Success - Create product with variant without options (default variant)",
		func(t *testing.T) {
			sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
			client.SetToken(sellerToken)

			requestBody := map[string]any{
				"name":       "Test Product - No Options",
				"categoryId": 4,
				"baseSku":    "TEST-NOOPT-001",
				"variants": []map[string]any{
					{
						"sku":   "TEST-NOOPT-001-V1",
						"price": 99.99,
						// No options - default variant with no variations
					},
				},
			}

			w := client.Post(t, "/api/product", requestBody)
			helpers.AssertSuccessResponse(t, w, http.StatusCreated)
		},
	)

	t.Run("Success - Create product with empty variant options array", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product - Empty Options",
			"categoryId": 4,
			"baseSku":    "TEST-EMPTYOPT-001",
			"variants": []map[string]any{
				{
					"sku":     "TEST-EMPTYOPT-001-V1",
					"price":   99.99,
					"options": []map[string]any{}, // Empty options array - same as no options
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	})

	t.Run("Error - Variant option missing optionName", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-NOOPTNAME-001",
			"variants": []map[string]any{
				{
					"sku":   "TEST-NOOPTNAME-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{
							// optionName is missing
							"value": "black",
						},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error - Variant option missing value", func(t *testing.T) {
		sellerToken := helpers.Login(t, client, helpers.SellerEmail, helpers.SellerPassword)
		client.SetToken(sellerToken)

		requestBody := map[string]any{
			"name":       "Test Product",
			"categoryId": 4,
			"baseSku":    "TEST-NOOPTVAL-001",
			"variants": []map[string]any{
				{
					"sku":   "TEST-NOOPTVAL-001-V1",
					"price": 99.99,
					"options": []map[string]any{
						{
							"optionName": "color",
							// value is missing
						},
					},
				},
			},
		}

		w := client.Post(t, "/api/product", requestBody)
		helpers.AssertErrorResponse(t, w, http.StatusBadRequest)
	})
}
