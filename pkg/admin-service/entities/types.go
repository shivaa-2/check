package entities

import (
	"context"
	"time"
)

var ctx = context.Background()

type CreatedOnData struct {
	CreatedOn time.Time `json:"created_on" bson:"created_on"`
	CreatedBy string    `json:"created_by" bson:"created_by"`
}

type Leave struct {
	Type      string    `json:"type" bson:"type"`
	EmpId     string    `json:"emp_id" bson:"emp_id"`
	StartDate time.Time `json:"start_date" bson:"start_date"`
	EndDate   time.Time `json:"end_date" bson:"end_date"`
	Status    string    `json:"status" bson:"status"`
}
type ReportRequest struct {
	OrgId      string    `json:"org_id" bson:"org_id"`
	EmpId      string    `json:"emp_id" bson:"emp_id"`
	EmpIds     []string  `json:"emp_ids" bson:"emp_ids"`
	Type       string    `json:"type" bson:"type"`
	DateColumn string    `json:"date_column" bson:"date_column"`
	StartDate  time.Time `json:"start_date" bson:"start_date"`
	EndDate    time.Time `json:"end_date" bson:"end_date"`
	Status     string    `json:"status" bson:"status"`
}
type Condition struct {
	Column   string      `json:"column" bson:"column"`
	Operator string      `json:"operator" bson:"operator"`
	Value    interface{} `json:"value" bson:"value"`
}

type PreSignedUploadUrlRequest struct {
	FolderPath string             `json:"folder_path" bson:"folder_path"`
	FileKey    string             `json:"file_key" bson:"file_key"`
	MetaData   map[string]*string `json:"metadata" bson:"metadata"`
}

type Filter struct {
	Clause     string      `json:"clause" bson:"clause"`
	Conditions []Condition `json:"conditions" bson:"conditions"`
}

type StockPurchase struct {
	PurchaseId         string    `json:"purchase_id" bson:"purchase_id"`
	SupplierId         string    `json:"supplier_id" bson:"supplier_id"`
	ShopId             string    `json:"shop_id" bson:"shop_id"`
	PurchaseDate       string    `json:"purchase_date" bson:"purchase_date"`
	TxnType            string    `json:"txn_type" bson:"txn_type"`
	InvoiceNumber      string    `json:"invoice_number" bson:"invoice_number"`
	InvoiceDate        time.Time `json:"invoice_date" bson:"invoice_date"`
	ProductId          string    `json:"product_id" bson:"product_id"`
	Quantity           int       `json:"quantity" bson:"quantity"`
	BatchNumber        string    `json:"batch_number" bson:"batch_number"`
	GstPercentage      float64   `json:"gst_percentage" bson:"gst_percentage"`
	Amount             float64   `json:"amount" bson:"amount"`
	DiscountPercentage float64   `json:"discount_percentage" bson:"discount_percentage"`
	DosageForm         string    `json:"dosage_form" bson:"dosage_form"`
	CreatedOn          time.Time `json:"created_on" bson:"created_on"`
	CreatedBy          string    `json:"created_by" bson:"created_by"`
	ExpiryDate         time.Time `json:"expiry_date" bson:"expiry_date"`
}

type Billing struct {
	BillNumber           string               `json:"bill_number" bson:"bill_number"`
	ShopId               string               `json:"shop_id" bson:"shop_id"`
	ProductId            string               `json:"product_id" bson:"product_id"`
	CustomerId           string               `json:"customer_id" bson:"customer_id"`
	Quantity             int                  `json:"quantity" bson:"quantity"`
	BatchNumber          string               `json:"batch_number" bson:"batch_number"`
	GstPercentage        float64              `json:"gst_percentage" bson:"gst_percentage"`
	PaymentMethod        string               `json:"payment_method" bson:"payment_method"`
	PrescriptionsDetails PrescriptionsDetails `json:"prescription_details" bson:"prescription_details"`
	Mrp                  float64              `json:"mrp" bson:"mrp"`
	DiscountPercentage   float64               `json:"discount_percentage" bson:"discount_percentage"`
	SellingPrice         float64              `json:"selling_price" bson:"selling_price"`
	CreatedOn            time.Time            `json:"created_on" bson:"created_on"`
	CreatedBy            string               `json:"created_by" bson:"created_by"`
	ExpiryDate           time.Time            `json:"expiry_date" bson:"expiry_date"`
}

type PrescriptionsDetails struct {
	PrescriptionNumber  string    `json:"prescription_number" bson:"prescription_number"`
	DoctorName          string    `json:"doctor_name" bson:"doctor_name"`
	DoctorContactNumber string    `json:"doctor_contact_number" bson:"doctor_contact_number"`
	DateOfPrescription  time.Time `json:"date_of_prescription" bson:"date_of_prescription"`
	DosageInstructions  string    `json:"dosage_instructions" bson:"dosage_instructions"`
}
