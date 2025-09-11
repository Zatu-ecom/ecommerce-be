package model

// ProductCreateRequest represents the request body for creating a product
type ProductCreateRequest struct {
	Name             string                    `json:"name" binding:"required,min=3,max=200"`
	CategoryID       uint                      `json:"categoryId" binding:"required"`
	Brand            string                    `json:"brand" binding:"max=100"`
	SKU              string                    `json:"sku" binding:"required,min=3,max=50"`
	Price            float64                   `json:"price" binding:"required,gt=0"`
	Currency         string                    `json:"currency" binding:"len=3"`
	ShortDescription string                    `json:"shortDescription" binding:"max=500"`
	LongDescription  string                    `json:"longDescription" binding:"max=5000"`
	Images           []string                  `json:"images" binding:"max=10"`
	IsPopular        bool                      `json:"isPopular"`
	Discount         int                       `json:"discount" binding:"min=0,max=100"`
	Tags             []string                  `json:"tags" binding:"max=20"`
	Attributes       []ProductAttributeRequest `json:"attributes" binding:"required"`
	PackageOptions   []PackageOptionRequest    `json:"packageOptions"`
}

// ProductUpdateRequest represents the request body for updating a product
type ProductUpdateRequest struct {
	Name             string                    `json:"name" binding:"min=3,max=200"`
	CategoryID       uint                      `json:"categoryId"`
	Brand            string                    `json:"brand" binding:"max=100"`
	Price            float64                   `json:"price" binding:"gt=0"`
	Currency         string                    `json:"currency" binding:"len=3"`
	ShortDescription string                    `json:"shortDescription" binding:"max=500"`
	LongDescription  string                    `json:"longDescription" binding:"max=5000"`
	Images           []string                  `json:"images" binding:"max=10"`
	IsPopular        bool                      `json:"isPopular"`
	Discount         int                       `json:"discount" binding:"min=0,max=100"`
	Tags             []string                  `json:"tags" binding:"max=20"`
	Attributes       []ProductAttributeRequest `json:"attributes"`
	PackageOptions   []PackageOptionRequest    `json:"packageOptions"`
}

// ProductAttributeRequest represents a product attribute in requests
type ProductAttributeRequest struct {
	Key       string `json:"key" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Value     string `json:"value" binding:"required"`
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
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Quantity    int     `json:"quantity" binding:"required,gt=0"`
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
type ProductResponse struct {
	ID               uint                       `json:"id"`
	Name             string                     `json:"name"`
	CategoryID       uint                       `json:"categoryId"`
	Category         CategoryHierarchyInfo      `json:"category"`
	Brand            string                     `json:"brand"`
	SKU              string                     `json:"sku"`
	Price            float64                    `json:"price"`
	Currency         string                     `json:"currency"`
	ShortDescription string                     `json:"shortDescription"`
	LongDescription  string                     `json:"longDescription"`
	Images           []string                   `json:"images"`
	InStock          bool                       `json:"inStock"`
	IsPopular        bool                       `json:"isPopular"`
	Discount         int                        `json:"discount"`
	Tags             []string                   `json:"tags"`
	Attributes       []ProductAttributeResponse `json:"attributes"`
	PackageOptions   []PackageOptionResponse    `json:"packageOptions"`
	CreatedAt        string                     `json:"createdAt"`
	UpdatedAt        string                     `json:"updatedAt"`
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

// ProductStockUpdateRequest represents the request body for updating product stock
type ProductStockUpdateRequest struct {
	InStock bool `json:"inStock" binding:"required"`
}

// SearchResult represents a product search result
type SearchResult struct {
	ID               uint     `json:"id"`
	Name             string   `json:"name"`
	Price            float64  `json:"price"`
	ShortDescription string   `json:"shortDescription"`
	Images           []string `json:"images"`
	RelevanceScore   float64  `json:"relevanceScore"`
	MatchedFields    []string `json:"matchedFields"`
}

// SearchResponse represents the response for product search
type SearchResponse struct {
	Query      string             `json:"query"`
	Results    []SearchResult     `json:"results"`
	Pagination PaginationResponse `json:"pagination"`
	SearchTime string             `json:"searchTime"`
}

// RelatedProductResponse represents a related product
type RelatedProductResponse struct {
	ID               uint     `json:"id"`
	Name             string   `json:"name"`
	Price            float64  `json:"price"`
	ShortDescription string   `json:"shortDescription"`
	Images           []string `json:"images"`
	RelationReason   string   `json:"relationReason"`
}

// RelatedProductsResponse represents the response for getting related products
type RelatedProductsResponse struct {
	RelatedProducts []RelatedProductResponse `json:"relatedProducts"`
}

// PackageOptionCreateRequest represents the request body for creating a package option
type PackageOptionCreateRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Quantity    int     `json:"quantity" binding:"required,gt=0"`
}

// PackageOptionUpdateRequest represents the request body for updating a package option
type PackageOptionUpdateRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Quantity    int     `json:"quantity" binding:"required,gt=0"`
}

// PackageOptionsResponse represents the response for getting package options
type PackageOptionsResponse struct {
	PackageOptions []PackageOptionResponse `json:"packageOptions"`
}
