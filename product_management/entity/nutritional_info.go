package entity

type NutritionalInfo struct {
	Calories    int     `json:"calories"`
	Protein    float64 `json:"protein"`
	Fat       float64 `json:"fat"`
	Carbohydrates float64 `json:"carbohydrates"`
}
