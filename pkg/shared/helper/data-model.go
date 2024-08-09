package helper

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
)

// vars primitive type
var strType string
var timeType time.Time
var boolType bool
var intType int
var int32Type int32
var int64Type int64
var floatType float64
var dataModels []bson.M

type Location struct {
	Type        string         `json:"type" validate:"required"`
	Coordinates [2]json.Number `json:"coordinates" validate:"required"`
}

type Config struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	JName    string `json:"jname"`
	Required bool   `json:"required"`
	Omit     bool   `json:"omit"`
}
type AggregateResult struct {
	Id     string `bson:"_id"`
	Fields []Config
}

// define map of primitive types
var typeMap = map[string]interface{}{
	"string":   &strType,
	"time":     &timeType,
	"bool":     &boolType,
	"int":      &intType,
	"int32":    &int32Type,
	"int64":    &int64Type,
	"float64":  &floatType,
	"location": &Location{},

	// .....
}

func GetObjectType(collectionName string) interface{} {
	return typeMap[collectionName]
}

func LoadDataModelFromDB(OrgId string) map[string][]Config {
	var queryStr = `[
		{
			"$match":{"status":"A"}
		},
		{
			"$group": {
				"_id": "$model_id",
        "fields": {
          "$push": {
            "name":"$name",
            "type": "$data_type",
            "jname":"$json_name",
            "required":"$required",
            "omit":"$omit_empty"
			    }
        }
      }
		}
	]`
	var query interface{}
	json.Unmarshal([]byte(queryStr), &query)
	var result []AggregateResult
	response, err := ExecuteAggregateQuery(OrgId, "model_detail", query)
	if err != nil {
		return nil
	}
	if err = response.All(ctx, &result); err != nil {
		//log.Errorf("Collection:%s Error: data_models", err.Error())
		return nil
	}
	resultMap := map[string][]Config{}
	for _, res := range result {
		resultMap[res.Id] = res.Fields
	}
	return resultMap
}

func loadModels(models map[string][]Config, key string) interface{} {
	if model, exists := typeMap[key]; exists {
		return model
	}
	// create Struct fields
	var dynamicStruct []reflect.StructField
	for _, field := range models[key] {
		fieldType := typeMap[field.Type]
		if _, exists := typeMap[field.Type]; !exists {
			// recursively load dependent models
			fieldType = loadModels(models, field.Type)
		}
		var tag = "json:\"" + field.JName + "\" bson:\"" + field.JName + "\""
		if field.Required {
			tag += " validate:\"required\""
		}
		dynamicStruct = append(dynamicStruct,
			reflect.StructField{
				Name: field.Name,
				Type: reflect.TypeOf(fieldType),
				Tag:  reflect.StructTag(tag),
			},
		)
	}

	//created_by
	var tag = `json:"created_by" bson:"created_by"`
	dynamicStruct = append(dynamicStruct,
		reflect.StructField{
			Name: "CreatedBy",
			Type: reflect.TypeOf(typeMap["string"]),
			Tag:  reflect.StructTag(tag),
		},
	)

	//created_On
	tag = `json:"created_on" bson:"created_on"`
	dynamicStruct = append(dynamicStruct,
		reflect.StructField{
			Name: "CreatedOn",
			Type: reflect.TypeOf(typeMap["time"]),
			Tag:  reflect.StructTag(tag),
		},
	)
	//create struct object
	obj := reflect.StructOf(dynamicStruct)
	objIns := reflect.New(obj).Interface()
	// set to typeMap
	typeMap[key] = objIns
	return objIns
}

func createDynamicTypes(data map[string][]Config) {
	for key := range data {
		loadModels(data, key)
	}
}

func ValidateInputJson(orgId string, collectionName string, inputByte []byte, userToken UserToken) (interface{}, error) {
	//check collection name struct type already loaded or not
	//if not, read the scheme info from db and load in typeMap array
	//orgCollectionName := orgId + "_"+ collectionName
	if _, exists := typeMap[collectionName]; !exists {
		var data map[string][]Config
		data = LoadDataModelFromDB(orgId)
		createDynamicTypes(data)
	}
	if _, exists := typeMap[collectionName]; !exists {
		return nil, BadRequest("Validation schema doesn't exists")
	}
	//inputByte, _ := json.Marshal(requestBody)
	//dynamically create the collection's schema struct
	objIns := reflect.New(reflect.TypeOf(typeMap[collectionName])).Interface()
	json.Unmarshal(inputByte, &objIns)
	// loop through pointer to get the actual struct
	rv := reflect.ValueOf(objIns)
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	// Validate Struct (Dynamically)
	validate := validator.New()
	validationErr := validate.Struct(rv.Interface())
	if validationErr != nil { // validation failed
		//	_, errorFields := GetSchemValidationError(validationErr)
		return nil, validationErr
	}
	setCurrentTime(rv, "CreatedOn")
	setStringField(rv, "CreatedBy", userToken.UserId)
	return objIns, nil
}

func setCurrentTime(rv reflect.Value, fieldName string) {
	field := rv.FieldByName(fieldName)
	if field.IsValid() {
		dt := time.Now()
		field.Set(reflect.ValueOf(&dt))
	}
}

func setStringField(rv reflect.Value, fieldName string, value string) {
	field := rv.FieldByName(fieldName)
	if field.IsValid() {
		field.Set(reflect.ValueOf(&value))
	}
}

func GetSchemValidationError(err error) (errMsg string, errorFields map[string]string) {
	errorFields = map[string]string{}
	if _, ok := err.(*validator.InvalidValidationError); ok {
		errMsg = "Invalid Validation Error"
		//fmt.Printf("Invalid Validation Error : %s", err.Error())
		return
	}

	for _, err := range err.(validator.ValidationErrors) {
		errMsg += fmt.Sprintf("%s validation failed for %s\n", err.Tag(), err.Field())
		//fmt.Printf(errMsg)
		errorFields[err.Namespace()] = err.Tag()
	}
	return
}
