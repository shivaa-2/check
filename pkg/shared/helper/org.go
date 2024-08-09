package helper

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"

	"kriyatec.com/go-api/pkg/shared/database"
)

func GetOrg(c *fiber.Ctx) (Organization, bool) {
	orgId := c.Get("OrgId")
	if orgId == "" || orgId == "null" {
		orgId = c.Subdomains(1)[0]
	}
	if orgId == "" {
		return Organization{}, false
	}
	if org, exists := OrgList[orgId]; exists {
		return org, true
	}
	LoadOrgConfig()
	if _, exists := OrgList[orgId]; !exists {
		return OrgList["dev"], true
	}
	return OrgList[orgId], true
}

func GetOrgIdFromDomainName(c *fiber.Ctx) string {
	org, exists := GetOrg(c)
	if !exists {
		return ""
	}
	return org.Id
}

func GetOrgIdFromHeader(c *fiber.Ctx) string {
	return c.Get("OrgId")
}

func LoadOrgConfig() {
	ctx := context.Background()
	cur, err := database.SharedDB.Collection("organization").Find(ctx, bson.D{})
	if err != nil {
		//log.Errorf("Organization Configuration Error %s", err.Error())
		defer cur.Close(ctx)
		return
	}
	var result []Organization
	if err = cur.All(ctx, &result); err != nil {
		return
	}
	for _, o := range result {
		OrgList[o.Id] = o
	}
}
