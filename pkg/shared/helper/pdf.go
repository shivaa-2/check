package helper

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GoPdf struct {
	*gofpdf.Fpdf
}

func GenerateInvoicePDF(invoice map[string]interface{}, orgId string) (map[string]interface{}, error) {
	// Attempt to retrieve and assert the "shop_result" from the invoice map
	Data, ok := invoice["shop_result"]
	if !ok {
		return nil, fmt.Errorf("key shop_result not found in invoice map")
	}

	shopData, ok := Data.(primitive.M)
	if !ok {
		return nil, fmt.Errorf("shop_result is of type %T, expected map[string]interface{}", Data)
	}

	// Attempt to retrieve and assert the "location" from the shopData map
	shopAddress, ok := shopData["location"].(primitive.M)

	if !ok {
		return nil, fmt.Errorf("error asserting location as map[string]interface{}")
	}

	// Attempt to retrieve and assert the "billing_details_result" as a slice of map[string]interface{}
	billingProducts, ok := invoice["billing_details_result"].(primitive.A)
	if !ok {
		return nil, fmt.Errorf("billing_details_result is of type %T, expected map[string]interface{}", invoice["billing_details_result"])
	}

	companyInfo := "No.10, 1st Street Valliammal Nagar Valasaravakkam, Chennai-600087, Tamil Nadu"
	CompanyContactNumber := "044 4852 1151"
	CompanyEmailId := "customercare@sakthipharma.com"

	pdf := GoPdf{gofpdf.New("P", "mm", "A4", "")}

	// Add a new page
	pdf.AddPage()
	incNo := ToString(GetNextSeqNumber(orgId, "SHOPINV"))
	// Set fonts
	pdf.SetFont("Arial", "B", 16)
	sakthiPharmaWidth := pdf.GetStringWidth("TAX INVOICE")
	pdf.SetXY(210-sakthiPharmaWidth-10, 10) // Right align "TAX INVOICE"
	pdf.Cell(sakthiPharmaWidth, 10, "TAX INVOICE")
	pdf.SetXY(210-sakthiPharmaWidth-10, 15)
	pdf.SetFont("Arial", "B", 8)
	pdf.Cell(sakthiPharmaWidth, 10, "DL No: TN-05-20-00267 & 21")
	pdf.SetXY(210-sakthiPharmaWidth-10, 19)
	pdf.Cell(sakthiPharmaWidth, 10, "GSTIN: 33AAECJ6856G1ZG")
	pdf.SetXY(10, 10) // Left align "SAKTHI PHARMA"
	pdf.ImageOptions("sakthi.png", pdf.GetX(), pdf.GetY(), 40, 15, false, gofpdf.ImageOptions{ReadDpi: true}, 0, "")
	pdf.Ln(12)

	// Company Info
	pdf.SetFont("Arial", "", 10)
	pdf.SetXY(10, pdf.GetY()+5) // Reset position
	pdf.MultiCell(60, 6, companyInfo, "", "", false)
	pdf.Ln(3)
	pdf.ImageOptions("phone.png", pdf.GetX(), pdf.GetY(), 5, 0, false, gofpdf.ImageOptions{ReadDpi: true}, 0, "")
	pdf.Cell(60, 6, "       "+CompanyContactNumber)
	pdf.Ln(6)
	pdf.ImageOptions("./mail.png", pdf.GetX(), pdf.GetY(), 5, 0, false, gofpdf.ImageOptions{ReadDpi: true}, 0, "")
	pdf.Cell(60, 6, "       "+CompanyEmailId)
	pdf.Ln(12)

	// Order and Invoice details
	pdf.SetFont("Arial", "B", 12)
	startX := pdf.GetX() // Save start X position
	startY := pdf.GetY() // Save start Y position
	fmt.Println(startX, startY, "xy")
	pdf.SetXY(startX, startY) // Set to saved X and Y positions
	pdf.Cell(95, 10, "Bill # : "+invoice["_id"].(string))
	pdf.SetXY(startX+135, startY)
	pdf.Cell(95, 10, "Invoice # : "+incNo)
	pdf.Ln(6)
	pdf.SetXY(startX, pdf.GetY()) // Reset X position for dates
	pdf.SetFont("Arial", "", 10)
	billCreatedOn := invoice["created_on"].(primitive.DateTime)

	// Convert primitive.DateTime to time.Time
	createdOnTime := billCreatedOn.Time()

	// Format the date as a string (e.g., "YYYY-MM-DD")
	dateString := createdOnTime.Format("02-01-2006")
	pdf.Cell(95, 10, "Date : "+dateString)
	pdf.SetXY(startX+135, pdf.GetY()) // Move to next column for Invoice Date
	pdf.Cell(95, 10, "Date : "+ToString(time.Now().Format("02-01-2006")))
	pdf.Ln(12)

	// Shipping Address
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(pdf.GetX(), pdf.GetY()) // Reset position
	pdf.Cell(95, 10, "Shop Address:")
	pdf.SetXY(startX+135, pdf.GetY()) // Adjust Y to align with Shipping Address
	pdf.Cell(95, 10, "Billing Address:")
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)
	y := pdf.GetY()
	pdf.SetXY(pdf.GetX(), y) // Reset position
	pdf.MultiCell(95, 6, fmt.Sprintf("Shop Name: %s\n%s\nMobile No: %s", shopData["shop_name"], shopAddress["street"], shopData["mobile_number"]), "", "", false)
	pdf.SetXY(pdf.GetX()+135, y) // Reset to the same starting X position
	pdf.MultiCell(95, 6, fmt.Sprintf("Customer Name: %s\n%s\nMobile No: %s", invoice["customer_name"].(string), invoice["customer_address"], invoice["customer_mobile_no"]), "", "", false)
	// Billing Address

	pdf.Ln(12)

	// Table header
	// pdf.SetFont("Arial", "B", 10)
	// pdf.SetXY(startX, pdf.GetY()) // Reset position for table header
	// pdf.CellFormat(10, 10, "#", "1", 0, "C", false, 0, "")
	// pdf.CellFormat(30, 10, "HSNC", "1", 0, "C", false, 0, "")
	// pdf.CellFormat(50, 10, "Description", "1", 0, "C", false, 0, "")
	// pdf.CellFormat(20, 10, "MRP", "1", 0, "C", false, 0, "")
	// pdf.CellFormat(20, 10, "Price", "1", 0, "C", false, 0, "")
	// pdf.CellFormat(10, 10, "Qty", "1", 0, "C", false, 0, "")
	// pdf.CellFormat(20, 10, "Amount", "1", 0, "C", false, 0, "")
	// pdf.CellFormat(10, 10, "GST%", "1", 0, "C", false, 0, "")
	// pdf.CellFormat(20, 10, "GST Total", "1", 1, "C", false, 0, "")
	// Table content

	pdf.SetFont("Arial", "", 10)

	headersMap := []string{"#", "HSNC", "Description", "MRP", "Price", "Qty", "Amount", "GST%", "GST", "Total"}
	headersWithDataType := convertHeaders(headersMap)
	finalHeadersData := GetWidthForPdfTable(headersWithDataType, headersMap)

	// for i, productData := range billingProducts {
	// 	product := productData.(primitive.M)
	//
	// 	purchaseData, ok := product["purchase_details"]
	// 	if !ok {
	// 		return nil, fmt.Errorf("key shop_result not found in invoice map")
	// 	}

	// 	originalProductData, ok := purchaseData.(primitive.M)

	// 	if !ok {
	// 		return nil, fmt.Errorf("shop_result is of type %T, expected map[string]interface{}", Data)
	// 	}
	// 	productName := originalProductData["name"]
	// 	productExpiryDate := product["expiry_date"]
	// 	batchNumber := product["batch_number"]
	// 	description := productName.(string) + " " + productExpiryDate.(string) + " " + batchNumber.(string)

	// 	pdf.CellFormat(10, 10, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
	// 	pdf.CellFormat(30, 10, product["hsnc_number"].(string), "1", 0, "C", false, 0, "")
	// 	// pdf.CellFormat(50, 10, description, "1", 0, "L", false, 0, "")
	// 	//pdf.SetX(pdf.GetX()) // Reset X position
	// 	pdf.MultiCell(50, 6, description, "1", "L", false)
	// 	//pdf.SetX(pdf.GetX())
	// 	pdf.CellFormat(20, 10, ToString(ConvertToDataType("float64", ToString(product["mrp"]))), "1", 0, "R", false, 0, "")
	// 	pdf.CellFormat(20, 10, ToString(ConvertToDataType("float64", ToString(product["amount"]))), "1", 0, "R", false, 0, "")
	// 	pdf.CellFormat(10, 10, ToString(product["quantity"]), "1", 0, "C", false, 0, "")
	// 	pdf.CellFormat(20, 10, ToString(ConvertToDataType("float64", ToString(product["amount"]))), "1", 0, "R", false, 0, "")
	// 	pdf.CellFormat(10, 10, ToString(ConvertToDataType("float64", ToString(product["gst_percentage"]))), "1", 0, "C", false, 0, "")
	// 	pdf.CellFormat(20, 10, ToString(ConvertToDataType("float64", ToString(product["amount_with_gst"]))), "1", 1, "R", false, 0, "")
	// }

	pdf.addTable1(finalHeadersData, billingProducts)
	pdf.Ln(8)
	// GST Tax Details
	pdf.SetFont("Arial", "B", 10)
	pageWidth, _ := pdf.GetPageSize()
	cellWidth := 50.0
	//cellHeight := 10.0

	// Calculate X position to center the cell
	centerX := (pageWidth - cellWidth) / 2
	pdf.SetXY(centerX, pdf.GetY()) // Reset position for GST details
	pdf.CellFormat(50, 10, "GST Tax Details", "0", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(30, 10, "Value", "1", 0, "C", false, 0, "") //1

	pdf.CellFormat(30, 10, "CGST%", "1", 0, "C", false, 0, "")    //2
	pdf.CellFormat(30, 10, "CGST Amt", "1", 0, "C", false, 0, "") //3
	pdf.CellFormat(30, 10, "SGST%", "1", 0, "C", false, 0, "")    //4
	pdf.CellFormat(30, 10, "SGST Amt", "1", 0, "C", false, 0, "") //5
	pdf.CellFormat(30, 10, "Tax Amt", "1", 1, "C", false, 0, "")  //6

	pdf.CellFormat(30, 10, ToString(ConvertToDataType("float64", ToString(invoice["amount"]))), "1", 0, "C", false, 0, "") //1

	pdf.CellFormat(30, 10, "6", "1", 0, "C", false, 0, "")                                                                      //2
	pdf.CellFormat(30, 10, ToString(ConvertToDataType("float64", ToString(invoice["cgst_amount"]))), "1", 0, "C", false, 0, "") //3 cgst_amount
	pdf.CellFormat(30, 10, "6", "1", 0, "C", false, 0, "")                                                                      //4
	pdf.CellFormat(30, 10, ToString(ConvertToDataType("float64", ToString(invoice["sgst_amount"]))), "1", 0, "C", false, 0, "") //5

	taxAmount := (invoice["sgst_amount"].(float64) + invoice["cgst_amount"].(float64))
	formatedTax := fmt.Sprintf("%.2f", taxAmount)
	pdf.CellFormat(30, 10, formatedTax, "1", 1, "C", false, 0, "") //6

	pdf.Ln(12)

	// Footer

	pdf.SetFont("Arial", "I", 8)
	pdf.SetXY(startX, pdf.GetY()) // Reset position for footer
	pdf.CellFormat(190, 10, "All disputes related to this order are subject to the jurisdiction of courts at Chennai, Tamil Nadu.", "0", 1, "C", false, 0, "")
	pdf.Ln(12)
	pdf.SetFont("Arial", "B", 12)
	signX := pdf.GetX()
	signY := pdf.GetY()
	pdf.SetXY(signX, signY)
	//pdf.CellFormat(80, 10, "Sakthi Pharma Ltd", "0", 1, "L", false, 0, "")
	//pdf.Cell(95, 10, "Sakthi Pharma Ltd")

	pdf.CellFormat(190, 10, "Sakthi Pharma Ltd", "0", 1, "R", false, 0, "")
	pdf.Ln(12)
	pdf.CellFormat(190, 10, "_______________________", "0", 1, "R", false, 0, "")
	pdf.Ln(6)
	//pdf.SetXY(160, signY)
	pdf.CellFormat(190, 10, "Pharmacist Signature", "0", 1, "R", false, 0, "")
	pdf.Ln(12)
	pdf.CellFormat(190, 10, "_______________________", "0", 1, "R", false, 0, "")

	currentTime := time.Now()
	formattedTime := currentTime.Format("02-01-2006 15:04")

	// Replace spaces, dashes, and colons with underscores
	safeTime := strings.NewReplacer(" ", "_", "-", "_", ":", "_").Replace(formattedTime)

	filePath := fmt.Sprintf("/uploads/system/shop_invoice/%s__%s.pdf", "INV__"+incNo, safeTime)

	// Output the PDF (assuming you have a valid pdf object)

	var buffer bytes.Buffer
	err := pdf.Output(&buffer)
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	// Get the PDF content as []byte
	pdfContent := buffer.Bytes()

	// Create a *multipart.FileHeader from the opened file
	// fileHeader := &multipart.FileHeader{
	// 	Filename: "bank_statement.pdf",
	// }

	link, err := UploadFile("sakthipharma", filePath, "", "Shop InVoice", pdfContent)
	if err != nil {
		return nil, Unexpected(err.Error())
	}
	// filePath = "/uploads/system/shop_invoice/INCV_" + incNo
	// // err := pdf.OutputFileAndClose("invoice.pdf")

	// if err != nil {
	// 	panic(err)
	// }

	// Get the size of the generated PDF file
	// fileInfo, err := os.Stat(filePath)

	// if err != nil {
	// 	panic(err)
	// }

	//fileSize := fileInfo.Size()

	id := uuid.New().String()
	fileName := filepath.Base(filePath)

	apiResponse := bson.M{
		"_id":          id,
		"category":     "shop_invoice",
		"file_name":    incNo + ".pdf",
		"storage_name": incNo + "__" + safeTime,
		"extn":         filepath.Ext(fileName),
		"file_path":    link,
		"active":       "Y",
	}

	// InsertData(c, orgId, "shop_invoice", apiResponse)
	return apiResponse, nil
}

