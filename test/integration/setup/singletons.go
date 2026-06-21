package setup

import (
	fileSingleton "ecommerce-be/file/factory/singleton"
	inventorySingleton "ecommerce-be/inventory/factory/singleton"
	orderSingleton "ecommerce-be/order/factory/singleton"
	productSingleton "ecommerce-be/product/factory/singleton"
	promotionSingleton "ecommerce-be/promotion/factory/singleton"
	reportSingleton "ecommerce-be/report/factory/singleton"
	userSingleton "ecommerce-be/user/factory/singleton"
)

// ResetAllModuleSingletons clears module singleton factories so each test server
// wires fresh services against the current DB/Redis clients. Required because Go
// reuses one process across parallel integration packages.
func ResetAllModuleSingletons() {
	userSingleton.ResetInstance()
	productSingleton.ResetInstance()
	inventorySingleton.ResetInstance()
	orderSingleton.ResetInstance()
	promotionSingleton.ResetInstance()
	reportSingleton.ResetInstance()
	fileSingleton.ResetInstance()
}

// ResetProductSingletons resets product module singletons (legacy helper).
func ResetProductSingletons() {
	productSingleton.ResetInstance()
}
