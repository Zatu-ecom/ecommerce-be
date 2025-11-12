package model

// ProductCreateRequest represents the request body for creating a product
// Note: Product requires at least one variant with price and images
// Frontend handles variant generation from options - backend only saves the final variants
type ProductCreateRequest struct {
	Name             string   `json:"name"             binding:"required,min=3,max=200"`
	CategoryID       uint     `json:"categoryId"       binding:"required"`
	Brand            string   `json:"brand"            binding:"max=100"`
	BaseSKU          string   `json:"baseSku"          binding:"max=50"` // Optional, no validation
	ShortDescription string   `json:"shortDescription" binding:"max=500"`
	LongDescription  string   `json:"longDescription"  binding:"max=5000"`
	Tags             []string `json:"tags"             binding:"max=20"`
	SellerID         *uint    `json:"sellerId"` // Optional: set by backend from auth context this is required in case of admin creates product for a seller

	// Options and Variants
	// Frontend generates variant combinations from options and sends final variants
	Options  []ProductOptionCreateRequest `json:"options"  binding:"dive"` // Product options (color, size, etc.)
	Variants []CreateVariantRequest       `json:"variants" binding:"dive"` // Variants selected by seller (required, min=1)

	// Product attributes and package options
	Attributes     []ProductAttributeRequest `json:"attributes"     binding:"dive"`
	PackageOptions []PackageOptionRequest    `json:"packageOptions" binding:"dive"`
}

// ProductUpdateRequest represents the request body for updating a product
// Note: Price, images, stock are managed at variant level
type ProductUpdateRequest struct {
	Name             string                    `json:"name"             binding:"min=3,max=200"`
	CategoryID       uint                      `json:"categoryId"`
	Brand            string                    `json:"brand"            binding:"max=100"`
	ShortDescription string                    `json:"shortDescription" binding:"max=500"`
	LongDescription  string                    `json:"longDescription"  binding:"max=5000"`
	Tags             []string                  `json:"tags"             binding:"max=20"`
	Attributes       []ProductAttributeRequest `json:"attributes"`
	PackageOptions   []PackageOptionRequest    `json:"packageOptions"`
}

// ProductAttributeRequest represents a product attribute in requests
type ProductAttributeRequest struct {
	Key       string `json:"key"       binding:"required"`
	Name      string `json:"name"      binding:"required"`
	Value     string `json:"value"     binding:"required"`
	Unit      string `json:"unit"`
	SortOrder uint   `json:"sortOrder"`
}

// ProductAttributeResponse represents a product attribute in responses
type ProductAttributeResponse struct {
	ID        uint   `json:"id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	SortOrder uint   `json:"sortOrder"`
}

// PackageOptionRequest represents a package option in requests
type PackageOptionRequest struct {
	Name        string  `json:"name"        binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price"       binding:"required,gt=0"`
	Quantity    int     `json:"quantity"    binding:"required,gt=0"`
}

