package entities

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/teris-io/shortid"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"kriyatec.com/go-api/pkg/shared/database"
	"kriyatec.com/go-api/pkg/shared/helper"
)

var opts = options.Update().SetUpsert(true)
var product_collection_name = "product"
var fileUploadPath = helper.GetenvStr("FILE_UPLOAD_PATH", "/uploads")

// postEntitiesHandler - Create Entities
func postDocHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	seq := c.Query("sequence")
	userToken := helper.GetUserTokenValue(c)
	collectionName := c.Params("collectionName")
	// inputData, errorMsg := helper.ValidateInputJson(orgId, collectionName, c.Request().Body(), userToken)
	// if errorMsg != nil {
	// 	return errorMsg
	// }

	// Insert data to collection
	inputData := make(map[string]interface{})

	err := c.BodyParser(&inputData)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	//update date string to time object
	helper.UpdateDateObject(inputData)
	inputData["created_on"] = time.Now()
	inputData["created_by"] = userToken.UserId
	//include default order status
	if collectionName == "order" {
		inputData["status_history"] = []bson.M{
			{"date": time.Now(), "status": "order placed"},
		}
	}

	if collectionName == "user" {
		inputData["pwd"] = helper.PasswordHash(inputData["pwd"].(string))
		// delete(inputData, "pwdConfirm")
	}

	if seq == "true" {
		value := helper.GetNextSeqNumber(orgId, inputData["_id"].(string))
		inputData["_id"] = inputData["_id"].(string) + "-" + helper.ToString(value)
	}

	res, err := helper.InsertData(c, orgId, collectionName, inputData)
	if err == nil && collectionName == "order" {
		go helper.SendOrderSMS(userToken.UserId, inputData)
	}

	return res
	//return nil
}

// putEntitiesHandler - Update Entities
func putDocByIdHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	token := helper.GetUserTokenValue(c)
	filter := helper.DocIdFilter(c.Params("id"))
	collectionName := c.Params("collectionName")
	var inputData map[string]interface{}
	err := c.BodyParser(&inputData)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	//delete the _id field
	delete(inputData, "_id")
	//update date string to time object
	helper.UpdateDateObject(inputData)

	//add updated by and on values
	inputData["updated_on"] = time.Now()
	inputData["updated_by"] = token.UserId
	//Update
	response, err := database.GetConnection(orgId).Collection(collectionName).UpdateOne(
		ctx,
		filter,
		bson.M{"$set": inputData},
		opts,
	)
	if err != nil {
		fmt.Println(err.Error())
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

// putEntitiesHandler - Update Entities
func postArrayEntityByIDHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	token := helper.GetUserTokenValue(c)
	parentId := c.Params("pid")
	arrayDoc := c.Params("child")
	docId := c.Params("cid")
	collectionName := c.Params("collectionName")
	var inputData map[string]interface{}
	err := c.BodyParser(&inputData)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	//find respective entry is there or not
	filter := bson.M{
		"_id":             parentId,
		arrayDoc + "._id": docId,
	}
	var result bson.M
	if err := database.GetConnection(orgId).Collection(collectionName).FindOne(ctx, filter).Decode(&result); err == nil {
		//same entry exists
		return helper.SuccessResponse(c, result)
	}
	//entry not available
	//update date string to time object
	helper.UpdateDateObject(inputData)
	//add updated by and on values
	inputData["updated_on"] = time.Now()
	inputData["updated_by"] = token.UserId
	//Update
	response, err := database.GetConnection(orgId).Collection(collectionName).UpdateOne(
		ctx,
		bson.M{"_id": parentId},
		bson.M{"$addToSet": bson.M{arrayDoc: inputData}},
		opts,
	)
	if err != nil {
		fmt.Println(err.Error())
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

// getProduct Details by its ID
func getDocByIdHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	id := c.Params("id")
	filter := helper.DocIdFilter(id)
	collectionName := c.Params("collectionName")
	fmt.Println("Get API ", collectionName, " by ID ", id)
	response, err := helper.GetQueryResult(orgId, collectionName, filter, int64(0), int64(1), nil)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

// getProduct Details by its ID
func getDocsHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	filter := bson.M{}
	sort := bson.M{}
	page := c.Params("page")
	limit := c.Params("limit")
	sortVal := c.Params("sort")
	collectionName := c.Params("collectionName")
	if sortVal == "" {
		sort = nil
	} else {
		order := helper.SortOrdering(c.Query("order"))
		sort = bson.M{sortVal: order}
	}
	fmt.Println("Get "+collectionName+" Docs method called with page ", page, ", Limit ", limit, ", sort ", sort)
	response, err := helper.GetQueryResult(orgId, collectionName, filter, helper.Page(page), helper.Limit(limit), sort)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

// getProduct Details by its ID
func getDocsByDateHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	page := c.Params("page")
	limit := c.Params("limit")
	//default filter
	filter := bson.M{}
	filterDate, err := time.Parse(time.RFC3339, c.Params("date"))
	if err == nil {
		filter = bson.M{"updated_on": bson.M{"$gte": filterDate}}
	}
	collectionName := c.Params("collectionName")
	response, err := helper.GetQueryResult(orgId, collectionName, filter, helper.Page(page), helper.Limit(limit), nil)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

// getProduct Details by its ID
func getDocsByKeyValueHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")

	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}

	token := helper.GetUserTokenValue(c)
	page := c.Params("page")
	limit := c.Params("limit")
	collectionName := c.Params("collectionName")
	key := c.Params("key")
	value := c.Params("value")

	if value == "_" {
		fmt.Println("_ User Id  ")
		value = token.UserId
		fmt.Println(token.UserId)
	}

	filter := bson.M{key: value}

	response, err := helper.GetQueryResult(orgId, collectionName, filter, helper.Page(page), helper.Limit(limit), nil)

	if err != nil {
		return helper.BadRequest(err.Error())
	}

	return helper.SuccessResponse(c, response)
}

// deleteDocsByIdHandler - Delete Entity By ID
func deleteDocByIdHandler(c *fiber.Ctx) error {
	var filter bson.M
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	value := c.Params("value")
	if value == "_" {
		token := helper.GetUserTokenValue(c)
		value = token.UserId
	}
	collectionName := c.Params("collectionName")
	if c.Params("colName") == "created_by" && collectionName != "shop_cart" {
		filter = bson.M{"user_id": value}
	} else {
		filter = bson.M{c.Params("colName"): value}
	}

	//delete operation only allowed for wishlist and card table
	if collectionName == "wishlist" || collectionName == "shop_cart" || collectionName == "cart" || collectionName == "customer_address" || collectionName == "purchase-invoice" || collectionName == "purchase" || collectionName == "purchase_details" {
		response, err := database.GetConnection(orgId).Collection(collectionName).DeleteMany(ctx, filter)
		if err != nil {
			return helper.BadRequest(err.Error())
		}
		return helper.SuccessResponse(c, response)
	}
	return helper.BadRequest("Delete access has been denied")
}

// Search EntitiesHandler - Get Entities
func searchDocsHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	var collectionName = c.Params("collectionName")
	var conditions []helper.Filter
	err := c.BodyParser(&conditions)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	response, err := helper.GetSearchQueryResult(orgId, collectionName, conditions)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

func textSearchhHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	var collectionName = c.Params("collectionName")
	page := c.Params("page")
	limit := c.Params("limit")
	searchKey, _ := url.QueryUnescape(c.Params("key"))

	regSearch := bson.D{
		{"$regex", primitive.Regex{Pattern: "^" + searchKey}},
		{"$options", "i"},
	}

	filter := bson.M{
		"$or": []bson.M{
			{"name": regSearch},
			{"category_name": regSearch},
			{"desc": regSearch},
		},
	}

	//filter := bson.M{"name": regSearch}
	response, err := helper.GetQueryResult(orgId, collectionName, filter, helper.Page(page), helper.Limit(limit), nil)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	return helper.SuccessResponse(c, response)

}

// Search EntitiesHandler - Get Entities
func DataLookupDocsHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	var lookupQuery helper.LookupQuery
	err := c.BodyParser(&lookupQuery)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	response, err := helper.ExecuteLookupQuery(orgId, lookupQuery)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

// Search EntitiesHandler - Get Entities
func searchEntityWithChildCountHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	var parentCollection = c.Params("parent_collection")
	var keyColumn = c.Params("key_column")
	var childCollection = c.Params("child_collection")
	var lookupColumn = c.Params("lookup_column")
	var conditions []helper.Filter
	err := c.BodyParser(&conditions)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	response, err := helper.GetSearchQueryWithChildCount(orgId, parentCollection, keyColumn, childCollection, lookupColumn, conditions)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

func sharedDBEntityHandler(c *fiber.Ctx) error {
	var collectionName = c.Params("collectionName")
	if collectionName == "db_config" {
		return helper.BadRequest("Access Denied")
	}
	cur, err := database.SharedDB.Collection(collectionName).Find(ctx, bson.D{})
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	var response []bson.M
	if err = cur.All(ctx, &response); err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

// Search EntitiesHandler - Get Entities
func rawQueryHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	var collectionName = c.Params("collectionName")
	var query map[string]interface{}
	err := c.BodyParser(&query)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	helper.UpdateDateObject(query)
	var response []primitive.M
	if c.Params("type") == "aggregate" {
		response, err = helper.GetAggregateQueryResult(orgId, collectionName, query)
	} else {
		response, err = helper.GetQueryResult(orgId, collectionName, query, int64(0), int64(200), nil)
	}
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

func getNextSeqNumberHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	return c.JSON(helper.GetNextSeqNumber(orgId, c.Params("key")))
}

func getPreSignedUploadUrlHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	request := new(PreSignedUploadUrlRequest)
	err := c.BodyParser(request)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	fileName := request.FolderPath + "/" + request.FileKey
	return c.JSON(helper.GetUploadUrl("tpctrz", fileName, request.MetaData))
}

func fileUpload(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	fileCategory := c.Params("category")
	request, err := c.MultipartForm()
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"errors": err.Error()})
	}
	token := helper.GetUserTokenValue(c)
	//check the user folder,
	folderName := fileUploadPath + "/" + token.UserId + "/" + fileCategory
	// if _, err := os.Stat(folderName); os.IsNotExist(err) {
	// 	os.MkdirAll(folderName, 0777)
	// }
	var result []interface{}
	for _, file := range request.File {
		fileExtn := filepath.Ext(file[0].Filename)
		fileName := strings.TrimSuffix(file[0].Filename, fileExtn)
		fileName = fileName + "__" + time.Now().Format("2006-01-02-15-04-05") + fileExtn

		// response := c.SaveFile(file[0], fmt.Sprintf("%s/%s", folderName, fileName))
		// if response != nil {
		// 	return c.Status(422).JSON(fiber.Map{"errors": response.Error()})
		// }

		// Open the file
		openFile, err := file[0].Open()
		if err != nil {
			fmt.Println(err)
		}
		defer openFile.Close()

		// Get the file size
		// fileInfo, err := file.Stat()
		// if err != nil {
		// 	fmt.Println("Error getting file info:", err)

		// }

		buf := bytes.NewBuffer(nil)
		_, err = buf.ReadFrom(openFile)
		if err != nil {
			fmt.Println(err)
		}
		fileLink, err := helper.UploadFile("sakthipharma", folderName, "", "", buf.Bytes())
		if err != nil {
			return helper.Unexpected(err.Error())
		}

		//Save file name to the DB
		id := uuid.New().String()
		orderId := ""
		if len(request.Value["order_id"]) > 0 {
			orderId = request.Value["order_id"][0]
		}

		apiResponse := bson.M{"_id": id, "ref_id": token.UserId, "category": fileCategory, "order_id": orderId, "file_name": file[0].Filename, "storage_name": fileName, "file_path": fileLink, "extn": filepath.Ext(fileName), "size": file[0].Size, "active": "Y"}
		helper.InsertData(c, orgId, "user_files", apiResponse)
		result = append(result, apiResponse)
		//fmt.Printf("User Id %s, File Name:%s, Size:%d", "test", fileName, file[0].Size)
	}
	return helper.SuccessResponse(c, result)
}

