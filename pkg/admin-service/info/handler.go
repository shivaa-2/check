package info

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"kriyatec.com/go-api/pkg/shared/helper"
)

var ctx = context.Background()

func getDocByIdHandler(c *fiber.Ctx) error {
	orgId := c.Get("OrgId")
	if orgId == "" {
		return helper.BadRequest("Organization Id missing")
	}
	c.Response().Header.Add("Cache-Time", "600000000")
	c.Response().Header.Add("Cache-Control", "Public")
	filter := helper.DocIdFilter(c.Params("id"))
	response, err := helper.GetQueryResult(orgId, "information", filter, int64(0), int64(1), nil)
	if err != nil {
		return helper.BadRequest(err.Error())
	}
	return helper.SuccessResponse(c, response)
}
