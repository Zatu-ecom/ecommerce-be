package handler

import (
	"net/http"

	"ecommerce-be/common/handler"
	"ecommerce-be/report/service"
	"ecommerce-be/report/util"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	*handler.BaseHandler
	reportSvc service.ReportService
}

func NewReportHandler(reportSvc service.ReportService) *ReportHandler {
	return &ReportHandler{
		BaseHandler: handler.NewBaseHandler(),
		reportSvc:   reportSvc,
	}
}

// GetSummary returns high-level summary metrics
func (h *ReportHandler) GetSummary(c *gin.Context) {
	var filter util.ReportQueryFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		h.HandleError(c, err, "Invalid query parameters")
		return
	}

	if c.Query("compare") == "" && filter.Compare == false {
		filter.Compare = true
	}

	res, err := h.reportSvc.GetSummary(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid query filter: " + err.Error(),
		})
		return
	}
	h.Success(c, http.StatusOK, "Success", res)
}

// GetSalesTrends returns time-series data for sales and order volume
func (h *ReportHandler) GetSalesTrends(c *gin.Context) {
	var filter util.ReportQueryFilter

	if err := c.ShouldBindQuery(&filter); err != nil {
		h.HandleError(c, err, "Invalid query parameters")
		return
	}

	res, err := h.reportSvc.GetSalesTrends(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to fetch sales trends: " + err.Error(),
		})
		return
	}
	h.Success(c, http.StatusOK, "Success", res)
}

// GetOrderDistribution returns the breakdown of orders by status
func (h *ReportHandler) GetOrderDistribution(c *gin.Context) {
	h.Success(c, http.StatusOK, "Success", nil)
}

// GetTopSellingProducts returns top selling products
func (h *ReportHandler) GetTopSellingProducts(c *gin.Context) {
	h.Success(c, http.StatusOK, "Success", nil)
}

// GetCustomerRetention returns customer retention data
func (h *ReportHandler) GetCustomerRetention(c *gin.Context) {
	h.Success(c, http.StatusOK, "Success", nil)
}

// GetPromotionPerformance returns promotion usage and performance
func (h *ReportHandler) GetPromotionPerformance(c *gin.Context) {
	h.Success(c, http.StatusOK, "Success", nil)
}