func (pdf *GoPdf) addTable1(headers []map[string]interface{}, data []interface{}) {

	const marginCell = 2.0

	_, pageHeight := pdf.GetPageSize()
	_, _, _, marginBottom := pdf.GetMargins()

	pdf.header(headers)
	//var isEvenRow bool
	// columnWidth := math.Ceil(190 / float64(len(headers)))
	for _, rowInterface := range data {

		// // Toggle the flag for even and odd rows
		// isEvenRow = i%2 == 0

		// if isEvenRow {
		// 	// Set color for even rows
		// 	pdf.SetFillColor(240, 240, 240) // Example: Light Gray
		// } else {
		// 	// Set color for odd rows
		// 	pdf.SetFillColor(255, 255, 255) // Example: White
		// }

		curX, y := pdf.GetXY()
		x := curX
		totalHeight := 0.0
		var rowCount = 0
		if row, ok := rowInterface.(primitive.M); ok {
			for _, header := range headers {
				columnWidth := header["width"].(float64)
				str := header["fieldName"].(string)
				allHeaders := header["totalHeaders"].([]string)
				if value, exists := row[str]; exists {
					var valueAlign string
					switch value.(type) {
					case string:
						valueAlign = "L"
					case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
						valueAlign = "R"
					case float32, float64:
						valueAlign = "R"
						value = fmt.Sprintf("%.2f", value)
					default:
						valueAlign = "L"
					}
					stringValue := fmt.Sprintf("%v", value)
					_, lineHeight := pdf.GetFontSize()
					//	lastCellWidth := math.Ceil(pdf.GetStringWidth(stringValue))
					maxHeight := pdf.CreateBorderBasedValue(row, allHeaders, columnWidth)

					totalHeight = math.Max(totalHeight, maxHeight)

					if pdf.GetY()+totalHeight > pageHeight-marginBottom {
						pdf.SetDrawColor(0, 0, 0)
						pdf.AddPage()
						pdf.header(headers)
						y = pdf.GetY()
						curX, _ = pdf.GetXY()
						x = curX

					}

					if str == "#" {
						rowCount = rowCount + 1
						stringValue = ToString(rowCount)
					}
					if str == "Transaction Type" {

						if value == "Cr" {
							pdf.SetTextColor(0, 128, 0) // Green for Credit
						} else if value == "Dr" {
							pdf.SetTextColor(255, 0, 0) // Red for Debit
						} else {
							pdf.SetTextColor(0, 0, 0) // Reset text color for other cells
						}
					} else {
						pdf.SetTextColor(0, 0, 0) // Reset text color for other cells
					}
					pdf.SetDrawColor(200, 200, 200)
					// tempColumnWidth := columnWidth
					// if str == "Description" {
					// 	columnWidth = columnWidth + 10
					// }
					// Draw the background color for the entire cell

					pdf.Rect(x, y, columnWidth, totalHeight, "FD")

					pdf.MultiCell(columnWidth, lineHeight+marginCell, stringValue, "", valueAlign, false)
					x += columnWidth
					pdf.SetXY(x, y)
					//columnWidth = tempColumnWidth

				}
			}
		}

		pdf.SetXY(curX, y+totalHeight)
	}

	pdf.footer()
}

