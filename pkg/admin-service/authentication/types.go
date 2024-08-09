package authentication

import (
	"kriyatec.com/go-api/pkg/shared/helper"
)

// LoginRequest
type LoginRequest struct {
	Id       string `json:"id" validate:"required"`
	Password string `json:"password" validate:"required"`
}
type EmpLoginRequest struct {
	ShopId       string `json:"shop_id" validate:"required"`
	MobileNumber string `json:"mobile_number" bson:"mobile_number"`
}

type OTPValidateRequest struct {
	Id  string `json:"id" validate:"required"`
	OTP int32  `json:"otp" validate:"required"`
}

// LoginResponse - for Login Response
type LoginResponse struct {
	Name     string              `json:"name"`
	UserRole string              `json:"role"`
	UserOrg  helper.Organization `json:"org" bson:"org"`
	Token    string              `json:"token"`
}

type OtpResponse struct {
	Name   string `json:"firstName"`
	Mobile string `json:"mobile"`
	Role   string `json:"role"`
	Token  string `json:"token"`
	Id     string `json:"_id"`
}

// ResetPasswordRequestDto - Dto for reset password Request
type ResetPasswordRequest struct {
	Id     string `json:"id" validate:"required,id"`
	OldPwd string `json:"old_pwd" bson:"old_pwd" validate:"required"`
	NewPwd string `json:"new_pwd" bson:"new_pwd" validate:"required"`
}
