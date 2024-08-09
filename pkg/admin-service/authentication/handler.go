package authentication

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"kriyatec.com/go-api/pkg/shared/database"
	"kriyatec.com/go-api/pkg/shared/helper"
)

var ctx = context.Background()
var updateOpts = options.Update().SetUpsert(true)

// Login
// @Summary User Login
// @Description User Login using User Id and Password
// @Accept  json
// @Produce  json
// @Param loginRequest body LoginRequest true "Login Method"
// @Param OrgId header string true "tpctrz.com"
// @Success 200 {object} LoginResponse
// @Router /api/auth/login [post]
func LoginHandler(c *fiber.Ctx) error {
	if strings.Index(c.Get("origin"), "localhost") == -1 {
		c.Set("OrgId", "")
	}
	org, exists := helper.GetOrg(c)
	if !exists {
		//send error
		return helper.BadRequest("Invalid Org Id")
	}
	ctx := context.Background()
	loginRequest := new(LoginRequest)
	if err := c.BodyParser(loginRequest); err != nil {
		return helper.BadRequest("Invalid params")
	}

	// validate := validator.New()
	// err = validate.Struct(&body)

	// if err != nil { // validation failed
	// 	errMsg := utils.GetValidationError(err, log)
	// 	http.Error(w, errMsg, http.StatusBadRequest)
	// 	return
	// }

	//TO BE
	result := database.GetConnection(org.Id).Collection("user").FindOne(ctx, bson.M{
		"_id": loginRequest.Id,
	})
	var user bson.M
	err := result.Decode(&user)
	if err == mongo.ErrNoDocuments {
		return helper.BadRequest("Invalid User Id / Password")
	}
	if err != nil {
		return helper.BadRequest("Internal server Error")
	}
	if !helper.ValidatePassword(loginRequest.Password, user["pwd"].(string)) {
		return helper.BadRequest("Invalid ID / Password")
	}

	claims := helper.GetNewJWTClaim()
	claims["id"] = user["_id"]
	claims["role"] = user["role"]
	claims["uo_id"] = org.Id
	claims["uo_group"] = org.Group

	token := helper.GenerateJWTToken(claims, 365*10)
	response := &LoginResponse{
		Name:     user["name"].(string),
		UserRole: user["role"].(string),
		UserOrg:  org,
		Token:    token,
	}
	return c.JSON(response)
}

func OTPValidateHandler(c *fiber.Ctx) error {
	if strings.Index(c.Get("origin"), "localhost") == -1 {
		c.Set("OrgId", "")
	}
	org, exists := helper.GetOrg(c)
	if !exists {
		//send error
		return helper.BadRequest("Invalid Org Id")
	}
	OtpValidateRequest := new(OTPValidateRequest)
	if err := c.BodyParser(OtpValidateRequest); err != nil {
		return helper.BadRequest("Invalid OTP params")
	}

	result := database.GetConnection(org.Id).Collection("customer").FindOne(ctx, bson.M{
		"_id": OtpValidateRequest.Id,
	})
	var customer bson.M
	err := result.Decode(&customer)
	if err == mongo.ErrNoDocuments {
		return helper.BadRequest("Invalid ID")
	}
	if err != nil {
		return helper.BadRequest("Internal server Error")
	}
	// if customer["otp"] != OtpValidateRequest.OTP {
	if customer["otp"] != OtpValidateRequest.OTP {
		return helper.BadRequest("Invalid OTP")
	}
	name := ""
	if customer["firstName"] != nil {
		name = customer["firstName"].(string)
	}
	claims := helper.GetNewJWTClaim()
	claims["id"] = customer["_id"]
	claims["name"] = name
	claims["role"] = "1"
	claims["uo_id"] = "sakthi"
	claims["uo_group"] = "sakthi"

	token := helper.GenerateJWTToken(claims, 365*10)
	response := &OtpResponse{
		Name:   name,
		Mobile: customer["mobile"].(string),
		Role:   "1",
		Token:  token,
		Id:     customer["_id"].(string),
	}
	return c.JSON(response)
}

func ResetPasswordHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	userToken := helper.GetUserTokenValue(c)
	ctx := context.Background()
	req := new(ResetPasswordRequest)
	err := c.BodyParser(req)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	if req.Id == "" {
		req.Id = userToken.UserId
	}

	result := database.GetConnection(orgId).Collection("user").FindOne(ctx, bson.M{
		"_id": req.Id,
	})
	var user bson.M
	err = result.Decode(&user)
	if err == mongo.ErrNoDocuments {
		return helper.BadRequest("User Id not available")
	}

	if err != nil {
		return helper.BadRequest("Internal server Error")
	}

	if userToken.UserRole == "SA" {
		//Check the old password
		if !helper.CheckPasswordHash(req.OldPwd, user["pwd"].(primitive.Binary)) {
			return helper.BadRequest("Given user id and old password mismated")
		}
	}

	// TODO set random string - hard coded for now
	passwordHash := helper.PasswordHash(req.NewPwd)

	_, err = database.GetConnection(orgId).Collection("user").UpdateByID(ctx,
		req.Id,
		bson.M{"$set": bson.M{"pwd": passwordHash}},
	)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return c.JSON("Password Updated")
	// automatically return 200 success (http.StatusOK) - no need to send explictly
}

func ChangePasswordHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	userToken := helper.GetUserTokenValue(c)
	ctx := context.Background()
	req := new(ResetPasswordRequest)
	err := c.BodyParser(req)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	if req.Id == "" {
		req.Id = userToken.UserId
	}

	result := database.GetConnection(orgId).Collection("user").FindOne(ctx, bson.M{
		"_id": req.Id,
	})
	var user bson.M
	err = result.Decode(&user)
	if err == mongo.ErrNoDocuments {
		return helper.BadRequest("User Id not available")
	}
	if err != nil {
		return helper.BadRequest("Internal server Error")
	}
	//Check given old password is right or not?
	if !helper.ValidatePassword(req.OldPwd, user["pwd"].(string)) {
		return helper.BadRequest("Your Old password is Wrong!")
	}
	//update new password hash to the table
	passwordHash := helper.PasswordHash(req.OldPwd)
	_, err = database.GetConnection(orgId).Collection("user").UpdateByID(ctx,
		req.Id,
		bson.M{"$set": bson.M{"pwd": passwordHash}},
	)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return c.JSON("Password Updated")
	// automatically return 200 success (http.StatusOK) - no need to send explictly
}

func OrgConfigHandler(c *fiber.Ctx) error {
	org, exists := helper.GetOrg(c)
	if !exists {
		//send error
		return helper.BadRequest("Org not found")
	}
	return helper.SuccessResponse(c, org)
}

// postEntitiesHandler - Create Entities
func RegistrationHandler(c *fiber.Ctx) error {

	orgId := c.Get("OrgId")

	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	// Insert data to collection
	inputData := make(map[string]interface{})

	err := c.BodyParser(&inputData)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	//update date string to time object
	// helper.UpdateDateObject(inputData)
	//Hardcoded OTP
	filter := bson.M{"_id": inputData["mobile"]}
	otp := helper.GetNewOtp()
	inputData["otp"] = otp
	response, err := database.GetConnection(orgId).Collection("customer").UpdateOne(
		ctx,
		filter,
		bson.M{"$set": inputData},
		updateOpts,
	)
	if err != nil {
		fmt.Println(err.Error())
		return helper.BadRequest(err.Error())
	}
	helper.SendOTP(inputData["mobile"].(string), fmt.Sprintf("%d", otp))
	return helper.SuccessResponse(c, response.MatchedCount)
}

func EmpLoginHandler(c *fiber.Ctx) error {
	if strings.Index(c.Get("origin"), "localhost") == -1 {
		c.Set("OrgId", "")
	}
	org, exists := helper.GetOrg(c)
	if !exists {
		//send error
		return helper.BadRequest("Invalid Org Id")
	}
	ctx := context.Background()
	loginRequest := new(EmpLoginRequest)
	if err := c.BodyParser(loginRequest); err != nil {
		return helper.BadRequest("Invalid params")
	}

	// validate := validator.New()
	// err = validate.Struct(&body)

	// if err != nil { // validation failed
	// 	errMsg := utils.GetValidationError(err, log)
	// 	http.Error(w, errMsg, http.StatusBadRequest)
	// 	return
	// }
	//TO BE
	result := database.GetConnection(org.Id).Collection("shop_employee").FindOne(ctx, bson.M{
		"shop_id":    loginRequest.ShopId,
		"emp_mobile": loginRequest.MobileNumber,
	})
	var user bson.M
	err := result.Decode(&user)
	if err == mongo.ErrNoDocuments {
		return helper.BadRequest("Invalid Shop Id / Mobile Number")
	}
	if err != nil {
		return helper.BadRequest("Internal server Error")
	}

	claims := helper.GetNewJWTClaim()
	claims["id"] = user["_id"]
	claims["role"] = user["role"]
	claims["shop_id"] = loginRequest.ShopId
	claims["uo_id"] = org.Id
	claims["uo_group"] = org.Group

	token := helper.GenerateJWTToken(claims, 365*10)
	response := &LoginResponse{
		Name:     user["emp_name"].(string),
		UserRole: user["role"].(string),
		UserOrg:  org,
		Token:    token,
	}
	return c.JSON(response)
}

// Online Javascript Editor for free
// Write, Edit and Run your Javascript code using JS Online Compiler