func (pdf *GoPdf) header(hdr []map[string]interface{}) *GoPdf {
	// pdf.SetFont("Times", "B", 16)
	pdf.SetFillColor(240, 240, 240)

	// columnWidth := math.Ceil(190 / float64(len(hdr)))
	cellHeight := 7.0
	for _, strData := range hdr {

		pdf.SetDrawColor(200, 200, 200)
		pdf.SetFillColor(173, 216, 230)
		columnWidth := strData["width"].(float64)
		str := strData["fieldName"].(string)

		lastCellWidth := math.Ceil(pdf.GetStringWidth(str))
		if lastCellWidth > columnWidth {
			cellHeight = cellHeight + 2
		}
		// tempColumnWidth := columnWidth
		// if str == "Description" {
		// 	columnWidth = columnWidth + 10
		// }

		pdf.CellFormat(columnWidth, cellHeight, str, "1", 0, "C", true, 0, "")
		pdf.SetFillColor(240, 240, 240)
		//columnWidth = tempColumnWidth
	}

	pdf.Ln(-1)
	return pdf
}

func (pdf *GoPdf) footer() {
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		// pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(0, 0, fmt.Sprintf("Page %d", pdf.PageNo()), "0", 0, "R", false, 0, "")
	})
}

// func (pdf *GoPdf) CreateBorderBasedValue(row map[string]interface{}, headers []string, cellWidth float64) float64 {
// 	maxHeight := 0.0

