package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"kriyatec.com/go-api/pkg/shared/database"
)

var updateOpts = options.Update().SetUpsert(true)
var findUpdateOpts = options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
var ctx = context.Background()

func InsertData(c *fiber.Ctx, orgId string, collectionName string, data interface{}) (error, error) {
	response, err := database.GetConnection(orgId).Collection(collectionName).InsertOne(ctx, data)
	if err != nil {
		return BadRequest(err.Error()), err
	}
	return SuccessResponse(c, response), nil
}

func GetAggregateQueryResult(orgId string, collectionName string, query interface{}) ([]bson.M, error) {
	response, err := ExecuteAggregateQuery(orgId, collectionName, query)
	if err != nil {
		return nil, err
	}
	var result []bson.M
	//var result map[string][]Config
	if err = response.All(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func ExecuteAggregateQuery(orgId string, collectionName string, query interface{}) (*mongo.Cursor, error) {
	cur, err := database.GetConnection(orgId).Collection(collectionName).Aggregate(ctx, query)
	if err != nil {
		return nil, err
	}
	return cur, nil
}

func GetQueryResult(orgId string, collectionName string, query interface{}, page int64, limit int64, sort interface{}) ([]bson.M, error) {
	response, err := ExecuteQuery(orgId, collectionName, query, page, limit, sort)
	if err != nil {
		return nil, err
	}
	var result []bson.M
	//var result map[string][]Config
	if err = response.All(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func ExecuteQuery(orgId string, collectionName string, query interface{}, page int64, limit int64, sort interface{}) (*mongo.Cursor, error) {
	pageOptions := options.Find()
	skip := int64(0)
	if page > 0 {
		skip = (page - int64(1)) * limit
	}
	pageOptions.SetSkip(skip)   //0-i
	pageOptions.SetLimit(limit) // number of records to return
	if sort != nil {
		pageOptions.Sort = sort
	}
	response, err := database.GetConnection(orgId).Collection(collectionName).Find(ctx, query, pageOptions)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func ExecuteFindAndModifyQuery(orgId string, collectionName string, filter interface{}, data interface{}) (bson.M, error) {
	var result bson.M
	err := database.GetConnection(orgId).Collection(collectionName).FindOneAndUpdate(ctx, filter, data, findUpdateOpts).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetReportQueryResult(orgId string, collectioinName string, req ReportRequest) ([]bson.M, error) {
	//build filter query
	query := make(map[string]interface{})
	//Check emp id
	if req.EmpId != "" {
		query["eid"] = req.EmpId
	}

	//check emp id
	if len(req.EmpIds) > 0 {
		query["eid"] = bson.M{"$in": req.EmpIds}
	}
	//if date filter presented or not
	if req.DateColumn == "" { // start & end filter
		if !req.StartDate.IsZero() && !req.EndDate.IsZero() {
			query["start_date"] = bson.M{"$gte": req.StartDate, "$lte": req.EndDate}
			query["end_date"] = bson.M{"$gte": req.StartDate, "$lte": req.EndDate}
		} else if !req.StartDate.IsZero() && req.EndDate.IsZero() {
			query["start_date"] = bson.M{"$gte": req.StartDate}
		} else if req.StartDate.IsZero() && !req.EndDate.IsZero() {
			query["end_date"] = bson.M{"$lte": req.EndDate}
		}
	} else { // in between date filter
		if !req.StartDate.IsZero() && !req.EndDate.IsZero() {
			query[req.DateColumn] = bson.M{"$gte": req.StartDate, "$lte": req.EndDate}
		} else if !req.StartDate.IsZero() && req.EndDate.IsZero() {
			query[req.DateColumn] = bson.M{"$gte": req.StartDate}
		} else if req.StartDate.IsZero() && !req.EndDate.IsZero() {
			query[req.DateColumn] = bson.M{"$lte": req.EndDate}
		}
	}
	if req.Type != "" {
		query["type"] = req.Type
	}
	if req.Status != "" {
		query["status"] = req.Status
	}
	return GetQueryResult(orgId, collectioinName, query, int64(1), int64(200), nil)
}

func generateSearchQuery(filters []Filter) interface{} {
	if len(filters) == 0 {
		return nil
	}
	//build query
	var finalQuery interface{}
	var queryArray [](map[string][]bson.M)
	for _, filter := range filters {
		filterQuery := make(map[string][]bson.M)
		var con []bson.M
		conditions := filter.Conditions
		for _, condition := range conditions {
			var f bson.M
			if condition.Type == "date" {
				date, _ := time.Parse(time.RFC3339, condition.Value)
				f = bson.M{condition.Column: bson.M{condition.Operator: date}}
			} else {
				f = bson.M{condition.Column: bson.M{condition.Operator: condition.Value}}
			}
			con = append(con, f)
		}
		filterQuery[filter.Clause] = con
		queryArray = append(queryArray, filterQuery)
	}
	if len(filters) == 1 {
		finalQuery = queryArray[0]
	} else {
		finalQuery = bson.M{"$and": queryArray}
	}
	//fmt.Println(finalQuery)
	//query, _ := json.Marshal(finalQuery)
	//fmt.Println(string(query))
	return finalQuery
}

func GetSearchQueryResult(orgId string, collectionName string, filters []Filter) ([]bson.M, error) {
	query := generateSearchQuery(filters)
	return GetQueryResult(orgId, collectionName, query, int64(1), int64(200), nil)
}

func GetSearchQueryWithChildCount(orgId string, collectionName string, keyColumn string, childCollectionName string, lookupColumn string, filters []Filter) ([]bson.M, error) {
	matchQuery := generateSearchQuery(filters)
	pipeline := []bson.M{
		{"$lookup": bson.M{
			"from":         childCollectionName,
			"localField":   keyColumn,
			"foreignField": lookupColumn,
			"as":           "details",
		}},
		{"$addFields": bson.M{"count": bson.M{"$size": "$details"}}},
		{"$unset": "details"},
	}
	if matchQuery != nil {
		pipeline = append([]bson.M{{"$match": matchQuery}}, pipeline...)
	}
	fmt.Println(pipeline)
	return GetAggregateQueryResult(orgId, collectionName, pipeline)
}

func ExecuteLookupQuery(orgId string, query LookupQuery) ([]bson.M, error) {
	matchQuery := generateSearchQuery(query.ParentRef.Filter)
	pipeline := []bson.M{
		{"$lookup": bson.M{
			"from":         query.ParentRef.Name,
			"localField":   query.ParentRef.Key,
			"foreignField": query.ChildRef.Key,
			"as":           "details",
		},
		},
	}
	if query.Operation == "count" {
		pipeline = append(pipeline, bson.M{"$addFields": bson.M{"count": bson.M{"$size": "$details"}}})
		pipeline = append(pipeline, bson.M{"$unset": "details"})
	}
	if matchQuery != nil {
		pipeline = append([]bson.M{{"$match": matchQuery}}, pipeline...)
	}
	//fmt.Println(pipeline)
	return GetAggregateQueryResult(orgId, query.ParentRef.Name, pipeline)
}