func systemFileUpload(c *fiber.Ctx) error {

	orgId := c.Get("OrgId")

	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}

	fileCategory := c.Params("category")

	request, err := c.MultipartForm()

	if err != nil {
		return c.Status(422).JSON(fiber.Map{"errors": err.Error()})
	}

	//check the user folder,
	folderName := fileUploadPath + "/system/" + fileCategory

	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		os.MkdirAll(folderName, 0777)
	}

	var result []interface{}
	for _, file := range request.File {
		fileExtn := filepath.Ext(file[0].Filename)
		fileName := strings.TrimSuffix(file[0].Filename, fileExtn)
		fileName = fileName + "__" + time.Now().Format("2006-01-02-15-04-05") + fileExtn

		openFile, err := file[0].Open()
		if err != nil {
			fmt.Println(err)
		}
		defer openFile.Close()
		buf := bytes.NewBuffer(nil)
		_, err = buf.ReadFrom(openFile)
		if err != nil {
			fmt.Println(err)
		}
		fileLink, err := helper.UploadFile("sakthipharma", folderName, "", "", buf.Bytes())
		if err != nil {
			return helper.Unexpected(err.Error())
		}
		//response := c.SaveFile(file[0], fmt.Sprintf("%s/%s", folderName, fileName))

		// if err != nil {
		// 	return helper.Unexpected(err.Error())

		// }

		//Save file name to the DB
		id := uuid.New().String()
		apiResponse := bson.M{"_id": id, "category": fileCategory, "file_name": file[0].Filename, "file_path": fileLink, "storage_name": fileName, "extn": filepath.Ext(fileName), "size": file[0].Size, "active": "Y"}
		helper.InsertData(c, orgId, "system_files", apiResponse)
		result = append(result, apiResponse)
		//fmt.Printf("User Id %s, File Name:%s, Size:%d", "test", fileName, file[0].Size)
	}
	return helper.SuccessResponse(c, result)
}

func fileDownload(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	fileCategory := c.Params("category")
	name := c.Params("fileName")
	token := helper.GetUserTokenValue(c)
	//check the user folder,
	fileName := fileUploadPath + "/" + token.UserId + "/" + fileCategory + "/" + name
	return c.SendFile(fileName, true)
}

