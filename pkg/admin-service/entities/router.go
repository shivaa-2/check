package entities

import (
	"github.com/gofiber/fiber/v2"

	"kriyatec.com/go-api/pkg/shared/helper"
)

func SetupAllRoutes(app *fiber.App) {
	SetupCRUDRoutes(app)
	SetupOrderRoutes(app)
	SetupAppRoutes(app)
	SetupSearchRoutes(app)
	SetupLookupRoutes(app)
	SetupQueryRoutes(app) //raw query
	SetupSharedDBRoutes(app)
	SetupUtilRoutes(app)
	SetupUploadRoutes(app)
	SetupDownloadRoutes(app)
	SetupGetUpdateRoutes(app)
	SetupPurchaseBillingRoutes(app)
	SetupMultifilterRoutes(app)
	SetupPurchaseRoutes(app)
	SetupShopInvoiceRoutes(app)
	SetupShortenUrl(app)
	SetupMultifilterLookUpRoutes(app)
	SetupS3UploadsRoutes(app)
	SetupSMSSampleRoutes(app)
	app.Static("/image", "./uploads")
	app.Get("/image", func(c *fiber.Ctx) error {
		return c.SendString("https://blr1.digitaloceanspaces.com/sakthipharma/uploads/system/shop_invoice/INV__170__05_08_2024_10_32.pdf")
	})
	app.Get("/url/:id", RedirectURL)
}

func SetupCRUDRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/api", "REST API")
	r.Get("/:collectionName/filter/:key/:value/:page?/:limit?", getDocsByKeyValueHandler)
	r.Get("/:collectionName/:id", getDocByIdHandler)
	r.Get("/:collectionName/:page?/:limit?/:sort?", getDocsHandler)
	r.Post("/:collectionName", postDocHandler)
	r.Post("/:collectionName/search/:page?/:limit?", searchDocsHandler)
	r.Put("/:collectionName/:id", putDocByIdHandler)
	r.Delete("/:collectionName/:colName/:value", deleteDocByIdHandler)
}

func SetupAppRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/app-api", "Mobile REST API")
	r.Get("/:collectionName/:date/:page?/:limit?", getDocsByDateHandler)
}

func SetupSearchRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/search", "Search API")
	r.Get("/:collectionName/:key/:page?/:limit?", textSearchhHandler)
	r.Post("/:collectionName/filter", searchDocsHandler)
	r.Post("/:parent_collection/:key_column/:child_collection/:lookup_column", searchEntityWithChildCountHandler)
}

func SetupLookupRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/lookup", "Data Lookup API")
	r.Post("/:collectionName", DataLookupDocsHandler)
}

func SetupQueryRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/query", "Raw Query API")
	r.Post("/:type/:collectionName", rawQueryHandler)
}

func SetupSharedDBRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/shared", "Shared DB API")
	r.Get("/:collectionName", sharedDBEntityHandler)
	//	r.HandleFunc("/search/{collection_name}", sharedDBSearchEntiryHandler).Methods(http.MethodPost)
}

func SetupUtilRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/util", "util APIs")
	r.Get("/nextseq/:key", getNextSeqNumberHandler)
	r.Post("/getuploadurl", getPreSignedUploadUrlHandler)
}

func SetupUploadRoutes(app *fiber.App) {
	//without JWT Token validation (without auth)
	upload := helper.CreateRouteGroup(app, "/upload", "Upload APIs")
	upload.Post("/:category", fileUpload)
	upload.Post("/system/:category", systemFileUpload)
}

func SetupDownloadRoutes(app *fiber.App) {
	//without JWT Token validation (without auth)
	r := helper.CreateRouteGroup(app, "/file", "Upload APIs")
	r.Get("/all/:category/:status/:page?/:limit?", getAllFileDetails)
	r.Get("/:category", getFileDetails)
	//r.Get("/:category/:fileName", fileDownload)
}

func SetupOrderRoutes(app *fiber.App) {
	//without JWT Token validation (without auth)
	r := helper.CreateRouteGroup(app, "/order", "Order Releated APIs")
	r.Post("/", postDocHandler)
	r.Post("/payment/init", createOrder)
	r.Post("/payment/status_update", orderStatusUpdate)
	r.Post("/payment/gettoken", getPaymentToken)
	r.Post("/update/status/:id", statusUpdate)
	r.Post("/payment/create_order", createOrderNewVersion)
	r.Post("/payment/refund/:order_id", refundOrder)
	//r.Get("/:category/:fileName", fileDownload)
}

func SetupGetUpdateRoutes(app *fiber.App) {
	//without JWT Token validation (without auth)
	r := helper.CreateRouteGroup(app, "/update", "Master Data Update APIs")
	r.Get("/:collectionName/:date/:page?/:limit?", getUpdateDocsHandler)
}

func SetupPurchaseBillingRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/stock", "Purchase and Billing APIs")
	r.Post("/purchase", CreatePurchase)
	r.Post("/billing", CreateBilling)
}

func SetupMultifilterRoutes(app *fiber.App) {

	r := helper.CreateRouteGroup(app, "/multifilter", "Multi Filter APIs")
	r.Post("/:collectionName", GetDataByFilterQuery)

}

func SetupMultifilterLookUpRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/multifilterlookup", "LookUp APIs")
	r.Post("/:collectionName", GetDataByFilterQuery1)
}

func SetupPurchaseRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/upload-excel", "Purchase Invoice bulk Upload APIs")
	r.Post("/purchase-invoice", PurchaseInvoiceUpload)
}

func SetupShopInvoiceRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/generate-invoice", "Shop Invoice APIs")
	r.Get("/shop/:BillNumber", GenerateInvoice)
}

func SetupS3UploadsRoutes(app *fiber.App) {
	r := helper.CreateRouteGroup(app, "/s3-upload", "S3 APIs")
	r.Post("/file", S3Upload)
	r.Get("/create-bucket/:name", CreateS3Bucket)
}

func SetupSMSSampleRoutes(app *fiber.App) {

	r := helper.CreateRouteGroup(app, "/sms", "SMS Sample APIs")
	r.Get("/send", SendSMOtp)

}

func SetupShortenUrl(app *fiber.App) {

	r := helper.CreateRouteGroup(app, "/shorten_url", "SMS Sample APIs")
	r.Post("/short-hand", ShortenURL)
}
