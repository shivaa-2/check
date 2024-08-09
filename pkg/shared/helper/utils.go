package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"strconv"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetNewInstallCode() string {
	min := 1
	max := 8
	rand.Seed(time.Now().UnixNano())
	fd := rand.Intn(max-min+1) + min
	sd := rand.Intn(max-min+1) + min
	c1 := string(rune(65 + fd))
	c2 := string(rune(65 + fd + sd))
	c5 := string(rune(65 + fd + sd + sd))
	c6 := string(rune(65 + sd + sd))
	return fmt.Sprintf("%s%s%d%d%s%s", c1, c2, fd, sd, c5, c6)
}

func IsValidInstallCode(code string) bool {
	chars := []rune(code)
	d1 := int(chars[2]) - 48
	d2 := int(chars[3]) - 48
	return int(chars[0])+d2 == int(chars[1]) && int(chars[0])+d2+d2 == int(chars[4]) && int(chars[4]) == int(chars[5])+d1
}

func GetNextSeqNumber(orgId string, key string) int32 {
	//update to database
	filter := bson.M{"_id": key}
	updateData := bson.M{
		"$inc": bson.M{"value": 1},
	}
	result, _ := ExecuteFindAndModifyQuery(orgId, "sequence", filter, updateData)
	return result["value"].(int32)
}
func UpdateDateObject(input map[string]interface{}) error {
	for k, v := range input {
		if v == nil {
			continue
		}
		ty := reflect.TypeOf(v).Kind().String()
		if ty == "string" {
			val := reflect.ValueOf(v).String()
			t, err := time.Parse(time.RFC3339, val)
			if err == nil {
				input[k] = t.UTC()
			}
		} else if ty == "map" {
			return UpdateDateObject(v.(map[string]interface{}))
		} else if ty == "slice" {
			for _, e := range v.([]interface{}) {
				if reflect.TypeOf(e).Kind().String() == "map" {
					return UpdateDateObject(e.(map[string]interface{}))
				}
			}
		}
	}
	return nil
}

func Toint64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func Sort(s string) int64 {
	if s == "" {
		return 1
	}
	return -1
}

func Page(s string) int64 {
	if s == "" || s == "0" {
		return Toint64("1")
	}
	return Toint64(s)
}
func ToString(input interface{}) string {
	return fmt.Sprintf("%v", input)
}

func SortOrdering(s string) int {
	switch s {
	case "1":
		return 1
	case "-1":
		return -1
	default:
		return 1
	}
}

func Limit(s string) int64 {
	if s == "" {
		s = GetenvStr("DEFAULT_FETCH_ROWS", "200")
	}
	return Toint64(s)
}

func DocIdFilter(id string) bson.M {
	if id == "" {
		return bson.M{}
	}
	docId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return bson.M{"_id": id}
	} else {
		return bson.M{"_id": docId}
	}
}

func GetNewOtp() int32 {
	rand.Seed(time.Now().UnixNano())
	min := 10000
	max := 99999
	return int32(rand.Intn(max-min+1) + min)
}

func GHttpPost(url string, requestBody []byte) (interface{}, error) {
	r, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("x-client-id", appId)
	r.Header.Add("x-client-secret", secretKey)
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var response interface{}
	derr := json.NewDecoder(res.Body).Decode(&response)
	if derr != nil {
		return nil, derr
	}
	return response, nil
}

func GetRandomUUID() string {
	u4 := uuid.NewV4()
	uuid := uuid.NewV5(u4, "KT")
	return ToString(uuid)
}

func ConvertToDataType(DataType string, Value string) interface{} {
	switch DataType {
	case "int":
		i, err := strconv.Atoi(Value)
		if err != nil {
			return err // or you can return a default value or handle error accordingly
		}
		return i
	case "int64":
		i, err := strconv.ParseInt(Value, 10, 64)
		if err != nil {
			return err
		}
		return i
	case "float64":
		f, err := strconv.ParseFloat(Value, 64)
		if err != nil {
			return err
		}
		return f
	case "bool":
		b, err := strconv.ParseBool(Value)
		if err != nil {
			return err
		}
		return b
	case "date":

		d, err := parseDate(Value)
		if err != nil {
			return err
		}
		return d
	case "string":
		return Value
	default:
		return fmt.Errorf("unsupported data type: %s", DataType)
	}
}

func parseDate(dateStr string) (time.Time, error) {
	layouts := []string{
		"2 Jan 2006",                // dd M yyyy (e.g., 02 Jan 2006)
		"02 Jan 2006",               // dd M yyyy (e.g., 02 Jan 2006)
		"Jan 2, 2006",               // M dd, yyyy (e.g., Jan 2, 2006)
		"January 2, 2006",           // Month dd, yyyy (e.g., January 2, 2006)
		"2 January 2006",            // dd Month yyyy (e.g., 2 January 2006)
		"02 January 2006",           // dd Month yyyy (e.g., 02 January 2006)
		"2/Jan/06",                  // d/Mon/yy
		"2/January/06",              // d/Month/yy
		"2006-01-02T15:04:05Z07:00", // RFC3339
		"2/1/06",                    // d/m/yy
		"2/1/2006",                  // d/m/yyyy
		"02/01/06",                  // dd/mm/yy
		"02/01/2006",
	}
	// []string{
	// 	"2006-01-02",                    // yyyy-mm-dd
	// 	"2006/01/02",                    // yyyy/mm/dd
	// 	"2006-01-02T15:04:05Z07:00",     // RFC3339
	// 	"1/2/06",                        // m/d/yy
	// 	"1/2/2006",                      // m/d/yyyy
	// 	"2 Jan 2006",                    // dd M yyyy (e.g., 02 Jan 2006)
	// 	"02 Jan 2006",                   // dd M yyyy (e.g., 02 Jan 2006)
	// 	"Jan 2, 2006",                   // M dd, yyyy (e.g., Jan 2, 2006)
	// 	"January 2, 2006",               // Month dd, yyyy (e.g., January 2, 2006)
	// 	"2 January 2006",                // dd Month yyyy (e.g., 2 January 2006)
	// 	"02 January 2006",               // dd Month yyyy (e.g., 02 January 2006)
	// 	"06-01-02",                      // yy-mm-dd
	// 	"01-02-2006",                    // mm-dd-yyyy
	// 	"02-01-2006",                    // dd-mm-yyyy
	// 	"2-Jan-06",                      // d-Mon-yy
	// 	"2-January-06",                  // d-Month-yy
	// 	"2/Jan/06",                      // d/Mon/yy
	// 	"2/January/06",                  // d/Month/yy
	// 	"Mon, 02 Jan 2006 15:04:05 MST", // RFC822 with timezone
	// }

	var dateTime time.Time
	var err error

	for _, layout := range layouts {
		dateTime, err = time.Parse(layout, dateStr)
		if err == nil {
			return dateTime, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date")
}
