package entity

type PackageOption struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	ProductID  uint   `json:"productId" gorm:"index"`
	Name       string `json:"name" binding:"required"`
	Price     float64 `json:"price" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required"`
}