// PackageOptionResponse represents a package option in responses
type PackageOptionResponse struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// ProductResponse represents the product data returned in API responses
// Now includes variant information since products always have variants
type ProductResponse struct {
	ID               uint                  `json:"id"`
	Name             string                `json:"name"`
	CategoryID       uint                  `json:"categoryId"`
	Category         CategoryHierarchyInfo `json:"category"`
	Brand            string                `json:"brand"`
	SKU              string                `json:"sku"` // Base SKU (PRD Section 3.1)
	ShortDescription string                `json:"shortDescription"`
	LongDescription  string                `json:"longDescription"`
	Tags             []string              `json:"tags"`
	SellerID         uint                  `json:"sellerId"`

	// Variant information (from aggregated variants) for a get all products API
	HasVariants    bool            `json:"hasVariants"`              // Product has variants
	PriceRange     *PriceRange     `json:"priceRange,omitempty"`     // Min and max variant prices
	AllowPurchase  bool            `json:"allowPurchase"`            // At least one variant allows purchase
	Images         []string        `json:"images"`                   // Main product images
	VariantPreview *VariantPreview `json:"variantPreview,omitempty"` // Option preview for listings

	// Detail product info (for get product by ID)
	Attributes     []ProductAttributeResponse    `json:"attributes,omitempty"`
	PackageOptions []PackageOptionResponse       `json:"packageOptions,omitempty"`
	Options        []ProductOptionDetailResponse `json:"options,omitempty"`  // Full options with values (detail view)
	Variants       []VariantDetailResponse       `json:"variants,omitempty"` // Full variants with selected options (detail view)

	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// CategoryInfo represents basic category information
type CategoryInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// CategoryHierarchyInfo represents category information with parent
type CategoryHierarchyInfo struct {
	ID     uint          `json:"id"`
	Name   string        `json:"name"`
	Parent *CategoryInfo `json:"parent"`
}

// ProductsResponse represents the response for getting all products
type ProductsResponse struct {
	Products   []ProductResponse  `json:"products"`
	Pagination PaginationResponse `json:"pagination"`
}

// ProductStockUpdateRequest represents the request body for updating product purchase availability
// Deprecated: Stock management removed, use variant allowPurchase instead
type ProductStockUpdateRequest struct {
	AllowPurchase bool `json:"allowPurchase" binding:"required"`
}

// SearchResult represents a product search result
// Uses struct embedding to extend ProductResponse with search-specific fields
type SearchResult struct {
	ProductResponse          // Embedded - includes all product fields with variants
	RelevanceScore  float64  `json:"relevanceScore"`
	MatchedFields   []string `json:"matchedFields"`
}

// SearchResponse represents the response for product search
type SearchResponse struct {
	Query      string             `json:"query"`
	Results    []SearchResult     `json:"results"`
	Pagination PaginationResponse `json:"pagination"`
	SearchTime string             `json:"searchTime"`
}

// RelatedProductItem represents a related product with relation reason
// Uses struct embedding to extend ProductResponse with additional field
// Any changes to ProductResponse will automatically be available here
type RelatedProductItem struct {
	ProductResponse        // Embedded struct - all fields promoted to top level
	RelationReason  string `json:"relationReason"` // Additional field for relation reason
}

// RelatedProductsResponse represents the response for getting related products
type RelatedProductsResponse struct {
	RelatedProducts []RelatedProductItem `json:"relatedProducts"`
}

// RelatedProductItemScored represents a scored related product with additional metadata
type RelatedProductItemScored struct {
	ProductResponse        // Embedded struct - all fields promoted to top level
	RelationReason  string `json:"relationReason"` // Reason for relation
	Score           int    `json:"score"`          // Relevance score
	StrategyUsed    string `json:"strategyUsed"`   // Strategy that matched this product
}

// RelatedProductsScoredResponse represents the response with scoring and pagination
type RelatedProductsScoredResponse struct {
	RelatedProducts []RelatedProductItemScored `json:"relatedProducts"`
	Pagination      PaginationResponse         `json:"pagination"`
	Meta            RelatedProductsMeta        `json:"meta"`
}

// RelatedProductsMeta represents metadata about the related products query
type RelatedProductsMeta struct {
	StrategiesUsed []string `json:"strategiesUsed"` // List of strategies that found products
	AvgScore       float64  `json:"avgScore"`       // Average relevance score
	TotalStrategies int     `json:"totalStrategies"` // Total strategies attempted
}

// PackageOptionCreateRequest represents the request body for creating a package option
type PackageOptionCreateRequest struct {
	Name        string  `json:"name"        binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price"       binding:"required,gt=0"`
	Quantity    int     `json:"quantity"    binding:"required,gt=0"`
}

// PackageOptionUpdateRequest represents the request body for updating a package option
type PackageOptionUpdateRequest struct {
	Name        string  `json:"name"        binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price"       binding:"required,gt=0"`
	Quantity    int     `json:"quantity"    binding:"required,gt=0"`
}

// PackageOptionsResponse represents the response for getting package options
type PackageOptionsResponse struct {
	PackageOptions []PackageOptionResponse `json:"packageOptions"`
}

// PriceRange represents the minimum and maximum price for a product's variants
type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// OptionPreview represents basic option information for variant preview
type OptionPreview struct {
	Name            string   `json:"name"`
	DisplayName     string   `json:"displayName"`
	AvailableValues []string `json:"availableValues"`
}

// VariantPreview represents summarized variant information for product listings
type VariantPreview struct {
	TotalVariants int             `json:"totalVariants"`
	Options       []OptionPreview `json:"options"`
}
