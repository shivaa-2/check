package entities

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"

	"kriyatec.com/go-api/pkg/shared/database"
	"kriyatec.com/go-api/pkg/shared/helper"
)

func saveOrder(c *fiber.Ctx) error {
	return postDocHandler(c)
	//Adjust Stock
}

func createOrder(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	var request map[string]interface{}
	err := c.BodyParser(&request)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	//add order meta data
	request["order_meta"] = helper.GetOrderMeta()
	//interface to []byte
	r, err := json.Marshal(request)
	fmt.Println(string(r))
	response, err := helper.CreateOrder(r)
	response["_id"] = uuid.New().String()
	response["txn_type"] = "order"
	response["mode"] = "web"
	helper.InsertData(c, orgId, "payment_details", response)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

func refundOrder(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	orderId := c.Params("order_id")
	response, err := helper.RefundOrder(orderId, c.Body())
	response["_id"] = uuid.New().String()
	response["txn_type"] = "refund"
	response["mode"] = "web"
	helper.InsertData(c, orgId, "payment_details", response)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

func orderStatusUpdate(c *fiber.Ctx) error {
	r := string(c.Body())
	fmt.Println(r)
	return c.SendString("OK")
}

func getCreatePaymentOrder(c *fiber.Ctx) error {
	var inputData helper.OrderRequest
	err := c.BodyParser(&inputData)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	link, err := helper.OCreateOrder(inputData)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, link)
}

func createOrderNewVersion(c *fiber.Ctx) error {
	response, err := helper.CreateOrderOnUpdatedCashFreeVersion(c.Body())
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	return helper.SuccessResponse(c, response)
}

func getPaymentToken(c *fiber.Ctx) error {
	response, err := helper.GetPaymentToken(c.Body())
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

func statusUpdate(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	filter := helper.DocIdFilter(c.Params("id"))
	var collectionName = "order"
	var inputData map[string]interface{}
	err := c.BodyParser(&inputData)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	//get Order Details
	var order bson.M
	result := database.GetConnection(orgId).Collection(collectionName).FindOne(ctx, filter)
	err = result.Decode(&order)
	if err != nil {
		fmt.Println(err.Error())
	}
	userName := order["name"].(string)
	mobileNo := order["created_by"].(string)
	inputData["date"] = time.Now()

	//update date string to time object
	helper.UpdateDateObject(inputData)
	query := bson.M{
		"$addToSet": bson.M{
			"status_history": inputData,
		},
	}
	//Update
	response, err := database.GetConnection(orgId).Collection(collectionName).UpdateOne(
		ctx,
		filter,
		query,
		opts,
	)
	if err != nil {
		fmt.Println(err.Error())
		return helper.BadRequest(err.Error())
	}

	//get Order Details
	if inputData["status"] == "out for delivery" {
		helper.SendTakenForDeliverySMS(mobileNo, c.Params("id"), userName)
	} else if inputData["status"] == "delivered" {
		helper.SendDeliverySMS(mobileNo, c.Params("id"), userName)
	}
	return helper.SuccessResponse(c, response)
	//Adjust Stock
}

func updatePaymentStatus(c *fiber.Ctx) error {

	return helper.SuccessResponse(c, "payment detail")
}