// 	marginCell := 2.0
// 	for _, header := range headers {
// 		if value, exists := row[header]; exists {
// 			stringValue := fmt.Sprintf("%v", value)
// 			_, lineHeight := pdf.GetFontSize()
// 			lines := pdf.SplitLines([]byte(stringValue), cellWidth)

// 			cellHeight := float64(len(lines))*lineHeight + marginCell*float64(len(lines))

// 			maxHeight = math.Max(maxHeight, cellHeight)

// 		}
// 	}

//		return maxHeight
//	}
func (pdf *GoPdf) CreateBorderBasedValue(row map[string]interface{}, headers []string, cellWidth float64) float64 {
	maxHeight := 0.0

	for _, header := range headers {
		if value, exists := row[header]; exists {
			stringValue := fmt.Sprintf("%v", value)
			stringWidth := pdf.GetStringWidth(stringValue)
			_, lineHeight := pdf.GetFontSize()
			// Calculate the number of lines needed
			numberOfLines := math.Ceil(stringWidth / cellWidth)

			// Calculate the total cell height
			cellHeight := numberOfLines * lineHeight * 1.8

			maxHeight = math.Max(maxHeight, cellHeight)
		}
	}

	return maxHeight
}
func GetWidthForPdfTable(data []map[string]interface{}, totalAllHeaders []string) []map[string]interface{} {
	overallTableWidth := 190.0                 // Total width available for the table
	columnWidths := make([]float64, len(data)) // Array to hold the width of each column
	//countedWidth := 0
	// Iterate over the data to calculate the width for each column
	for i, obj := range data {
		value := obj["fieldName"].(string)
		dataType := obj["dataType"].(string)
		multicell := obj["multicell"].(bool)

		switch dataType {
		case "string":
			// Assuming an average character width for string data
			if multicell {
				columnWidths[i] = float64(len(value)) * 2.4
			} else {
				strLen := float64(len(value))
				if strLen > 15 {
					columnWidths[i] = strLen
				} else {
					columnWidths[i] = 18
				}
			}
		case "int", "float":
			// Setting a fixed width for numeric data
			columnWidths[i] = 18.0
		case "date":
			// Setting a fixed width for date data
			columnWidths[i] = 25.0
		default:
			// Default width for unknown data types
			columnWidths[i] = 15.0
		}
	}

	// Normalize column widths to fit within the overallTableWidth
	totalWidth := 0.0
	for _, width := range columnWidths {
		totalWidth += width
	}

	if totalWidth > overallTableWidth {
		scaleFactor := overallTableWidth / totalWidth
		for i := range columnWidths {
			columnWidths[i] *= scaleFactor
		}
	}

	// Create the result slice with updated widths
	result := make([]map[string]interface{}, len(data))
	for i, obj := range data {

		result[i] = map[string]interface{}{
			"fieldName":    obj["fieldName"],
			"dataType":     obj["dataType"],
			"multicell":    obj["multicell"],
			"totalHeaders": totalAllHeaders,
			"width":        math.Ceil(columnWidths[i]), // Round up to the nearest integer
		}
	}

	return result
}

func convertHeaders(headers []string) []map[string]interface{} {
	// Define a map of data types for each header
	dataTypes := map[string]string{
		"#":           "int",
		"HSNC":        "string",
		"Description": "string",
		"MRP":         "float",
		"Price":       "float",
		"Qty":         "int",
		"Amount":      "float",
		"GST%":        "float",
		"GST":         "float",
		"Total":       "float",
	}

	// Define a map to specify whether each header should use multicell
	multicellMap := map[string]bool{
		"#":           false,
		"HSNC":        false,
		"Description": true,
		"MRP":         false,
		"Price":       false,
		"Qty":         false,
		"Amount":      false,
		"GST%":        false,
		"GST":         false,
		"Total":       false,
	}

	headersWithDataTypes := make([]map[string]interface{}, len(headers))

	for i, header := range headers {
		headersWithDataTypes[i] = map[string]interface{}{
			"fieldName": header,
			"dataType":  dataTypes[header],
			"multicell": multicellMap[header],
		}
	}

	return headersWithDataTypes
}
