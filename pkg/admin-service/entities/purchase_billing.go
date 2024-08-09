package entities

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"kriyatec.com/go-api/pkg/shared/database"
	"kriyatec.com/go-api/pkg/shared/helper"
)

//shop_supplies

func CreatePurchase(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}

	var req StockPurchase

	var purchaseData map[string]interface{}
	token := helper.GetUserTokenValue(c)

	err := c.BodyParser(&req)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	req.CreatedBy = token.UserId

	req.CreatedOn = time.Now()

	purchaseData = map[string]interface{}{
		"_id":            req.PurchaseId,
		"purchase_date":  time.Now(),
		"supplier_id":    req.SupplierId,
		"invoice_number": req.InvoiceNumber,
		"invoice_date":   req.InvoiceDate,
		"txn_type":       req.TxnType,
	}

	helper.UpdateDateObject(purchaseData)
	if req.TxnType == "ST" {
		ShopTransfer(c, req, orgId)
	}

	_, err = helper.InsertData(c, orgId, "purchase", purchaseData)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	_, err = helper.InsertData(c, orgId, "purchase_details", req)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	return helper.SuccessResponse(c, purchaseData)

}

func CreateBilling(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	var req Billing
	var billingData map[string]interface{}
	token := helper.GetUserTokenValue(c)

	err := c.BodyParser(&req)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	req.CreatedBy = token.UserId
	req.CreatedOn = time.Now()

	billingData = map[string]interface{}{
		"_id":           req.BillNumber,
		"billing_date":  time.Now(),
		"shop_id":       req.ShopId,
		"product_id":    req.ProductId,
		"customer_id":   req.CustomerId,
		"selling_price": req.SellingPrice,
	}
	pipeline := bson.A{
		bson.D{
			{"$match",
				bson.D{
					{"product_id", req.ProductId},
					{"batch_number", req.BatchNumber},
					{"shop_id", req.ShopId},
				},
			},
		},
	}
	availableStockData, err := checkAvailableStock(orgId, pipeline)
	if err != nil {

		return helper.Unexpected(err.Error())

	}

	if len(availableStockData) > 0 {
		updateId := availableStockData[0]["_id"]
		_, err := database.GetConnection(orgId).Collection("shop_inwards").UpdateOne(
			ctx,
			bson.M{"_id": updateId},
			bson.M{"$inc": bson.M{"available_stock": (req.Quantity) * -1}},
			opts,
		)
		if err != nil {
			return err
		}

	} else {

		return helper.Unexpected("No Stock available")

	}

	_, err = helper.InsertData(c, orgId, "billing", billingData)

	if err != nil {
		return helper.BadRequest(err.Error())
	}

	_, err = helper.InsertData(c, orgId, "billing_details", req)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	return helper.SuccessResponse(c, billingData)

}

func ShopTransfer(c *fiber.Ctx, shopdata StockPurchase, orgId string) error {

	value := helper.GetNextSeqNumber(orgId, "shopTransfer")

	shopData := map[string]interface{}{
		"_id":             "ST" + helper.ToString(value),
		"shop_id":         shopdata.ShopId,
		"product_id":      shopdata.ProductId,
		"expiry_date":     shopdata.ExpiryDate,
		"available_stock": shopdata.Quantity,
		"batch_number":    shopdata.BatchNumber,
		"dop":             time.Now(),
	}

	pipeline := bson.A{
		bson.D{
			{"$match",
				bson.D{
					{"product_id", shopdata.ProductId},
					{"batch_number", shopdata.BatchNumber},
				},
			},
		},
	}

	availableStockData, err := checkAvailableStock(orgId, pipeline)
	if err != nil {
		return helper.Unexpected(err.Error())
	}
	if len(availableStockData) > 0 {
		updateId := availableStockData[0]["_id"]
		_, err := database.GetConnection(orgId).Collection("shop_inwards").UpdateOne(
			ctx,
			bson.M{"_id": updateId},
			bson.M{"$inc": bson.M{"available_stock": shopdata.Quantity}},
			opts,
		)
		if err != nil {
			return err
		}
	} else {
		_, err = helper.InsertData(c, orgId, "shop_inwards", shopData)
		if err != nil {
			return helper.BadRequest(err.Error())
		}
	}

	return nil

}

func checkAvailableStock(orgId string, pipeline primitive.A) ([]primitive.M, error) {

	data, err := helper.GetAggregateQueryResult(orgId, "shop_inwards", pipeline)

	if err != nil {
		return nil, err
	}

	return data, nil

}