func getFileDetails(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	fileCategory := c.Params("category")
	token := helper.GetUserTokenValue(c)
	query := bson.M{"ref_id": token.UserId, "category": fileCategory}
	response, err := helper.GetQueryResult(orgId, "user_files", query, int64(0), int64(200), nil)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

func getAllFileDetails(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	fileCategory := c.Params("category")
	//status := c.Params("status")
	page := c.Params("page")
	limit := c.Params("limit")
	query := bson.M{"category": fileCategory}
	response, err := helper.GetQueryResult(orgId, "user_files", query, helper.Page(page), helper.Limit(limit), nil)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

func getUpdateDocsHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	var collectionName = c.Params("collectionName")
	date, err := time.Parse(time.RFC3339, c.Params("date"))
	if err == nil {
		date = date.UTC()
	} else {
		return helper.BadRequest(err.Error())
	}
	page := c.Params("page")
	limit := c.Params("limit")
	filter := bson.M{"updated_on": bson.M{"$gte": date}}
	response, err := helper.GetQueryResult(orgId, collectionName, filter, helper.Page(page), helper.Limit(limit), nil)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}

func GetDataByFilterQuery(c *fiber.Ctx) error {

	collectionName := c.Params("collectionName")
	var requestBody helper.PaginationRequest
	var orgID string

	if err := c.BodyParser(&requestBody); err != nil {
		return nil
	}

	orgID = c.Get("Orgid")
	//userToken := helper.GetUserTokenValue(c)

	finalFilter, textSearch := helper.MasterAggreagationPiepline(requestBody, c) //Build the filter

	if requestBody.Start == 0 && requestBody.End == 0 {
		requestBody.Start = 0
		requestBody.End = 50000
	}
	if !textSearch {
		if len(requestBody.Filter) > 0 {
			requestBody.IsGridSearch = true
		} else {
			requestBody.IsGridSearch = false
		}
	}

	//childepipeline is bson.A convert to bson.D

	//pipe := bson.D{
	// {"$facet", bson.D{
	// 	{"response", bson.A{
	// 		bson.D{{"$skip", requestBody.Start}},
	// 		bson.D{{"$limit", requestBody.End - requestBody.Start}},
	// 	}},
	// {"pagination", bson.A{
	// 	bson.D{{"$skip", requestBody.Start}},
	// 	bson.D{{"$limit", requestBody.End - requestBody.Start}},
	// 	bson.D{{"$count", "totalDocs"}},
	// }},
	//	}},
	//}

	childepipeline, flag := childepipeline(collectionName, orgID)
	if !requestBody.IsGridSearch && textSearch {
		if flag {
			for _, stage := range childepipeline {
				finalFilter = append(finalFilter, stage.(primitive.D))
			}
		}
	} else {
		for _, stage := range finalFilter {
			childepipeline = append(childepipeline, stage)
		}
		var finalFilter1 []primitive.D
		for _, stage := range childepipeline {
			finalFilter1 = append(finalFilter1, stage.(primitive.D))
		}
		finalFilter = finalFilter1
	}

	//finalFilter = append(finalFilter, pipe)
	// finalFilter = append(finalFilter, pipe1)
	// finalFilter = append(finalFilter, bson.D{
	// 	{"pagination", bson.A{ // Create an array for "pagination"
	// 		bson.D{
	// 			{"totalDocs", bson.D{{"$size", "$response"}}}, // Calculate the count of the "response" array
	// 		},
	// 	}},
	// })

	results, err := helper.GetAggregateQueryResult(orgID, collectionName, finalFilter)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	// Check if "response" and "pagination" arrays are empty

	if len(results) > 0 {
		responseArray, responseArrayExists := results[0]["response"].(primitive.A)
		paginationArray, paginationArrayExists := results[0]["pagination"].(primitive.A)

		if responseArrayExists && len(responseArray) == 0 || paginationArrayExists && len(paginationArray) == 0 {
			return helper.EntityNotFound("No Data Found")
		}
		if results == nil {
			return helper.EntityNotFound("No Data Found")
		}
	}

	return helper.SuccessResponse(c, results)

}

func childepipeline(collectionName string, orgId string) (bson.A, bool) {

	var childpipeline bson.A
	var flag bool = false

	// if collectionName == "purchase" {
	// 	childpipeline = bson.A{
	// 		bson.D{
	// 			{"$lookup",
	// 				bson.D{
	// 					{"from", "purchase_details"},
	// 					{"localField", "_id"},
	// 					{"foreignField", "purchase_id"},
	// 					{"as", "purchase_details"},
	// 				},
	// 			},
	// 		},
	// 		bson.D{
	// 			{"$unwind",
	// 				bson.D{
	// 					{"path", "$purchase_details"},
	// 					{"preserveNullAndEmptyArrays", true},
	// 				},
	// 			},
	// 		},
	// 		bson.D{
	// 			{"$group",
	// 				bson.D{
	// 					{"_id",
	// 						bson.D{
	// 							{"product_id", "$purchase_details.product_id"},
	// 							{"expiry_date", "$purchase_details.expiry_date"},
	// 						},
	// 					},
	// 					{"availableStock",
	// 						bson.D{
	// 							{"$sum",
	// 								bson.D{
	// 									{"$switch",
	// 										bson.D{
	// 											{"branches",
	// 												bson.A{
	// 													bson.D{
	// 														{"case",
	// 															bson.D{
	// 																{"$eq",
	// 																	bson.A{
	// 																		"$txn_type",
	// 																		"P",
	// 																	},
	// 																},
	// 															},
	// 														},
	// 														{"then", "$purchase_details.quantity"},
	// 													},
	// 													bson.D{
	// 														{"case",
	// 															bson.D{
	// 																{"$eq",
	// 																	bson.A{
	// 																		"$txn_type",
	// 																		"SR",
	// 																	},
	// 																},
	// 															},
	// 														},
	// 														{"then", "$purchase_details.quantity"},
	// 													},
	// 													bson.D{
	// 														{"case",
	// 															bson.D{
	// 																{"$eq",
	// 																	bson.A{
	// 																		"$txn_type",
	// 																		"ST",
	// 																	},
	// 																},
	// 															},
	// 														},
	// 														{"then",
	// 															bson.D{
	// 																{"$multiply",
	// 																	bson.A{
	// 																		"$purchase_details.quantity",
	// 																		-1,
	// 																	},
	// 																},
	// 															},
	// 														},
	// 													},
	// 													bson.D{
	// 														{"case",
	// 															bson.D{
	// 																{"$eq",
	// 																	bson.A{
	// 																		"$txn_type",
	// 																		"R",
	// 																	},
	// 																},
	// 															},
	// 														},
	// 														{"then",
	// 															bson.D{
	// 																{"$multiply",
	// 																	bson.A{
	// 																		"$purchase_details.quantity",
	// 																		-1,
	// 																	},
	// 																},
	// 															},
	// 														},
	// 													},
	// 												},
	// 											},
	// 											{"default", 0},
	// 										},
	// 									},
	// 								},
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		bson.D{
	// 			{"$set",
	// 				bson.D{
	// 					{"_id", "$_id.product_id"},
	// 					{Key: "product_expiry_date", Value: "$_id.expiry_date"},
	// 				},
	// 			},
	// 		},
	// 		bson.D{
	// 			{"$project",
	// 				bson.D{
	// 					{"_id", 1},
	// 					{"availableStock", 1},
	// 					{"product_expiry_date", 1},
	// 				},
	// 			},
	// 		},
	// 	}
	// 	flag = true
	// } else
	if collectionName == "stockDetails" {
		childpipeline = bson.A{
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "purchase"},
						{"localField", "purchase_id"},
						{"foreignField", "_id"},
						{"as", "purchase"},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"txn_type",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$purchase.txn_type",
										0,
									},
								},
							},
						},
						{"purchase_on",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$purchase.invoice_date",
										0,
									},
								},
							},
						},
						{"supplier_id",
							bson.D{
								{"$ifNull",
									bson.A{
										bson.D{
											{"$arrayElemAt",
												bson.A{
													"$purchase.supplier_id",
													0,
												},
											},
										},
										"Unknown",
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$group",
					bson.D{
						{"_id",
							bson.D{
								{"product_id", "$product_id"},
								{"batch_number", "$batch_number"},
								{"expiry_date", "$expiry_date"},
							},
						},
						{"supplier_id", bson.D{{"$first", "$supplier_id"}}},
						{"mrp", bson.D{{"$first", "$mrp"}}},
						{"purchase_on", bson.D{{"$first", "$purchase_on"}}},
						{"available_stock",
							bson.D{
								{"$sum",
									bson.D{
										{"$switch",
											bson.D{
												{"branches",
													bson.A{
														bson.D{
															{"case",
																bson.D{
																	{"$eq",
																		bson.A{
																			"$txn_type",
																			"P",
																		},
																	},
																},
															},
															{"then", "$quantity"},
														},
														bson.D{
															{"case",
																bson.D{
																	{"$eq",
																		bson.A{
																			"$txn_type",
																			"SR",
																		},
																	},
																},
															},
															{"then", "$quantity"},
														},
														bson.D{
															{"case",
																bson.D{
																	{"$eq",
																		bson.A{
																			"$txn_type",
																			"ST",
																		},
																	},
																},
															},
															{"then",
																bson.D{
																	{"$multiply",
																		bson.A{
																			"$quantity",
																			-1,
																		},
																	},
																},
															},
														},
														bson.D{
															{"case",
																bson.D{
																	{"$eq",
																		bson.A{
																			"$txn_type",
																			"R",
																		},
																	},
																},
															},
															{"then",
																bson.D{
																	{"$multiply",
																		bson.A{
																			"$quantity",
																			-1,
																		},
																	},
																},
															},
														},
													},
												},
												{"default", 0},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"_id", "$_id.product_id"},
						{"product_batch_number", "$_id.batch_number"},
						{"product_expiry_date", "$_id.expiry_date"},
					},
				},
			},
			bson.D{
				{"$project",
					bson.D{
						{"_id", 1},
						{"available_stock", 1},
						{"product_batch_number", 1},
						{"supplier_id", 1},
						{"product_expiry_date", 1},
						{"purchase_on", 1},
						{"mrp", 1},
					},
				},
			},
			bson.D{{"$match", bson.D{{"available_stock", bson.D{{"$gt", 0}}}}}},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "product"},
						{"localField", "_id"},
						{"foreignField", "_id"},
						{"as", "product_result"},
					},
				},
			},
			bson.D{
				{"$unwind",
					bson.D{
						{"path", "$product_result"},
						{"preserveNullAndEmptyArrays", true},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"product_name", "$product_result.name"},
						{"product_category", "$product_result.category_name"},
						{"endField", time.Now()},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"dateDiff",
							bson.D{
								{"$dateDiff",
									bson.D{
										{"startDate", "$purchase_on"},
										{"endDate", "$endField"},
										{"unit", "day"},
									},
								},
							},
						},
						{"monthDiff",
							bson.D{
								{"$dateDiff",
									bson.D{
										{"startDate", "$purchase_on"},
										{"endDate", "$endField"},
										{"unit", "month"},
									},
								},
							},
						},
					},
				},
			},
		}

		flag = true
	} else if collectionName == "purchaseDetails" {
		childpipeline = bson.A{
			bson.D{{"$match", bson.D{{"txn_type", "P"}}}},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "supplier"},
						{"localField", "supplier_id"},
						{"foreignField", "supplier_id"},
						{"as", "supplier"},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"parent_id", bson.D{{"$toString", "$_id"}}},
						{"sup_Name",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$supplier.supplier_name",
										0,
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "purchase_details"},
						{"let", bson.D{{"fieldValue", "$parent_id"}}},
						{"pipeline",
							bson.A{
								bson.D{
									{"$match",
										bson.D{
											{"$expr",
												bson.D{
													{"$eq",
														bson.A{
															"$$fieldValue",
															"$purchase_id",
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "product"},
											{"localField", "product_id"},
											{"foreignField", "_id"},
											{"as", "product_resule"},
										},
									},
								},
								bson.D{
									{"$set",
										bson.D{
											{"product_Name",
												bson.D{
													{"$arrayElemAt",
														bson.A{
															"$product_resule.name",
															0,
														},
													},
												},
											},
										},
									},
								},
								bson.D{{"$unset", "product_resule"}},
							},
						},
						{"as", "result"},
					},
				},
			},
			bson.D{{"$unset", "supplier"}},
		}
		flag = true
	} else if collectionName == "shopStockAvailalility" {
		// childpipeline = bson.A{
		// 	bson.D{
		// 		{"$lookup",
		// 			bson.D{
		// 				{"from", "purchase"},
		// 				{"let", bson.D{{"fieldValue", "$_id"}}},
		// 				{"pipeline",
		// 					bson.A{
		// 						bson.D{
		// 							{"$match",
		// 								bson.D{
		// 									{"$expr",
		// 										bson.D{
		// 											{"$eq",
		// 												bson.A{
		// 													"$$fieldValue",
		// 													"$shop_id",
		// 												},
		// 											},
		// 										},
		// 									},
		// 								},
		// 							},
		// 						},
		// 						bson.D{
		// 							{"$lookup",
		// 								bson.D{
		// 									{"from", "purchase_details"},
		// 									{"let", bson.D{{"idString", bson.D{{"$toString", "$_id"}}}}},
		// 									{"pipeline",
		// 										bson.A{
		// 											bson.D{
		// 												{"$match",
		// 													bson.D{
		// 														{"$expr",
		// 															bson.D{
		// 																{"$eq",
		// 																	bson.A{
		// 																		"$purchase_id",
		// 																		"$$idString",
		// 																	},
		// 																},
		// 															},
		// 														},
		// 													},
		// 												},
		// 											},
		// 										},
		// 									},
		// 									{"as", "purchase_details"},
		// 								},
		// 							},
		// 						},
		// 						bson.D{{"$unwind", "$purchase_details"}},
		// 						bson.D{
		// 							{"$lookup",
		// 								bson.D{
		// 									{"from", "product"},
		// 									{"let", bson.D{{"idString", bson.D{{"$toString", "$purchase_details.product_id"}}}}},
		// 									{"pipeline",
		// 										bson.A{
		// 											bson.D{
		// 												{"$match",
		// 													bson.D{
		// 														{"$expr",
		// 															bson.D{
		// 																{"$eq",
		// 																	bson.A{
		// 																		"$_id",
		// 																		"$$idString",
		// 																	},
		// 																},
		// 															},
		// 														},
		// 													},
		// 												},
		// 											},
		// 										},
		// 									},
		// 									{"as", "real_product_details"},
		// 								},
		// 							},
		// 						},
		// 						bson.D{{"$unwind", "$real_product_details"}},
		// 						bson.D{
		// 							{"$set",
		// 								bson.D{
		// 									{"strip_tab_count", "$real_product_details.no_of_tablets_per_strip"},
		// 									{"real_purchase_count", "$purchase_details.quantity"},
		// 								},
		// 							},
		// 						},
		// 						bson.D{
		// 							{"$set",
		// 								bson.D{
		// 									{"purchase_details.quantity",
		// 										bson.D{
		// 											{"$multiply",
		// 												bson.A{
		// 													"$strip_tab_count",
		// 													"$real_purchase_count",
		// 												},
		// 											},
		// 										},
		// 									},
		// 								},
		// 							},
		// 						},
		// 						bson.D{
		// 							{"$group",
		// 								bson.D{
		// 									{"_id",
		// 										bson.D{
		// 											{"product_id", "$purchase_details.product_id"},
		// 											{"batch_number", "$purchase_details.batch_number"},
		// 											{"product_expiry_date", "$purchase_details.expiry_date"},
		// 										},
		// 									},
		// 									{"totalPurchasedQuantity", bson.D{{"$sum", "$purchase_details.quantity"}}},
		// 								},
		// 							},
		// 						},
		// 					},
		// 				},
		// 				{"as", "purchases"},
		// 			},
		// 		},
		// 	},
		// 	bson.D{
		// 		{"$lookup",
		// 			bson.D{
		// 				{"from", "billing"},
		// 				{"let", bson.D{{"fieldValue", "$_id"}}},
		// 				{"pipeline",
		// 					bson.A{
		// 						bson.D{
		// 							{"$match",
		// 								bson.D{
		// 									{"$expr",
		// 										bson.D{
		// 											{"$eq",
		// 												bson.A{
		// 													"$$fieldValue",
		// 													"$shop_id",
		// 												},
		// 											},
		// 										},
		// 									},
		// 								},
		// 							},
		// 						},
		// 						bson.D{
		// 							{"$lookup",
		// 								bson.D{
		// 									{"from", "billing_details"},
		// 									{"let", bson.D{{"idString", bson.D{{"$toString", "$_id"}}}}},
		// 									{"pipeline",
		// 										bson.A{
		// 											bson.D{
		// 												{"$match",
		// 													bson.D{
		// 														{"$expr",
		// 															bson.D{
		// 																{"$eq",
		// 																	bson.A{
		// 																		"$bill_number",
		// 																		"$$idString",
		// 																	},
		// 																},
		// 															},
		// 														},
		// 													},
		// 												},
		// 											},
		// 										},
		// 									},
		// 									{"as", "billing_details"},
		// 								},
		// 							},
		// 						},
		// 						bson.D{{"$unwind", "$billing_details"}},
		// 						bson.D{
		// 							{"$group",
		// 								bson.D{
		// 									{"_id",
		// 										bson.D{
		// 											{"product_id", "$billing_details.product_id"},
		// 											{"batch_number", "$billing_details.batch_number"},
		// 										},
		// 									},
		// 									{"totalSoldQuantity", bson.D{{"$sum", "$billing_details.quantity"}}},
		// 								},
		// 							},
		// 						},
		// 					},
		// 				},
		// 				{"as", "billings"},
		// 			},
		// 		},
		// 	},
		// 	bson.D{
		// 		{"$project",
		// 			bson.D{
		// 				{"products",
		// 					bson.D{
		// 						{"$setUnion",
		// 							bson.A{
		// 								bson.D{
		// 									{"$map",
		// 										bson.D{
		// 											{"input", "$purchases"},
		// 											{"as", "purchase"},
		// 											{"in",
		// 												bson.D{
		// 													{"product_id", "$$purchase._id.product_id"},
		// 													{"batch_number", "$$purchase._id.batch_number"},
		// 													{"product_expiry_date", "$$purchase._id.product_expiry_date"},
		// 												},
		// 											},
		// 										},
		// 									},
		// 								},
		// 								bson.D{
		// 									{"$map",
		// 										bson.D{
		// 											{"input", "$billings"},
		// 											{"as", "billing"},
		// 											{"in",
		// 												bson.D{
		// 													{"product_id", "$$billing._id.product_id"},
		// 													{"batch_number", "$$billing._id.batch_number"},
		// 												},
		// 											},
		// 										},
		// 									},
		// 								},
		// 							},
		// 						},
		// 					},
		// 				},
		// 				{"purchases", 1},
		// 				{"billings", 1},
		// 			},
		// 		},
		// 	},
		// 	bson.D{{"$unwind", "$products"}},
		// 	bson.D{
		// 		{"$addFields",
		// 			bson.D{
		// 				{"totalPurchasedQuantity",
		// 					bson.D{
		// 						{"$ifNull",
		// 							bson.A{
		// 								bson.D{
		// 									{"$arrayElemAt",
		// 										bson.A{
		// 											bson.D{
		// 												{"$filter",
		// 													bson.D{
		// 														{"input", "$purchases"},
		// 														{"as", "purchase"},
		// 														{"cond",
		// 															bson.D{
		// 																{"$and",
		// 																	bson.A{
		// 																		bson.D{
		// 																			{"$eq",
		// 																				bson.A{
		// 																					"$$purchase._id.product_id",
		// 																					"$products.product_id",
		// 																				},
		// 																			},
		// 																		},
		// 																		bson.D{
		// 																			{"$eq",
		// 																				bson.A{
		// 																					"$$purchase._id.batch_number",
		// 																					"$products.batch_number",
		// 																				},
		// 																			},
		// 																		},
		// 																	},
		// 																},
		// 															},
		// 														},
		// 													},
		// 												},
		// 											},
		// 											0,
		// 										},
		// 									},
		// 								},
		// 								bson.D{{"totalPurchasedQuantity", 0}},
		// 							},
		// 						},
		// 					},
		// 				},
		// 				{"totalSoldQuantity",
		// 					bson.D{
		// 						{"$ifNull",
		// 							bson.A{
		// 								bson.D{
		// 									{"$arrayElemAt",
		// 										bson.A{
		// 											bson.D{
		// 												{"$filter",
		// 													bson.D{
		// 														{"input", "$billings"},
		// 														{"as", "billing"},
		// 														{"cond",
		// 															bson.D{
		// 																{"$and",
		// 																	bson.A{
		// 																		bson.D{
		// 																			{"$eq",
		// 																				bson.A{
		// 																					"$$billing._id.product_id",
		// 																					"$products.product_id",
		// 																				},
		// 																			},
		// 																		},
		// 																		bson.D{
		// 																			{"$eq",
		// 																				bson.A{
		// 																					"$$billing._id.batch_number",
		// 																					"$products.batch_number",
		// 																				},
		// 																			},
		// 																		},
		// 																	},
		// 																},
		// 															},
		// 														},
		// 													},
		// 												},
		// 											},
		// 											0,
		// 										},
		// 									},
		// 								},
		// 								bson.D{{"totalSoldQuantity", 0}},
		// 							},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	bson.D{
		// 		{"$set",
		// 			bson.D{
		// 				{"remaining_stock",
		// 					bson.D{
		// 						{"$subtract",
		// 							bson.A{
		// 								"$totalPurchasedQuantity.totalPurchasedQuantity",
		// 								"$totalSoldQuantity.totalSoldQuantity",
		// 							},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	bson.D{
		// 		{"$project",
		// 			bson.D{
		// 				{"_id", "$products.product_id"},
		// 				{"product_id", "$products.product_id"},
		// 				{"product_batch_number", "$products.batch_number"},
		// 				{"product_expiry_date", "$products.product_expiry_date"},
		// 				{"available_stock_tab_count", "$remaining_stock"},
		// 				{"shop_id", "$_id"},
		// 			},
		// 		},
		// 	},
		// 	bson.D{
		// 		{"$lookup",
		// 			bson.D{
		// 				{"from", "product"},
		// 				{"localField", "_id"},
		// 				{"foreignField", "_id"},
		// 				{"as", "product"},
		// 			},
		// 		},
		// 	},
		// 	bson.D{
		// 		{"$unwind",
		// 			bson.D{
		// 				{"path", "$product"},
		// 				{"includeArrayIndex", "string"},
		// 				{"preserveNullAndEmptyArrays", true},
		// 			},
		// 		},
		// 	},
		// 	bson.D{
		// 		{"$set",
		// 			bson.D{
		// 				{"product_name", "$product.name"},
		// 				{"available_stock_strip_count",
		// 					bson.D{
		// 						{"$divide",
		// 							bson.A{
		// 								"$available_stock_tab_count",
		// 								"$product.no_of_tablets_per_strip",
		// 							},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	bson.D{
		// 		{"$group",
		// 			bson.D{
		// 				{"_id",
		// 					bson.D{
		// 						{"product_id", "$_id"},
		// 						{"shop_id", "$shop_id"},
		// 					},
		// 				},
		// 				{"product", bson.D{{"$first", "$product"}}},
		// 				{"shop_id", bson.D{{"$first", "$shop_id"}}},
		// 				{"available_stock_data",
		// 					bson.D{
		// 						{"$push",
		// 							bson.D{
		// 								{"product_batch_number", "$product_batch_number"},
		// 								{"product_expiry_date", "$product_expiry_date"},
		// 								{"available_stock_tab_count", "$available_stock_tab_count"},
		// 								{"available_stock_strip_count", "$available_stock_strip_count"},
		// 							},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// }
		childpipeline = bson.A{
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "purchase"},
						{"let", bson.D{{"fieldValue", "$_id"}}},
						{"pipeline",
							bson.A{
								bson.D{
									{"$match",
										bson.D{
											{"$expr",
												bson.D{
													{"$eq",
														bson.A{
															"$$fieldValue",
															"$shop_id",
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "purchase_details"},
											{"let", bson.D{{"idString", bson.D{{"$toString", "$_id"}}}}},
											{"pipeline",
												bson.A{
													bson.D{
														{"$match",
															bson.D{
																{"$expr",
																	bson.D{
																		{"$eq",
																			bson.A{
																				"$purchase_id",
																				"$$idString",
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
											{"as", "purchase_details"},
										},
									},
								},
								bson.D{{"$unwind", "$purchase_details"}},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "product"},
											{"let", bson.D{{"idString", bson.D{{"$toString", "$purchase_details.product_id"}}}}},
											{"pipeline",
												bson.A{
													bson.D{
														{"$match",
															bson.D{
																{"$expr",
																	bson.D{
																		{"$eq",
																			bson.A{
																				"$_id",
																				"$$idString",
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
											{"as", "real_product_details"},
										},
									},
								},
								bson.D{{"$unwind", "$real_product_details"}},
								bson.D{
									{"$set",
										bson.D{
											{"strip_tab_count", "$real_product_details.no_of_tablets_per_strip"},
											{"real_purchase_count", "$purchase_details.quantity"},
										},
									},
								},
								bson.D{
									{"$set",
										bson.D{
											{"purchase_details.quantity",
												bson.D{
													{"$multiply",
														bson.A{
															"$strip_tab_count",
															"$real_purchase_count",
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$group",
										bson.D{
											{"_id",
												bson.D{
													{"product_id", "$purchase_details.product_id"},
													{"batch_number", "$purchase_details.batch_number"},
													{"product_expiry_date", "$purchase_details.expiry_date"},
												},
											},
											{"totalPurchasedQuantity", bson.D{{"$sum", "$purchase_details.quantity"}}},
										},
									},
								},
							},
						},
						{"as", "purchases"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "billing"},
						{"let", bson.D{{"fieldValue", "$_id"}}},
						{"pipeline",
							bson.A{
								bson.D{
									{"$match",
										bson.D{
											{"$expr",
												bson.D{
													{"$eq",
														bson.A{
															"$$fieldValue",
															"$shop_id",
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "billing_details"},
											{"let", bson.D{{"idString", bson.D{{"$toString", "$_id"}}}}},
											{"pipeline",
												bson.A{
													bson.D{
														{"$match",
															bson.D{
																{"$expr",
																	bson.D{
																		{"$eq",
																			bson.A{
																				"$bill_number",
																				"$$idString",
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
											{"as", "billing_details"},
										},
									},
								},
								bson.D{{"$unwind", "$billing_details"}},
								bson.D{
									{"$group",
										bson.D{
											{"_id",
												bson.D{
													{"product_id", "$billing_details.product_id"},
													{"batch_number", "$billing_details.batch_number"},
													{"product_expiry_date", "$billing_details.expiry_date"},
												},
											},
											{"totalSoldQuantity", bson.D{{"$sum", "$billing_details.quantity"}}},
										},
									},
								},
							},
						},
						{"as", "billings"},
					},
				},
			},
			bson.D{
				{"$project",
					bson.D{
						{"products",
							bson.D{
								{"$setUnion",
									bson.A{
										bson.D{
											{"$map",
												bson.D{
													{"input", "$purchases"},
													{"as", "purchase"},
													{"in",
														bson.D{
															{"product_id", "$$purchase._id.product_id"},
															{"batch_number", "$$purchase._id.batch_number"},
															{"product_expiry_date", "$$purchase._id.product_expiry_date"},
														},
													},
												},
											},
										},
										bson.D{
											{"$map",
												bson.D{
													{"input", "$billings"},
													{"as", "billing"},
													{"in",
														bson.D{
															{"product_id", "$$billing._id.product_id"},
															{"batch_number", "$$billing._id.batch_number"},
															{"product_expiry_date", "$$billing._id.product_expiry_date"},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						{"purchases", 1},
						{"billings", 1},
					},
				},
			},
			bson.D{{"$unwind", "$products"}},
			bson.D{
				{"$addFields",
					bson.D{
						{"totalPurchasedQuantity",
							bson.D{
								{"$ifNull",
									bson.A{
										bson.D{
											{"$arrayElemAt",
												bson.A{
													bson.D{
														{"$filter",
															bson.D{
																{"input", "$purchases"},
																{"as", "purchase"},
																{"cond",
																	bson.D{
																		{"$and",
																			bson.A{
																				bson.D{
																					{"$eq",
																						bson.A{
																							"$$purchase._id.product_id",
																							"$products.product_id",
																						},
																					},
																				},
																				bson.D{
																					{"$eq",
																						bson.A{
																							"$$purchase._id.batch_number",
																							"$products.batch_number",
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
													0,
												},
											},
										},
										bson.D{{"totalPurchasedQuantity", 0}},
									},
								},
							},
						},
						{"totalSoldQuantity",
							bson.D{
								{"$ifNull",
									bson.A{
										bson.D{
											{"$arrayElemAt",
												bson.A{
													bson.D{
														{"$filter",
															bson.D{
																{"input", "$billings"},
																{"as", "billing"},
																{"cond",
																	bson.D{
																		{"$and",
																			bson.A{
																				bson.D{
																					{"$eq",
																						bson.A{
																							"$$billing._id.product_id",
																							"$products.product_id",
																						},
																					},
																				},
																				bson.D{
																					{"$eq",
																						bson.A{
																							"$$billing._id.batch_number",
																							"$products.batch_number",
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
													0,
												},
											},
										},
										bson.D{{"totalSoldQuantity", 0}},
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"remaining_stock",
							bson.D{
								{"$subtract",
									bson.A{
										"$totalPurchasedQuantity.totalPurchasedQuantity",
										"$totalSoldQuantity.totalSoldQuantity",
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$project",
					bson.D{
						{"_id", "$products.product_id"},
						{"product_id", "$products.product_id"},
						{"product_batch_number", "$products.batch_number"},
						{"product_expiry_date", "$products.product_expiry_date"},
						{"available_stock_tab_count", "$remaining_stock"},
						{"shop_id", "$_id"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "product"},
						{"localField", "_id"},
						{"foreignField", "_id"},
						{"as", "product"},
					},
				},
			},
			bson.D{
				{"$unwind",
					bson.D{
						{"path", "$product"},
						{"includeArrayIndex", "string"},
						{"preserveNullAndEmptyArrays", true},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"product_name", "$product.name"},
						{"available_stock_strip_count",
							bson.D{
								{"$divide",
									bson.A{
										"$available_stock_tab_count",
										"$product.no_of_tablets_per_strip",
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$group",
					bson.D{
						{"_id",
							bson.D{
								{"product_id", "$_id"},
								{"shop_id", "$shop_id"},
							},
						},
						{"product", bson.D{{"$first", "$product"}}},
						{"shop_id", bson.D{{"$first", "$shop_id"}}},
						{"available_stock_data",
							bson.D{
								{"$push",
									bson.D{
										{"product_batch_number", "$product_batch_number"},
										{"product_expiry_date", "$product_expiry_date"},
										{"available_stock_tab_count", "$available_stock_tab_count"},
										{"available_stock_strip_count", "$available_stock_strip_count"},
									},
								},
							},
						},
					},
				},
			},
		}
		flag = true
	} else if collectionName == "stockTransferList" {
		childpipeline = bson.A{
			bson.D{{"$match", bson.D{{"txn_type", "ST"}}}},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "shop"},
						{"localField", "shop_id"},
						{"foreignField", "_id"},
						{"as", "shop_result"},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"shop_name",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$shop_result.shop_name",
										0,
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "purchase_details"},
						{"let", bson.D{{"idString", bson.D{{"$toString", "$_id"}}}}},
						{"pipeline",
							bson.A{
								bson.D{
									{"$match",
										bson.D{
											{"$expr",
												bson.D{
													{"$eq",
														bson.A{
															"$purchase_id",
															"$$idString",
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "product"},
											{"localField", "product_id"},
											{"foreignField", "_id"},
											{"as", "product_result"},
										},
									},
								},
								bson.D{
									{"$set",
										bson.D{
											{"name",
												bson.D{
													{"$arrayElemAt",
														bson.A{
															"$product_result.name",
															0,
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$unset",
										bson.A{
											"product_result",
										},
									},
								},
							},
						},
						{"as", "purchase_result"},
					},
				},
			},
			bson.D{{"$unset", "shop_result"}},
		}
		flag = true
	} else if collectionName == "returnList" {
		childpipeline = bson.A{
			bson.D{
				{"$match",
					bson.D{
						{"txn_type",
							bson.D{
								{"$in",
									bson.A{
										"R",
										"SR",
										"STS",
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "purchase_details"},
						{"let", bson.D{{"idString", bson.D{{"$toString", "$_id"}}}}},
						{"pipeline",
							bson.A{
								bson.D{
									{"$match",
										bson.D{
											{"$expr",
												bson.D{
													{"$eq",
														bson.A{
															"$purchase_id",
															"$$idString",
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "product"},
											{"localField", "product_id"},
											{"foreignField", "_id"},
											{"as", "product_result"},
										},
									},
								},
								bson.D{
									{"$set",
										bson.D{
											{"name",
												bson.D{
													{"$arrayElemAt",
														bson.A{
															"$product_result.name",
															0,
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$unset",
										bson.A{
											"product_result",
										},
									},
								},
							},
						},
						{"as", "purchase_result"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "shop"},
						{"localField", "from_shop_id"},
						{"foreignField", "_id"},
						{"as", "from_shop"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "shop"},
						{"localField", "to_shop_id"},
						{"foreignField", "_id"},
						{"as", "to_shop"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "supplier"},
						{"localField", "supplier_id"},
						{"foreignField", "supplier_id"},
						{"as", "supplier"},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"from_shop_name",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$from_shop.shop_name",
										0,
									},
								},
							},
						},
						{"to_shop_name",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$to_shop.shop_name",
										0,
									},
								},
							},
						},
						{"supplier_name",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$supplier.supplier_name",
										0,
									},
								},
							},
						},
						{"transaction_type",
							bson.D{
								{"$cond",
									bson.D{
										{"if",
											bson.D{
												{"$eq",
													bson.A{
														"$txn_type",
														"R",
													},
												},
											},
										},
										{"then", "Return From Warehouse"},
										{"else",
											bson.D{
												{"$cond",
													bson.D{
														{"if",
															bson.D{
																{"$eq",
																	bson.A{
																		"$txn_type",
																		"SR",
																	},
																},
															},
														},
														{"then", "Return From Store"},
														{"else",
															bson.D{
																{"$cond",
																	bson.D{
																		{"if",
																			bson.D{
																				{"$eq",
																					bson.A{
																						"$txn_type",
																						"STS",
																					},
																				},
																			},
																		},
																		{"then", "Store To Store Transfer"},
																		{"else", "$txn_type"},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		flag = true
	} else if collectionName == "dashboardPurchase" {
		childpipeline = bson.A{
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "purchase"},
						{"localField", "purchase_id"},
						{"foreignField", "_id"},
						{"as", "purchase_result"},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"purchased_on",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$purchase_result.invoice_date",
										0,
									},
								},
							},
						},
						{"txn_type",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$purchase_result.txn_type",
										0,
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"formattedPurchaseDate",
							bson.D{
								{"$dateToString",
									bson.D{
										{"format", "%d-%m-%Y"},
										{"date", bson.D{{"$toDate", "$purchased_on"}}},
									},
								},
							},
						},
						{"mrp",
							bson.D{
								{"$multiply",
									bson.A{
										"$quantity",
										"$mrp",
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$match",
					bson.D{
						{"txn_type", "P"},
						{"purchased_on", bson.D{{"$ne", primitive.Null{}}}},
					},
				},
			},
			bson.D{
				{"$group",
					bson.D{
						{"_id", "$formattedPurchaseDate"},
						{"totalPurchaseAmount", bson.D{{"$sum", "$mrp"}}},
					},
				},
			},
			bson.D{{"$set", bson.D{{"purchased_on", bson.D{{"$toDate", "$_id"}}}, {"totalBillingAmount", 0}}}},
		}
		flag = true
	} else if collectionName == "dashboardWareHouseBilling" {
		childpipeline = bson.A{
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "purchase"},
						{"localField", "purchase_id"},
						{"foreignField", "_id"},
						{"as", "purchase_result"},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"purchased_on",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$purchase_result.invoice_date",
										0,
									},
								},
							},
						},
						{"txn_type",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$purchase_result.txn_type",
										0,
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"formattedPurchaseDate",
							bson.D{
								{"$dateToString",
									bson.D{
										{"format", "%d-%m-%Y"},
										{"date", bson.D{{"$toDate", "$purchased_on"}}},
									},
								},
							},
						},
						{"mrp",
							bson.D{
								{"$multiply",
									bson.A{
										"$quantity",
										"$mrp",
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$match",
					bson.D{
						{"txn_type", "ST"},
						{"purchased_on", bson.D{{"$ne", primitive.Null{}}}},
					},
				},
			},
			bson.D{
				{"$group",
					bson.D{
						{"_id", "$formattedPurchaseDate"},
						{"totalBillingAmount", bson.D{{"$sum", "$mrp"}}},
					},
				},
			},
			bson.D{{"$set", bson.D{{"purchased_on", bson.D{{"$toDate", "$_id"}}}, {"totalPurchaseAmount", 0}}}},
		}
		flag = true
	} else if collectionName == "billingDetails" {
		childpipeline = bson.A{
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "billing_details"},
						{"localField", "_id"},
						{"foreignField", "bill_number"},
						{"as", "billing_detail_result"},
					},
				},
			},
		}
		flag = true
	} else if collectionName == "shopTopProduct" {
		childpipeline = bson.A{
			bson.D{
				{"$group",
					bson.D{
						{"_id",
							bson.D{
								{"shop_id", "$shop_id"},
								{"product_id", "$product_id"},
							},
						},
						{"totally_sold", bson.D{{"$sum", "$quantity"}}},
					},
				},
			},
			bson.D{
				{"$project",
					bson.D{
						{"shop_id", "$_id.shop_id"},
						{"product_id", "$_id.product_id"},
						{"totally_sold", 1},
						{"_id", 0},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "product"},
						{"localField", "product_id"},
						{"foreignField", "_id"},
						{"as", "product_result"},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"product_name",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$product_result.name",
										0,
									},
								},
							},
						},
					},
				},
			},
		}
		flag = true
	} else if collectionName == "topSellingProduct" {
		childpipeline = bson.A{
			bson.D{
				{"$group",
					bson.D{
						{"_id", "$product_id"},
						{"product_id", bson.D{{"$first", "$product_id"}}},
						{"totally_sold", bson.D{{"$sum", "$quantity"}}},
					},
				},
			},
			bson.D{
				{"$project",
					bson.D{
						{"product_id", "$product_id"},
						{"totally_sold", 1},
						{"_id", 0},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "product"},
						{"localField", "product_id"},
						{"foreignField", "_id"},
						{"as", "product_result"},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"product_name",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$product_result.name",
										0,
									},
								},
							},
						},
					},
				},
			},
		}
		flag = true
	} else if collectionName == "shopPurchase" {
		childpipeline = bson.A{
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "purchase"},
						{"localField", "purchase_id"},
						{"foreignField", "_id"},
						{"as", "purchase_result"},
					},
				},
			},
			bson.D{
				{"$unwind",
					bson.D{
						{"path", "$purchase_result"},
						{"includeArrayIndex", "string"},
						{"preserveNullAndEmptyArrays", true},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"txn_type", "$purchase_result.txn_type"},
						{"shop_id", "$purchase_result.shop_id"},
						{"purchased_on", "$purchase_result.created_on"},
						{"formated_date",
							bson.D{
								{"$dateToString",
									bson.D{
										{"format", "%d-%m-%Y"},
										{"date", "$purchase_result.created_on"},
									},
								},
							},
						},
					},
				},
			},
			bson.D{{"$match", bson.D{{"txn_type", "ST"}}}},
			bson.D{
				{"$set",
					bson.D{
						{"total_amount",
							bson.D{
								{"$multiply",
									bson.A{
										"$quantity",
										"$mrp",
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$group",
					bson.D{
						{"_id",
							bson.D{
								{"purchased_date", "$formated_date"},
								{"shop_id", "$shop_id"},
							},
						},
						{"purchased_amount", bson.D{{"$sum", "$total_amount"}}},
					},
				},
			},
			bson.D{
				{"$project",
					bson.D{
						{"shop_id", "$_id.shop_id"},
						{"totalPurchaseAmount", "$purchased_amount"},
						{"purchased_on", bson.D{{"$toDate", "$_id.purchased_date"}}},
					},
				},
			},
		}
		flag = true
	} else if collectionName == "shopTotalBilling" {
		childpipeline = bson.A{
			bson.D{
				{"$set",
					bson.D{
						{"formatted_date",
							bson.D{
								{"$dateToString",
									bson.D{
										{"format", "%d-%m-%Y"},
										{"date", "$created_on"},
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$group",
					bson.D{
						{"_id", bson.D{
							{"purchased_on", "$formatted_date"},
							{"shop_id", "$shop_id"},
						},
						},
						{"amount", bson.D{{"$sum", "$amount"}}},
					},
				},
			},
			bson.D{
				{"$project",
					bson.D{
						{"shop_id", "$_id.shop_id"},
						{"totalBillingAmount", "$amount"},
						{"purchased_on", bson.D{{"$toDate", "$_id.purchased_on"}}},
					},
				},
			},
		}
		flag = true
	} else if collectionName == "shopRemainingStockAvailalility" {
		childpipeline = bson.A{
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "purchase"},
						{"let", bson.D{{"fieldValue", "$_id"}}},
						{"pipeline",
							bson.A{
								bson.D{
									{"$match",
										bson.D{
											{"$expr",
												bson.D{
													{"$or",
														bson.A{
															bson.D{
																{"$eq",
																	bson.A{
																		"$$fieldValue",
																		"$shop_id",
																	},
																},
															},
															bson.D{
																{"$eq",
																	bson.A{
																		"$$fieldValue",
																		"$from_shop_id",
																	},
																},
															},
															bson.D{
																{"$eq",
																	bson.A{
																		"$$fieldValue",
																		"$to_shop_id",
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "purchase_details"},
											{"let", bson.D{{"idString", bson.D{{"$toString", "$_id"}}}}},
											{"pipeline",
												bson.A{
													bson.D{
														{"$match",
															bson.D{
																{"$expr",
																	bson.D{
																		{"$eq",
																			bson.A{
																				"$purchase_id",
																				"$$idString",
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
											{"as", "purchase_details"},
										},
									},
								},
								bson.D{{"$unwind", "$purchase_details"}},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "product"},
											{"let", bson.D{{"idString", bson.D{{"$toString", "$purchase_details.product_id"}}}}},
											{"pipeline",
												bson.A{
													bson.D{
														{"$match",
															bson.D{
																{"$expr",
																	bson.D{
																		{"$eq",
																			bson.A{
																				"$_id",
																				"$$idString",
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
											{"as", "real_product_details"},
										},
									},
								},
								bson.D{{"$unwind", "$real_product_details"}},
								bson.D{
									{"$set",
										bson.D{
											{"strip_tab_count", "$real_product_details.no_of_tablets_per_strip"},
											{"real_purchase_count", "$purchase_details.quantity"},
										},
									},
								},
								bson.D{
									{"$set",
										bson.D{
											{"purchase_details.quantity",
												bson.D{
													{"$multiply",
														bson.A{
															"$strip_tab_count",
															"$real_purchase_count",
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$group",
										bson.D{
											{"_id",
												bson.D{
													{"product_id", "$purchase_details.product_id"},
													{"batch_number", "$purchase_details.batch_number"},
													{"product_expiry_date", "$purchase_details.expiry_date"},
												},
											},
											{"totalPurchasedQuantity",
												bson.D{
													{"$sum",
														bson.D{
															{"$switch",
																bson.D{
																	{"branches",
																		bson.A{
																			bson.D{
																				{"case",
																					bson.D{
																						{"$eq",
																							bson.A{
																								"$txn_type",
																								"P",
																							},
																						},
																					},
																				},
																				{"then", "$purchase_details.quantity"},
																			},
																			bson.D{
																				{"case",
																					bson.D{
																						{"$eq",
																							bson.A{
																								"$txn_type",
																								"ST",
																							},
																						},
																					},
																				},
																				{"then", "$purchase_details.quantity"},
																			},
																			bson.D{
																				{"case",
																					bson.D{
																						{"$eq",
																							bson.A{
																								"$txn_type",
																								"SR",
																							},
																						},
																					},
																				},
																				{"then",
																					bson.D{
																						{"$multiply",
																							bson.A{
																								"$purchase_details.quantity",
																								-1,
																							},
																						},
																					},
																				},
																			},
																			bson.D{
																				{"case",
																					bson.D{
																						{"$eq",
																							bson.A{
																								"$txn_type",
																								"R",
																							},
																						},
																					},
																				},
																				{"then",
																					bson.D{
																						{"$multiply",
																							bson.A{
																								"$purchase_details.quantity",
																								-1,
																							},
																						},
																					},
																				},
																			},
																			bson.D{
																				{"case",
																					bson.D{
																						{"$eq",
																							bson.A{
																								"$txn_type",
																								"STS",
																							},
																						},
																					},
																				},
																				{"then",
																					bson.D{
																						{"$cond",
																							bson.D{
																								{"if",
																									bson.D{
																										{"$eq",
																											bson.A{
																												"$$fieldValue",
																												"$from_shop_id",
																											},
																										},
																									},
																								},
																								{"then",
																									bson.D{
																										{"$multiply",
																											bson.A{
																												"$purchase_details.quantity",
																												-1,
																											},
																										},
																									},
																								},
																								{"else", "$purchase_details.quantity"},
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																	{"default", 0},
																},
															},
														},
													},
												},
											},
											{"purchase_on", bson.D{{"$first", "$invoice_date"}}},
										},
									},
								},
							},
						},
						{"as", "purchases"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "billing"},
						{"let", bson.D{{"fieldValue", "$_id"}}},
						{"pipeline",
							bson.A{
								bson.D{
									{"$match",
										bson.D{
											{"$expr",
												bson.D{
													{"$eq",
														bson.A{
															"$$fieldValue",
															"$shop_id",
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "billing_details"},
											{"let", bson.D{{"idString", bson.D{{"$toString", "$_id"}}}}},
											{"pipeline",
												bson.A{
													bson.D{
														{"$match",
															bson.D{
																{"$expr",
																	bson.D{
																		{"$eq",
																			bson.A{
																				"$bill_number",
																				"$$idString",
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
											{"as", "billing_details"},
										},
									},
								},
								bson.D{{"$unwind", "$billing_details"}},
								bson.D{
									{"$group",
										bson.D{
											{"_id",
												bson.D{
													{"product_id", "$billing_details.product_id"},
													{"batch_number", "$billing_details.batch_number"},
												},
											},
											{"totalSoldQuantity", bson.D{{"$sum", "$billing_details.quantity"}}},
										},
									},
								},
							},
						},
						{"as", "billings"},
					},
				},
			},
			bson.D{
				{"$project",
					bson.D{
						{"products",
							bson.D{
								{"$setUnion",
									bson.A{
										bson.D{
											{"$map",
												bson.D{
													{"input", "$purchases"},
													{"as", "purchase"},
													{"in",
														bson.D{
															{"product_id", "$$purchase._id.product_id"},
															{"batch_number", "$$purchase._id.batch_number"},
															{"product_expiry_date", "$$purchase._id.product_expiry_date"},
														},
													},
												},
											},
										},
										bson.D{
											{"$map",
												bson.D{
													{"input", "$billings"},
													{"as", "billing"},
													{"in",
														bson.D{
															{"product_id", "$$billing._id.product_id"},
															{"batch_number", "$$billing._id.batch_number"},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						{"purchases", 1},
						{"billings", 1},
					},
				},
			},
			bson.D{{"$unwind", "$products"}},
			bson.D{
				{"$addFields",
					bson.D{
						{"totalPurchasedQuantity",
							bson.D{
								{"$ifNull",
									bson.A{
										bson.D{
											{"$arrayElemAt",
												bson.A{
													bson.D{
														{"$filter",
															bson.D{
																{"input", "$purchases"},
																{"as", "purchase"},
																{"cond",
																	bson.D{
																		{"$and",
																			bson.A{
																				bson.D{
																					{"$eq",
																						bson.A{
																							"$$purchase._id.product_id",
																							"$products.product_id",
																						},
																					},
																				},
																				bson.D{
																					{"$eq",
																						bson.A{
																							"$$purchase._id.batch_number",
																							"$products.batch_number",
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
													0,
												},
											},
										},
										bson.D{{"totalPurchasedQuantity", 0}},
									},
								},
							},
						},
						{"totalSoldQuantity",
							bson.D{
								{"$ifNull",
									bson.A{
										bson.D{
											{"$arrayElemAt",
												bson.A{
													bson.D{
														{"$filter",
															bson.D{
																{"input", "$billings"},
																{"as", "billing"},
																{"cond",
																	bson.D{
																		{"$and",
																			bson.A{
																				bson.D{
																					{"$eq",
																						bson.A{
																							"$$billing._id.product_id",
																							"$products.product_id",
																						},
																					},
																				},
																				bson.D{
																					{"$eq",
																						bson.A{
																							"$$billing._id.batch_number",
																							"$products.batch_number",
																						},
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
													0,
												},
											},
										},
										bson.D{{"totalSoldQuantity", 0}},
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"remaining_stock",
							bson.D{
								{"$subtract",
									bson.A{
										"$totalPurchasedQuantity.totalPurchasedQuantity",
										"$totalSoldQuantity.totalSoldQuantity",
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$project",
					bson.D{
						{"_id", "$products.product_id"},
						{"product_id", "$products.product_id"},
						{"product_batch_number", "$products.batch_number"},
						{"product_expiry_date", "$products.product_expiry_date"},
						{"available_stock", "$remaining_stock"},
						{"purchase_on", "$totalPurchasedQuantity.purchase_on"},
						{"shop_id", "$_id"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "product"},
						{"localField", "_id"},
						{"foreignField", "_id"},
						{"as", "product"},
					},
				},
			},
			bson.D{
				{"$unwind",
					bson.D{
						{"path", "$product"},
						{"preserveNullAndEmptyArrays", true},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"product_name", "$product.name"},
						{"remaining_strips",
							bson.D{
								{"$divide",
									bson.A{
										"$available_stock",
										"$product.no_of_tablets_per_strip",
									},
								},
							},
						},
						{"endField", time.Now()},
					},
				},
			}, bson.D{
				{"$set",
					bson.D{
						{"dateDiff",
							bson.D{
								{"$dateDiff",
									bson.D{
										{"startDate", "$invoice_date"},
										{"endDate", "$endField"},
										{"unit", "day"},
									},
								},
							},
						},
						{"monthDiff",
							bson.D{
								{"$dateDiff",
									bson.D{
										{"startDate", "$invoice_date"},
										{"endDate", "$endField"},
										{"unit", "month"},
									},
								},
							},
						},
						{"available_stock", "$remaining_strips"},
					},
				},
			},
		}

		flag = true
	} else if collectionName == "purchase_details_list" {
		childpipeline = bson.A{
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "customer"},
						{"localField", "customer_id"},
						{"foreignField", "_id"},
						{"as", "customer_details"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "billing_details"},
						{"let", bson.D{{"fieldValue", "$_id"}}},
						{"pipeline",
							bson.A{
								bson.D{
									{"$match",
										bson.D{
											{"$expr",
												bson.D{
													{"$eq",
														bson.A{
															"$$fieldValue",
															"$bill_number",
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$lookup",
										bson.D{
											{"from", "product"},
											{"localField", "product_id"},
											{"foreignField", "_id"},
											{"as", "product_details"},
										},
									},
								},
								bson.D{
									{"$set",
										bson.D{
											{"productName",
												bson.D{
													{"$arrayElemAt",
														bson.A{
															"$product_details.name",
															0,
														},
													},
												},
											},
										},
									},
								},
								bson.D{
									{"$unset",
										bson.A{
											"product_details",
										},
									},
								},
							},
						},
						{"as", "billing_detail"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "shop"},
						{"localField", "shop_id"},
						{"foreignField", "_id"},
						{"as", "shop_details"},
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "shop_employee"},
						{"localField", "created_by"},
						{"foreignField", "_id"},
						{"as", "shop_employee_details"},
					},
				},
			},
			bson.D{
				{"$set",
					bson.D{
						{"enteredBy",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$shop_employee_details.emp_name",
										0,
									},
								},
							},
						},
						{"shop_Name",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$shop_details.shop_name",
										0,
									},
								},
							},
						},
						{"shop_Contact_Number",
							bson.D{
								{"$arrayElemAt",
									bson.A{
										"$shop_details.mobile_number",
										0,
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				{"$unset",
					bson.A{
						"shop_details",
						"shop_employee_details",
					},
				},
			},
			bson.D{
				{"$lookup",
					bson.D{
						{"from", "shop_invoice"},
						{"localField", "_id"},
						{"foreignField", "bill_no"},
						{"as", "shop_invoice_result"},
					},
				},
			},
			bson.D{
				{"$unwind",
					bson.D{
						{"path", "$shop_invoice_result"},
						{"preserveNullAndEmptyArrays", true},
					},
				},
			},
		}
		flag = true
	}

	// else if collectionName == "shopStockDetails" {
	// 	childpipeline = bson.A{
	// 		bson.D{{"$match", bson.D{{"txn_type", "ST"}}}},
	// 		bson.D{
	// 			{"$lookup",
	// 				bson.D{
	// 					{"from", "shop"},
	// 					{"let", bson.D{{"shopId", "$shop_id"}}},
	// 					{"pipeline",
	// 						bson.A{
	// 							bson.D{
	// 								{"$match",
	// 									bson.D{
	// 										{"$expr",
	// 											bson.D{
	// 												{"$eq",
	// 													bson.A{
	// 														"$$shopId",
	// 														"$_id",
	// 													},
	// 												},
	// 											},
	// 										},
	// 									},
	// 								},
	// 							},
	// 						},
	// 					},
	// 					{"as", "shop_result"},
	// 				},
	// 			},
	// 		},
	// 		bson.D{
	// 			{"$set",
	// 				bson.D{
	// 					{"parent_id", bson.D{{"$toString", "$_id"}}},
	// 					{"shop_name",
	// 						bson.D{
	// 							{"$arrayElemAt",
	// 								bson.A{
	// 									"$shop_result.shop_name",
	// 									0,
	// 								},
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		bson.D{
	// 			{"$lookup",
	// 				bson.D{
	// 					{"from", "purchase_details"},
	// 					{"let", bson.D{{"fieldValue", "$parent_id"}}},
	// 					{"pipeline",
	// 						bson.A{
	// 							bson.D{
	// 								{"$match",
	// 									bson.D{
	// 										{"$expr",
	// 											bson.D{
	// 												{"$eq",
	// 													bson.A{
	// 														"$$fieldValue",
	// 														"$purchase_id",
	// 													},
	// 												},
	// 											},
	// 										},
	// 									},
	// 								},
	// 							},
	// 							bson.D{
	// 								{"$lookup",
	// 									bson.D{
	// 										{"from", "product"},
	// 										{"localField", "product_id"},
	// 										{"foreignField", "_id"},
	// 										{"as", "product_resule"},
	// 									},
	// 								},
	// 							},
	// 							bson.D{
	// 								{"$set",
	// 									bson.D{
	// 										{"product_Name",
	// 											bson.D{
	// 												{"$arrayElemAt",
	// 													bson.A{
	// 														"$product_resule.name",
	// 														0,
	// 													},
	// 												},
	// 											},
	// 										},
	// 									},
	// 								},
	// 							},
	// 							bson.D{{"$unset", "product_resule"}},
	// 						},
	// 					},
	// 					{"as", "result"},
	// 				},
	// 			},
	// 		},
	// 		bson.D{{"$unset", "shop_result"}},
	// 	}
	// 	flag = true
	// }

	return childpipeline, flag
}

func GetDataByFilterQuery1(c *fiber.Ctx) error {

	collectionName := c.Params("collectionName")
	var requestBody helper.PaginationRequest
	var orgID string

	if err := c.BodyParser(&requestBody); err != nil {
		return nil
	}

	orgID = c.Get("Orgid")
	//userToken := helper.GetUserTokenValue(c)

	finalFilter, textSearch := helper.MasterAggreagationPiepline(requestBody, c) //Build the filter

	if requestBody.Start == 0 && requestBody.End == 0 {
		requestBody.Start = 0
		requestBody.End = 50000
	}

	if !textSearch {
		if len(requestBody.Filter) > 0 {
			requestBody.IsGridSearch = true
		} else {
			requestBody.IsGridSearch = false
		}
	}

	//childepipeline is bson.A convert to bson.D

	// pipe := bson.D{
	// 	{"$facet", bson.D{
	// 		{"response", bson.A{
	// 			bson.D{{"$skip", requestBody.Start}},
	// 			bson.D{{"$limit", requestBody.End - requestBody.Start}},
	// 		}},
	// 		{"pagination", bson.A{
	// 			bson.D{{"$skip", requestBody.Start}},
	// 			bson.D{{"$limit", requestBody.End - requestBody.Start}},
	// 			bson.D{{"$count", "totalDocs"}},
	// 		}},
	// 	}},
	// }
	childepipeline, flag := childepipeline(collectionName, orgID)
	if !requestBody.IsGridSearch && textSearch {
		if flag {
			for _, stage := range childepipeline {
				finalFilter = append(finalFilter, stage.(primitive.D))
			}
		}
	} else {
		for _, stage := range finalFilter {
			childepipeline = append(childepipeline, stage)
		}
		var finalFilter1 []primitive.D
		for _, stage := range childepipeline {
			finalFilter1 = append(finalFilter1, stage.(primitive.D))
		}
		finalFilter = finalFilter1
	}

	//finalFilter = append(finalFilter, pipe)
	// finalFilter = append(finalFilter, pipe1)
	// finalFilter = append(finalFilter, bson.D{
	// 	{"pagination", bson.A{ // Create an array for "pagination"
	// 		bson.D{
	// 			{"totalDocs", bson.D{{"$size", "$response"}}}, // Calculate the count of the "response" array
	// 		},
	// 	}},
	// })

	if collectionName == "stockDetails" || collectionName == "dashboardPurchase" || collectionName == "dashboardWareHouseBilling" || collectionName == "shopPurchase" {
		collectionName = "purchase_details"
	} else if collectionName == "purchaseDetails" || collectionName == "shopStockDetails" || collectionName == "stockTransferList" || collectionName == "returnList" {
		collectionName = "purchase"
	} else if collectionName == "shopStockAvailalility" || collectionName == "shopRemainingStockAvailalility" {
		collectionName = "shop"
	} else if collectionName == "billingDetails" || collectionName == "shopTotalBilling" || collectionName == "purchase_details_list" {
		collectionName = "billing"
	} else if collectionName == "shopTopProduct" || collectionName == "topSellingProduct" {
		collectionName = "billing_details"
	}

	results, err := helper.GetAggregateQueryResult(orgID, collectionName, finalFilter)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	// Check if "response" and "pagination" arrays are empty

	if len(results) > 0 {
		responseArray, responseArrayExists := results[0]["response"].(primitive.A)
		paginationArray, paginationArrayExists := results[0]["pagination"].(primitive.A)

		if responseArrayExists && len(responseArray) == 0 || paginationArrayExists && len(paginationArray) == 0 {
			return helper.EntityNotFound("No Data Found")
		}
		if results == nil {
			return helper.EntityNotFound("No Data Found")
		}
	}

	return helper.SuccessResponse(c, results)

}

// var inVoiceMappingKeywords = map[string]interface{}{
// 	"BATCH":          "batch_number",
// 	"CD AMOUNT":      "discount_percentage_amount",
// 	"CD%":            "discount_percentage",
// 	"CGST AMOUNT":    "cgst_amount",
// 	"CGST%":          "cgst_percentage",
// 	"DATE":           "invoice_date",
// 	"EXPDATE":        "expiry_date",
// 	"GST":            "gst_percentage",
// 	"INVNO":          "invoice_number",
// 	"INVOICE AMOUNT": "invoice_amount",
// 	"MRP":            "mrp",
// 	"PCODE":          "pos_id",
// 	"PRATE":          "purchase_rate",
// 	"QTY":            "quantity",
// 	"SGST AMOUNT":    "sgst_amount",
// 	"SGST%":          "sgst_percentage",
// 	"SRATE":          "supply_rate",
// }

var inVoiceMappingKeywordsWithDataType = map[string]map[string]interface{}{
	"BATCH": {
		"field_name": "batch_number",
		"data_type":  "string",
	},
	"CD AMOUNT": {
		"field_name": "discount_percentage_amount",
		"data_type":  "float64",
	},
	"CD%": {
		"field_name": "discount_percentage",
		"data_type":  "float64",
	},
	"CGST AMOUNT": {
		"field_name": "cgst_amount",
		"data_type":  "float64",
	},
	"CGST%": {
		"field_name": "cgst_percentage",
		"data_type":  "float64",
	},
	"DATE": {
		"field_name": "invoice_date",
		"data_type":  "date",
	},
	"EXPDATE": {
		"field_name": "expiry_date",
		"data_type":  "date",
	},
	"GST": {
		"field_name": "gst_percentage",
		"data_type":  "float64",
	},
	"INVNO": {
		"field_name": "invoice_number",
		"data_type":  "string",
	},
	"INVOICE AMOUNT": {
		"field_name": "invoice_amount",
		"data_type":  "float64",
	},
	"MRP": {
		"field_name": "mrp",
		"data_type":  "float64",
	},
	"PCODE": {
		"field_name": "pos_id",
		"data_type":  "string",
	},

	"PRATE": {
		"field_name": "purchase_rate",
		"data_type":  "float64",
	},
	"QTY": {
		"field_name": "quantity",
		"data_type":  "int64",
	},
	"SGST AMOUNT": {
		"field_name": "sgst_amount",
		"data_type":  "float64",
	},
	"SGST%": {
		"field_name": "sgst_percentage",
		"data_type":  "float64",
	},
	"SRATE": {
		"field_name": "supply_rate",
		"data_type":  "float64",
	},
}

func PurchaseInvoiceUpload(c *fiber.Ctx) error {
	supplierId := c.FormValue("ref_id")
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("File not found")
	}

	files := form.File["file"]
	storedData1 := make(map[string][]map[string]interface{})
	for _, file := range files {
		orgID := c.Get("Orgid")
		// Save the uploaded file
		filePath := "./" + file.Filename
		if err := c.SaveFile(file, filePath); err != nil {
			log.Println("Error saving file:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to save file")
		}

		// Open the Excel file
		f, err := excelize.OpenFile(filePath)
		if err != nil {
			log.Println("Error opening Excel file:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to open Excel file")
		}

		// Get all sheet names
		sheetList := f.GetSheetList()
		if len(sheetList) == 0 {
			log.Println("No sheets found in the Excel file")
			return c.Status(fiber.StatusBadRequest).SendString("No sheets found in the Excel file")
		}

		// Read the first sheet
		rows, err := f.GetRows(sheetList[0])
		if err != nil {
			log.Println("Error reading Excel sheet:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to read Excel sheet")
		}

		if len(rows) < 2 {
			log.Println("No data found in the Excel sheet")
			return c.Status(fiber.StatusBadRequest).SendString("No data found in the Excel sheet")
		}

		// Get headers from the first row
		headers := rows[0]
		var allData []map[string]interface{}
		storedData := make(map[string][]map[string]interface{})
		purchaseData := make(map[string]map[string]interface{})
		var purchaseId string

		// Iterate through the rows and insert into MongoDB
		for _, row := range rows[1:] {
			data := make(map[string]interface{})
			var InvoiceNumber string
			for i, cell := range row {
				if i < len(headers) {

					mappedData := inVoiceMappingKeywordsWithDataType[headers[i]]

					// dbField := helper.ToString(inVoiceMappingKeywords[headers[i]])

					// Try to convert to int, if possible

					if mappedData != nil {
						dbField := mappedData["field_name"].(string)
						dataType := mappedData["data_type"].(string)
						// if intValue, err := strconv.Atoi(cell); err == nil {
						// 	data[dbField] = intValue
						// } else {
						// 	data[dbField] = cell
						// }
						data[dbField] = helper.ConvertToDataType(dataType, cell)
						if dbField == "invoice_number" {
							InvoiceNumber = cell
						}
						if dbField == "pos_id" {

							pipeline := bson.A{
								bson.D{{"$match", bson.D{{"pos_id", cell}}}},
							}

							productData, err := helper.GetAggregateQueryResult(orgID, "product", pipeline)

							if err != nil {
								return helper.Unexpected(err.Error())
							}

							if len(productData) > 0 {
								product := productData[0]
								data["product_id"] = product["_id"]
							} else {
								fmt.Println(cell, "No Match")
							}

						}
					}
				}
			}

			var newPurchase bool
			if storedData[InvoiceNumber] != nil {
				getStoredData := storedData[InvoiceNumber]
				data["purchase_id"] = purchaseId
				data["_id"] = helper.GetRandomUUID()
				getStoredData = append(getStoredData, data)
				storedData[InvoiceNumber] = getStoredData
			} else {
				var nowData []map[string]interface{}
				newPurchase = true
				purchaseIdInt := helper.GetNextSeqNumber(orgID, "PURC")
				purchaseId = "PURC-" + helper.ToString(purchaseIdInt)
				data["purchase_id"] = purchaseId
				data["_id"] = helper.GetRandomUUID()
				nowData = append(nowData, data)
				storedData[InvoiceNumber] = nowData
			}
			if newPurchase {

				nowData := map[string]interface{}{
					"invoice_number": InvoiceNumber,
					"purchase_id":    purchaseId,
					"invoice_date":   data["invoice_date"],
					"status":         "Active",
					"supplier_id":    supplierId,
					"txn_type":       "P",
				}

				purchaseData[purchaseId] = nowData

			}

			allData = append(allData, data)
		}

		for purId, rowData := range purchaseData {
			rowData["_id"] = purId
			_, err = helper.InsertData(c, orgID, "purchase", rowData)
			if err != nil {
				return helper.BadRequest(err.Error())
			}
		}

		for _, purchaseDetailsData := range storedData {
			for _, purchaseDetailsData1 := range purchaseDetailsData {
				_, err = helper.InsertData(c, orgID, "purchase_details", purchaseDetailsData1)
				if err != nil {
					return helper.BadRequest(err.Error())
				}
			}
		}

		// Delete the uploaded file after processing
		if err := os.Remove(filePath); err != nil {
			log.Println("Error deleting file:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete file")
		}
		storedData1 = storedData

	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": storedData1, "status": 200, "message": "File Uploaded Successfully"})
}

// func PurchaseInvoiceUpload(c *fiber.Ctx) error {
// 	// file, err := c.FormFile("file")
// 	// if err != nil {
// 	// 	return c.Status(fiber.StatusBadRequest).SendString("File not found")
// 	// }
// 	supplierId := c.FormValue("ref_id")
// 	form, err := c.MultipartForm()
// 	if err != nil {

// 		return c.Status(fiber.StatusBadRequest).SendString("File not found")
// 	}

// 	files := form.File["file"]
// 	storedData1 := make(map[string][]map[string]interface{})
// 	for _, file := range files {
// 		var allData []map[string]interface{}
// 		data := make(map[string]interface{})
// 		storedData := make(map[string][]map[string]interface{})
// 		purchaseData := make(map[string]map[string]interface{})
// 		var purchaseId string
// 		orgID := c.Get("Orgid")
// 		// Save the uploaded file
// 		filePath := "./" + file.Filename
// 		if err := c.SaveFile(file, filePath); err != nil {
// 			log.Println("Error saving file:", err)
// 			return c.Status(fiber.StatusInternalServerError).SendString("Failed to save file")
// 		}

// 		// Open the Excel file
// 		f, err := excelize.OpenFile(filePath)
// 		if err != nil {
// 			log.Println("Error opening Excel file:", err)
// 			return c.Status(fiber.StatusInternalServerError).SendString("Failed to open Excel file")
// 		}

// 		// Get all sheet names
// 		sheetList := f.GetSheetList()
// 		if len(sheetList) == 0 {
// 			log.Println("No sheets found in the Excel file")
// 			return c.Status(fiber.StatusBadRequest).SendString("No sheets found in the Excel file")
// 		}

// 		// Read the first sheet
// 		rows, err := f.GetRows(sheetList[0])
// 		if err != nil {
// 			log.Println("Error reading Excel sheet:", err)
// 			return c.Status(fiber.StatusInternalServerError).SendString("Failed to read Excel sheet")
// 		}

// 		if len(rows) < 2 {
// 			log.Println("No data found in the Excel sheet")
// 			return c.Status(fiber.StatusBadRequest).SendString("No data found in the Excel sheet")
// 		}

// 		// Get headers from the first row
// 		headers := rows[0]
// 		// Iterate through the rows and insert into MongoDB
// 		for _, row := range rows[1:] {
// 			var InvoiceNumber string
// 			for i, cell := range row {
// 				if i < len(headers) {
// 					dbField := helper.ToString(inVoiceMappingKeywords[headers[i]])
// 					// Try to convert to int, if possible
// 					if dbField != "" {
// 						if intValue, err := strconv.Atoi(cell); err == nil {

// 							data[dbField] = intValue
// 						} else {
// 							data[dbField] = cell
// 						}
// 						if dbField == "invoice_number" {
// 							InvoiceNumber = cell
// 						}
// 					}
// 				}
// 			}

// 			var newPurchase bool
// 			if storedData[InvoiceNumber] != nil {
// 				getStoredData := storedData[InvoiceNumber]
// 				data["purchase_id"] = purchaseId
// 				data["_id"] = helper.GetRandomUUID()
// 			//	fmt.Println(data["_id"])
// 				getStoredData = append(getStoredData, data)
// 				storedData[InvoiceNumber] = getStoredData
// 			} else {
// 				var nowData []map[string]interface{}
// 				newPurchase = true
// 				purchaseIdInt := helper.GetNextSeqNumber(orgID, "PURC")
// 				purchaseId = "PURC-" + helper.ToString(purchaseIdInt)
// 				data["purchase_id"] = purchaseId
// 				data["_id"] = helper.GetRandomUUID()
// 				nowData = append(nowData, data)
// 				storedData[InvoiceNumber] = nowData
// 			}
// 			if newPurchase {
// 				nowData := map[string]interface{}{
// 					"invoice_number": InvoiceNumber,
// 					"purchase_id":    purchaseId,
// 					"invoice_date":   data["invoice_date"],
// 					"status":         "Active",
// 					"supplier_id":    supplierId,
// 					"txn_type":       "P",
// 				}
// 				purchaseData[purchaseId] = nowData
// 			}
// 			allData = append(allData, data)
// 			// _, err = collection.InsertOne(context.TODO(), data)
// 			// if err != nil {
// 			// 	log.Println("Error inserting data into MongoDB:", err)
// 			// 	return c.Status(fiber.StatusInternalServerError).SendString("Failed to insert data into MongoDB")
// 			// }
// 		}

// 		for purId, rowData := range purchaseData {
// 			rowData["_id"] = purId
// 			_, err = helper.InsertData(c, orgID, "purchase", rowData)
// 			if err != nil {
// 				return helper.BadRequest(err.Error())
// 			}
// 		}

// 		for _, purchaseDetailsData := range storedData {
// 			for _, purchaseDetailsData1 := range purchaseDetailsData {
// 				_, err = helper.InsertData(c, orgID, "purchase_details", purchaseDetailsData1)
// 				if err != nil {
// 					return helper.BadRequest(err.Error())
// 				}
// 			}
// 		}

// 		// Delete the uploaded file after processing
// 		if err := os.Remove(filePath); err != nil {
// 			log.Println("Error deleting file:", err)
// 			return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete file")
// 		}
// 		storedData1 = storedData
// 	}
// 	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": storedData1, "status": 200, "message": "File Uploaded Successfully"})
// }

// func PurchaseInvoiceUpload(c *fiber.Ctx) error {
// 	file, err := c.FormFile("file")
// 	if err != nil {
// 		return c.Status(fiber.StatusBadRequest).SendString("File not found")
// 	}
// 	orgID := c.Get("Orgid")
// 	filePath := "./" + file.Filename

// 	if err := c.SaveFile(file, filePath); err != nil {
// 		log.Println("Error saving file:", err)
// 		return c.Status(fiber.StatusInternalServerError).SendString("Failed to save file")
// 	}

// 	f, err := excelize.OpenFile(filePath)
// 	if err != nil {
// 		log.Println("Error opening Excel file:", err)
// 		return c.Status(fiber.StatusInternalServerError).SendString("Failed to open Excel file")
// 	}

// 	sheetList := f.GetSheetList()
// 	if len(sheetList) == 0 {
// 		log.Println("No sheets found in the Excel file")
// 		return c.Status(fiber.StatusBadRequest).SendString("No sheets found in the Excel file")
// 	}

// 	rows, err := f.GetRows(sheetList[0])
// 	if err != nil {
// 		log.Println("Error reading Excel sheet:", err)
// 		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read Excel sheet")
// 	}

// 	if len(rows) < 2 {
// 		log.Println("No data found in the Excel sheet")
// 		return c.Status(fiber.StatusBadRequest).SendString("No data found in the Excel sheet")
// 	}

// 	headers := rows[0]
// 	storedData := make(map[string][]map[string]interface{})
// 	purchaseData := make(map[string]map[string]interface{})

// 	for _, row := range rows[1:] {
// 		data := make(map[string]interface{})
// 		var invoiceNumber string

// 		for i, cell := range row {
// 			if i < len(headers) {
// 				dbField := helper.ToString(inVoiceMappingKeywords[headers[i]])
// 				if dbField != "" {
// 					if intValue, err := strconv.Atoi(cell); err == nil {
// 						data[dbField] = intValue
// 					} else {
// 						data[dbField] = cell
// 					}
// 					if dbField == "invoice_number" {
// 						invoiceNumber = cell
// 					}
// 				}
// 			}
// 		}

// 		if invoiceNumber == "" {
// 			continue
// 		}

// 		var purchaseId string
// 		if _, exists := storedData[invoiceNumber]; exists {
// 			data["purchase_id"] = purchaseId
// 			storedData[invoiceNumber] = append(storedData[invoiceNumber], data)
// 		} else {
// 			purchaseIdInt := helper.GetNextSeqNumber(orgID, "PURC")
// 			purchaseId = "PURC-" + helper.ToString(purchaseIdInt)
// 			data["purchase_id"] = purchaseId
// 			storedData[invoiceNumber] = []map[string]interface{}{data}
// 			purchaseData[purchaseId] = map[string]interface{}{
// 				"invoice_number": invoiceNumber,
// 				"purchase_id":    purchaseId,
// 				"invoice_date":   data["invoice_date"],
// 				"status":         "Active",
// 				"txn_type":       "P",
// 			}
// 		}
// 	}

// 	for purId, rowData := range purchaseData {
// 		rowData["_id"] = purId
// 		if _, err := helper.InsertData(c, orgID, "purchase", rowData); err != nil {
// 			return helper.BadRequest(err.Error())
// 		}
// 	}

// 	for _, details := range storedData {
// 		for _, detail := range details {
// 			if _, err := helper.InsertData(c, orgID, "purchase_details", detail); err != nil {
// 				return helper.BadRequest(err.Error())
// 			}
// 		}
// 	}

// 	return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": storedData, "len": len(storedData)})
// }

func GenerateInvoice(c *fiber.Ctx) error {

	billNumber := c.Params("BillNumber")
	orgID := c.Get("Orgid")
	userToken := helper.GetUserTokenValue(c)
	collectionName := "billing"
	billNumberFilter := bson.A{
		bson.D{{"$match", bson.D{{"bill_no", billNumber}}}},
	}
	data, err := helper.GetAggregateQueryResult(orgID, "shop_invoice", billNumberFilter)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	if len(data) > 0 {

		return helper.SuccessResponse(c, data[0])
	}

	pipeline := bson.A{
		bson.D{{"$match", bson.D{{"_id", billNumber}}}},
		bson.D{
			{"$lookup",
				bson.D{
					{"from", "billing_details"},
					{"let", bson.D{{"bill_id", "$_id"}}},
					{"pipeline",
						bson.A{
							bson.D{
								{"$match",
									bson.D{
										{"$expr",
											bson.D{
												{"$eq",
													bson.A{
														"$$bill_id",
														"$bill_number",
													},
												},
											},
										},
									},
								},
							},
							bson.D{
								{"$lookup",
									bson.D{
										{"from", "product"},
										{"let", bson.D{{"productId", bson.D{{"$toString", "$product_id"}}}}},
										{"pipeline",
											bson.A{
												bson.D{
													{"$match",
														bson.D{
															{"$expr",
																bson.D{
																	{"$eq",
																		bson.A{
																			"$_id",
																			"$$productId",
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
										{"as", "purchase_details"},
									},
								},
							},
							bson.D{{"$unwind", "$purchase_details"}},
							bson.D{{"$set", bson.D{{"hsnc_number", "$purchase_details.hsnc"}, {"formattedDate",
								bson.D{
									{"$dateToString",
										bson.D{
											{"format", "%d-%m-%Y"},
											{"date", bson.D{{"$toDate", "$expiry_date"}}},
										},
									},
								},
							}}}},
							bson.D{
								{"$project",
									bson.D{
										{"#", ""},
										{"HSNC", "$hsnc_number"},
										{"Description",
											bson.D{
												{"$concat",
													bson.A{
														"$purchase_details.name",
														"\n batch No : ",
														"$batch_number",
														"\n expiry Date : ",
														"$formattedDate",
													},
												},
											},
										},
										{"MRP", "$mrp"},
										{"Price", "$amount_without_gst"},
										{"Qty", "$quantity"},
										{"GST%", "$gst_percentage"},
										{"GST", "$amount"},
										{"Amount", "$amount"},
										{"Total", "$amount"},
									},
								},
							},
						},
					},
					{"as", "billing_details_result"},
				},
			},
		},
		bson.D{
			{"$lookup",
				bson.D{
					{"from", "shop"},
					{"localField", "shop_id"},
					{"foreignField", "_id"},
					{"as", "shop_result"},
				},
			},
		},
		bson.D{
			{"$unwind",
				bson.D{
					{"path", "$shop_result"},
					{"preserveNullAndEmptyArrays", true},
				},
			},
		},
		bson.D{
			{"$lookup",
				bson.D{
					{"from", "customer"},
					{"localField", "customer_id"},
					{"foreignField", "_id"},
					{"as", "customer_result"},
				},
			},
		},
		bson.D{
			{"$unwind",
				bson.D{
					{"path", "$customer_result"},
					{"preserveNullAndEmptyArrays", true},
				},
			},
		},
	}

	results, err := helper.GetAggregateQueryResult(orgID, collectionName, pipeline)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	payload, err := helper.GenerateInvoicePDF(results[0], orgID)
	if err != nil {
		return helper.Unexpected(err.Error())
	}

	payload["created_on"] = time.Now()
	payload["created_by"] = userToken.UserId
	payload["bill_no"] = billNumber

	helper.InsertData(c, orgID, "shop_invoice", payload)

	return helper.SuccessResponse(c, payload)
}

var s3Client *s3.S3

func S3Upload(c *fiber.Ctx) error {
	InitS3Client()

	file, err := c.FormFile("file")
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	res, err := helper.S3PdfFileUpload(s3Client, "uploads", file.Filename)
	if err != nil {
		return helper.BadRequest(err.Error())
	}

	return helper.SuccessResponse(c, res)

}

func InitS3Client() {

	var api_key = os.Getenv("S3_API_KEY")
	var secret = os.Getenv("S3_SECRET")
	var endpoint = os.Getenv("S3_ENDPOINT")
	var region = os.Getenv("S3_REGION")

	var s3Config = &aws.Config{
		Credentials:      credentials.NewStaticCredentials(api_key, secret, ""),
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(false),
	}
	var newSession = session.New(s3Config)
	s3Client = s3.New(newSession)

}

func CreateS3Bucket(c *fiber.Ctx) error {
	bucketName := c.Params("name")
	create := helper.CreateBucket(bucketName)
	if !create {
		return helper.Unexpected("Error in creating bucket")
	}

	return helper.SuccessResponse(c, create)
}

func SendSMOtp(c *fiber.Ctx) error {

	res := helper.SendSOTP("6385719863", "")

	return helper.SuccessResponse(c, res)
}

// type URL struct {
// 	OriginalURL string    `json:"original_url" bson:"original_url"`
// 	Id          string    `json:"_id" bson:"_id"`
// 	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
// }

type URL struct {
	OriginalURL string    `json:"original_url" bson:"original_url"`
	Id          string    `json:"_id" bson:"_id"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

func ShortenURL(c *fiber.Ctx) error {
	type Request struct {
		OriginalURL string `json:"original_url"`
	}
	orgId := c.Get("Orgid")
	var request Request
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	shortURL, err := shortid.Generate()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	newURL := URL{
		OriginalURL: request.OriginalURL,
		Id:          shortURL,
		CreatedAt:   time.Now(),
	}
	_, err = helper.InsertData(c, orgId, "shorten_url", newURL)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot store URL"})
	}

	return c.JSON(fiber.Map{"original_url": request.OriginalURL, "short_url": shortURL})
}

func RedirectURL(c *fiber.Ctx) error {
	//shortURL := c.Params("shortURL")

	var result []primitive.M

	// orgId := c.Get("OrgId")
	// if orgId == "" {
	// 	return helper.BadRequest("Organization Id missing")
	// }
	id := c.Params("id")
	filter := helper.DocIdFilter(id)
	result, err := helper.GetQueryResult("kt", "shorten_url", filter, int64(0), int64(1), nil)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	if len(result) == 0 {
		return helper.Unexpected("No content found")
	}

	resultMap := result[0]
	url := resultMap["original_url"].(string)
	fmt.Println(url)
	return c.Redirect(url, fiber.StatusMovedPermanently)
}
