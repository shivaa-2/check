package helper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func SendOTP(mobileNo string, otp string) string {
	sms_otp_url := os.Getenv("SMS_OTP_URL")
	url := fmt.Sprintf(sms_otp_url, mobileNo, otp)
	return sendSMS(url, "OTP sent to "+mobileNo)
}

func SendOrderSMS(mobileNo string, order map[string]interface{}) string {
	orderId := order["_id"].(string)
	userName := order["name"].(string)
	paymentMode := order["paymentMode"].(string)
	var sms_order_url string
	var msg string
	if paymentMode == "cod" {
		sms_order_url = os.Getenv("SMS_COD_ORDER_URL")
		msg = fmt.Sprintf("Dear_%s_Your order %s has been placed successfully. It should reach you on or before %s. Thank you. Team SakthiPharma",
			userName,
			orderId,
			time.Now().AddDate(0, 0, 2).Format("January 2, 2006"))
		fmt.Println("COD Order")
	} else {
		sms_order_url = os.Getenv("SMS_ONLINE_ORDER_URL")
		msg = fmt.Sprintf("Dear %s Your order %s has been placed successfully. Once we receive your payment, we will process your order. Thank You. Team SakthiPharma", userName, orderId)
		fmt.Println("Online Payment Order")
	}
	smsurl := fmt.Sprintf(sms_order_url+"&to=%s&message=%s", mobileNo, url.QueryEscape(msg))
	fmt.Println(smsurl)
	return sendSMS(smsurl, "SMS sent for Order Id:"+orderId)
}

func SendPaymentConfirmSMS(mobileNo string, order map[string]interface{}) string {
	orderId := order["_id"].(string)
	userName := order["name"].(string)
	amount := order["total_amount"].(string)
	paymentMode := order["paymentMode"].(string)

	sms_url := os.Getenv("SMS_PAYMENT_CONFIRM")
	msg := fmt.Sprintf("Dear %s We have received your payment of %s through %s for your order %s. It should reach you on or before %s. Thank You. Team SakthiPharma",
		userName,
		orderId,
		amount,
		paymentMode,
		time.Now().AddDate(0, 0, 2).Format("January 2, 2006"))
	smsurl := fmt.Sprintf(sms_url+"&to=%s&message=%s", mobileNo, url.QueryEscape(msg))
	fmt.Println(smsurl)
	return sendSMS(smsurl, "Payment Confirmation SMS sent for Order Id:"+orderId)
}

func SendTakenForDeliverySMS(mobileNo string, orderId string, userName string) string {
	sms_url := os.Getenv("SMS_DELIVERY_TAKEN_URL")
	msg := fmt.Sprintf("Dear %s Your Order %s is out for Delivery and should reach you soon. The delivery person may call you before reaching your place. Team SakthiPharma", userName, orderId)
	smsurl := fmt.Sprintf(sms_url+"&to=%s&message=%s", mobileNo, url.QueryEscape(msg))
	fmt.Println(smsurl)
	return sendSMS(smsurl, "Taken for Delivery SMS sent for Order Id:"+orderId)
}

func SendDeliverySMS(mobileNo string, orderId string, userName string) string {
	sms_url := os.Getenv("SMS_DELIVERY_URL")
	msg := fmt.Sprintf("Greetings from SakthiPharma. Your Order %s is delivered successfully. In case of any issues, please call our Customer Care %s Stay Happy and Healthy Team SakthiPharma", orderId, "044-48521151")
	smsurl := fmt.Sprintf(sms_url+"&to=%s&message=%s", mobileNo, url.QueryEscape(msg))
	fmt.Println(smsurl)
	return sendSMS(smsurl, "Delivery SMS sent for Order Id:"+orderId)
}

func sendSMS(url string, msg string) string {
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	smsId := string(body)
	if err != nil {
		return ""
	}
	fmt.Println("SMS ID:" + smsId + " - " + msg)
	return smsId
}

func SendSOTP(mobileNo string, otp string) string {
	sms_otp_url := os.Getenv("SMS_OTP_URL")
	//s:="https://blr1.digitaloceanspaces.com/sakthipharma/system/invoice/INV000003__2022-09-03-06-27-20.pdf"
	url := fmt.Sprintf(sms_otp_url, mobileNo, "sivabharathisivabharathi")
	return sendSMS(url, "OTP sent to "+mobileNo)
}
