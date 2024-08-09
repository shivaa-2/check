package helper

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	cashfreeSDK "github.com/cashfree/cashfree-pg-sdk-go/implementation"
	"github.com/google/uuid"
)

var apiVersion = GetenvStr("CASHFREE_API_VERSION", "2022-01-01")
var appId = GetenvStr("CASHFREE_APPID", "246185f4400a1c8c3384db822a581642")
var secretKey = GetenvStr("CASHFREE_SECRETKEY", "9e777991ddcc72b57859bd61ec3d9c6db913cd7c")
var baseUrl = GetenvStr("CASHFREE_BASE_URL", "https://api.cashfree.com")

var environment = strings.ToUpper(GetenvStr("CASHFREE_ENVIRONMENT", "SANDBOX"))
var returnUrl = GetenvStr("CASHFREE_RETURN_URL", "https://sakthipharma.com/order/payment_status/{order_id}/{order_token}")
var notifyUrl = GetenvStr("CASHFREE_NOTIFY_URL", "https://api.sakthipharma.com/order/payment/status_update/")

var tokenUrl = GetenvStr("CASHFREE_TOKEN_URL", "https://api.cashfree.com/api/v2/cftoken/order")

type OrderRequest struct {
	OrderId        string  `json:"order_id" bson:"order_id"`
	Amount         float64 `json:"amount" bson:"amount"`
	CustomerId     string  `json:"customer_id" bson:"customer_id"`
	CustomerMobile string  `json:"customer_mobile" bson:"customer_mobile"`
	CustomerEmail  string  `json:"customer_email" bson:"customer_email"`
}
type TokenRequest struct {
	OrderId string  `json:"order_id" bson:"order_id"`
	Amount  float64 `json:"amount" bson:"amount"`
}

type OrderResponse struct {
	OrderId     string `json:"order_id" bson:"order_id"`
	OrderToken  string `json:"order_token" bson:"order_token"`
	OrderStatus string `json:"order_status" bson:"order_status"`
	PaymentLink string `json:"paymentlink" bson:"paymentlink"`
}

func getSession() cashfreeSDK.CFConfig {
	env := cashfreeSDK.SANDBOX
	if environment == "PRODUCTION" {
		env = cashfreeSDK.PRODUCTION
	}
	return cashfreeSDK.CFConfig{
		Environment:  &env,
		ApiVersion:   &apiVersion,
		ClientId:     &appId,
		ClientSecret: &secretKey,
	}
}

func getHeader() cashfreeSDK.CFHeader {
	idempotencyKey := uuid.New().String()
	requestId := uuid.NewString()
	return cashfreeSDK.CFHeader{
		RequestID:      &requestId,
		IdempotencyKey: &idempotencyKey, //random string
	}
}

func getRequest(request OrderRequest) cashfreeSDK.CFOrderRequest {
	orderMeta := cashfreeSDK.CFOrderMeta{
		ReturnUrl: returnUrl,
		NotifyUrl: returnUrl,
	}
	return cashfreeSDK.CFOrderRequest{
		OrderId:       &request.OrderId,
		OrderAmount:   request.Amount,
		OrderCurrency: "INR",
		CustomerDetails: cashfreeSDK.CFCustomerDetails{
			CustomerId:    request.CustomerId,
			CustomerEmail: request.CustomerEmail,
			CustomerPhone: request.CustomerMobile,
		},
		OrderNote: &request.OrderId,
		OrderMeta: &orderMeta,
	}
}

func OCreateOrder(request OrderRequest) (OrderResponse, error) {
	cfOrder, _, cfError := executeOrder(request)
	var res OrderResponse
	if cfError != nil {
		return res, errors.New(cfError.GetCode() + "-" + cfError.GetMessage())
	}
	res.OrderId = cfOrder.GetOrderId()
	res.OrderToken = cfOrder.GetOrderToken()
	res.OrderStatus = cfOrder.GetOrderStatus()
	res.PaymentLink = cfOrder.GetPaymentLink()
	return res, nil
}

func executeOrder(request OrderRequest) (*cashfreeSDK.CFOrder, *cashfreeSDK.CFResponseHeader, *cashfreeSDK.CFError) {
	session := getSession()
	header := getHeader()
	return cashfreeSDK.CreateOrder(&session, &header, getRequest(request))
}

func CreateOrder(request []byte) (map[string]interface{}, error) {
	url := baseUrl + "/pg/orders"
	return HttpRequest(url, "POST", request)
}

func RefundOrder(orderId string, request []byte) (map[string]interface{}, error) {
	url := baseUrl + "/pg/orders/" + orderId + "/refunds"
	return HttpRequest(url, "POST", request)
}

func GetPaymentToken(request []byte) (map[string]interface{}, error) {
	return HttpRequest(tokenUrl, "POST", request)
}

func HttpRequest(url string, method string, requestBody []byte) (map[string]interface{}, error) {
	r, err := http.NewRequest(method, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("x-api-version", apiVersion)
	r.Header.Add("x-client-id", appId)
	r.Header.Add("x-client-secret", secretKey)
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var response map[string]interface{}
	//	fmt.Printf("Payment Status Code %d", res.StatusCode)
	derr := json.NewDecoder(res.Body).Decode(&response)
	if derr != nil {
		return nil, derr
	}
	//if error comes, we need to parse the request body and get the order id
	if res.StatusCode != 200 {
		input := getOrderInputData(requestBody)
		if input != nil {
			response["order_id"] = input["order_id"]
		}
	}
	response["response_code"] = res.StatusCode
	response["request_date"] = time.Now()
	return response, nil
}

func getOrderInputData(request []byte) map[string]interface{} {
	inputData := make(map[string]interface{})
	err := json.Unmarshal(request, &inputData)
	if err != nil {
		return nil
	}
	return inputData
}

func GetOrderMeta() map[string]string {
	return map[string]string{
		"notify_url": notifyUrl,
		"return_url": returnUrl,
	}
}

func CreateOrderOnUpdatedCashFreeVersion(request []byte) (map[string]interface{}, error) {
	url := baseUrl + "/pg/orders"
	method := "POST"
	client := &http.Client{}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(request))
	if err != nil {

		return nil, err
	}
	newApiVersion := "2023-08-01"
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("x-api-version", newApiVersion)
	req.Header.Add("x-client-id", appId)
	req.Header.Add("x-client-secret", secretKey)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var createOrderResp map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&createOrderResp)
	if err != nil {
		return nil, err
	}
	return createOrderResp, nil
}
