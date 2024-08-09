package helper

import (
	"github.com/gofiber/fiber/v2"
)

type Error struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Success struct {
	Status   int         `json:"status"`
	Data     interface{} `json:"data"`
	ErrorMsg string      `json:"error_msg"`
}

func (e *Error) Error() string {
	return e.Message
}

func EntityNotFound(m string) *Error {
	return &Error{Status: 404, Code: "entity-not-found", Message: m}
}

func BadRequest(m string) *Error {
	return &Error{Status: 400, Code: "bad-request", Message: m}
}

func Unexpected(m string) *Error {
	return &Error{Status: 500, Code: "internal-server", Message: m}
}

func SuccessResponse(c *fiber.Ctx, data interface{}) error {
	return c.JSON(&Success{Status: 200, Data: data, ErrorMsg: ""})
}
