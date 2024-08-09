package helper

import (
	// "fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

type PaginationRequest struct {
	//json:"sort" bson:"sort" validate:"omitempty"
	Start        int            `json:"start" bson:"start" validate:"omitempty"`
	End          int            `json:"end" bson:"end" validate:"omitempty"`
	Filter       []FilterClause `json:"filter" bson:"filter" validate:"omitempty"`
	Sort         []SortCriteria `json:"sort" bson:"sort" validate:"omitempty"`
	SwitchOver   bool           `json:"switch_over" bson:"switch_over"`
	IsGridSearch bool           `json:"is_grid_search" bson:"is_grid_search"`
}

type FilterClause struct {
	Clause     string            `json:"clause"`
	Conditions []FilterCondition `json:"conditions,omitempty"`
}

type FilterCondition struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"`
	Type     string      `json:"type,omitempty"`
	Value    interface{} `json:"value,omitempty"`
}

type SortCriteria struct {
	Sort  string `json:"sort"`
	ColID string `json:"colId"`
}

func MasterAggreagationPiepline(request PaginationRequest, c *fiber.Ctx) ([]bson.D, bool) {
	pipeline := []bson.D{}
	var textSearch bool
	// Extract filter conditions from the request
	matchConditions := []bson.M{}

	for _, filter := range request.Filter {

		conditions := []bson.M{}

		for _, condition := range filter.Conditions {

			column := condition.Column
			value := condition.Value
			if condition.Operator == "EQUALS" {
				if condition.Type == "string" || condition.Type == "text" {
					// conditions = append(conditions, bson.M{column: bson.M{"$regex": value.(string)}}) //, "$options": "i"
					// conditions = append(conditions, bson.M{column: bson.D{{"$eq", value.(string)}}})

					conditions = append(conditions, bson.M{column: value})

				} else if condition.Type == "date" {

					dateValue := condition.Value.(string)

					// Parse the date string into a time.Time value
					t, _ := time.Parse(time.RFC3339, dateValue)

					// Calculate the start and end of the day for the given date
					startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
					endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)

					// Create a date range query for the specified day
					dateRange := bson.M{condition.Column: bson.M{
						"$gte": startOfDay,
						"$lte": endOfDay,
					}}
					// fmt.Print(dateRange)
					conditions = append(conditions, dateRange)
				} else if condition.Type == "number" {
					conditions = append(conditions, bson.M{column: value})
				} else {
					conditions = append(conditions, bson.M{column: bson.D{{Key: "$eq", Value: value}}})
				}
			} else if condition.Operator == "NOTEQUAL" {
				if condition.Type == "string" || condition.Type == "text" {
					// Perform a case-insensitive text search for employee IDs with negation
					conditions = append(conditions, bson.M{column: bson.D{{Key: "$ne", Value: value.(string)}}})
				} else if condition.Type == "number" {
					// Filter based on a number field using $ne
					conditions = append(conditions, bson.M{column: bson.M{"$ne": value}})
				} else if condition.Type == "date" {
					dateValue := condition.Value.(string)
					// Parse the date string into a time.Time value
					t, _ := time.Parse(time.RFC3339, dateValue)
					// Calculate the start and end of the day for the given date
					startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
					endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)

					dateRange := bson.M{condition.Column: bson.M{
						"$not": bson.M{
							"$gte": startOfDay,
							"$lte": endOfDay,
						},
					}}
					conditions = append(conditions, dateRange)
				} else {
					conditions = append(conditions, bson.M{column: bson.M{"$ne": value}})
				}
			} else if condition.Operator == "CONTAINS" {
				// if condition.Type == "text" {
				// 	conditions = append(conditions, bson.M{
				// 		"$text": bson.M{
				// 			"$search": value.(string),
				// 		},
				// 	})
				// } else {
				conditions = append(conditions, bson.M{column: bson.M{"$regex": value.(string), "$options": "i"}})
				//}
			} else if condition.Operator == "TEXTSEARCH" {
				// conditions = append(conditions, bson.M{
				// 	"$text": bson.M{
				// 		"$search": value.(string),
				// 	},
				// })
				textSearch = true
				conditions = append(conditions, bson.M{
					"$text": bson.M{
						"$search": value.(string),
					},
				})

			} else if condition.Operator == "IN" {
				// if condition.Type == "text" {
				conditions = append(conditions, bson.M{column: bson.M{"$in": value.([]interface{})}})
				// }
			} else if condition.Operator == "NIN" {
				// if condition.Type == "text" {
				conditions = append(conditions, bson.M{column: bson.M{"$nin": value.([]interface{})}})
				// }
			} else if condition.Operator == "NOTCONTAINS" {
				conditions = append(conditions, bson.M{column: bson.M{"$not": bson.M{"$regex": value.(string)}}})
			} else if condition.Operator == "STARTSWITH" {
				// Perform a case-insensitive text search for employee IDs starting with the specified substring
				conditions = append(conditions, bson.M{column: bson.M{"$regex": "^" + value.(string), "$options": "i"}})

			} else if condition.Operator == "ENDSWITH" {
				// Perform a case-insensitive text search for employee IDs ending with the specified value
				conditions = append(conditions, bson.M{column: bson.M{"$regex": value.(string) + "$", "$options": "i"}})
			} else if condition.Operator == "LESSTHAN" {
				if condition.Type == "date" {
					dateValue := condition.Value.(string)
					//parse the date string into a time.Time value
					t, err := time.Parse(time.RFC3339, dateValue)
					if err == nil {
						conditions = append(conditions, bson.M{column: bson.M{"$lt": time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)}})
					}
				} else {
					conditions = append(conditions, bson.M{column: bson.M{"$lt": value}})
				}
			} else if condition.Operator == "GREATERTHAN" {
				if condition.Type == "date" {
					dateValue := condition.Value.(string)
					// Parse the date string into a time.Time value
					t, _ := time.Parse(time.RFC3339, dateValue)
					// Calculate the start and end of the day for the given date
					startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
					// Create a condition for dates greater than the specified value
					conditions = append(conditions, bson.M{column: bson.M{"$gt": startOfDay}})
				} else {
					conditions = append(conditions, bson.M{column: bson.M{"$gt": value}})
				}
			} else if condition.Operator == "LESSTHANOREQUAL" {
				if condition.Type == "date" {
					dateValue := condition.Value.(string)
					// Parse the date string into a time.Time value
					t, _ := time.Parse(time.RFC3339, dateValue)
					// Calculate the start and end of the day for the given date
					startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
					// Create a condition for dates greater than the specified value
					conditions = append(conditions, bson.M{column: bson.M{"$lte": startOfDay}})
				} else {
					conditions = append(conditions, bson.M{column: bson.M{"$lte": value}})
				}

			} else if condition.Operator == "GREATERTHANOREQUAL" {
				if condition.Type == "date" {
					dateValue := condition.Value.(string)
					// Parse the date string into a time.Time value
					t, _ := time.Parse(time.RFC3339, dateValue)
					// Calculate the start and end of the day for the given date
					startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
					// Create a condition for dates greater than the specified value
					conditions = append(conditions, bson.M{column: bson.M{"$gte": startOfDay}})
				} else {
					conditions = append(conditions, bson.M{column: bson.M{"$gte": value}})
				}

			} else if condition.Operator == "INRANGE" {

				if condition.Type == "date" {
					// Parse the date strings into time.Time values
					dateValues := condition.Value.([]interface{})
					if len(dateValues) == 2 {
						startDateValue := dateValues[0].(string)
						endDateValue := dateValues[1].(string)

						startDate, err1 := time.Parse(time.RFC3339, startDateValue)
						endDate, err2 := time.Parse(time.RFC3339, endDateValue)

						if err1 == nil && err2 == nil {
							// Create a condition for dates within the specified range
							conditions = append(conditions, bson.M{column: bson.M{
								"$gte": startDate,
								"$lte": endDate,
							}})
						}
					}
				} else {
					rangeValues, ok := value.([]interface{})
					if ok && len(rangeValues) == 2 {
						minValue := rangeValues[0]
						maxValue := rangeValues[1]
						conditions = append(conditions, bson.M{column: bson.M{"$gte": minValue, "$lte": maxValue}})
					}
				}
			} else if condition.Operator == "BLANK" {
				conditions = append(conditions, bson.M{column: bson.M{"$exists": false}})

			} else if condition.Operator == "NOTBLANK" {

				conditions = append(conditions, bson.M{column: bson.M{"$exists": true, "$ne": nil}})
			} else if condition.Operator == "EXISTS" {
				conditions = append(conditions, bson.M{column: bson.M{"$exists": value}})
			}

		}

		if filter.Clause == "AND" {
			matchConditions = append(matchConditions, bson.M{"$and": conditions})

		} else if filter.Clause == "OR" {
			matchConditions = append(matchConditions, bson.M{"$or": conditions})

		}
	}

	if len(matchConditions) > 0 {
		pipeline = append(pipeline, bson.D{{"$match",
			bson.M{"$and": matchConditions}}})
	}

	// Extract sorting criteria from the request
	sortConditions := bson.M{}

	for _, sort := range request.Sort {
		sortField := sort.ColID
		order := 1
		if sort.Sort == "desc" {
			order = -1
		}

		sortConditions[sortField] = order
	}

	if len(sortConditions) > 0 {
		pipeline = append(pipeline, bson.D{{"$sort", sortConditions}})
	}

	//If pagination is Start and end is 0 assign default value
	if request.Start == 0 && request.End == 0 {
		request.Start = 0
		request.End = 50000
	}

	//pipeline for totaldocs in this collection and set the limit
	pipeline = append(pipeline, bson.D{
		{
			"$facet",
			bson.D{
				{
					"response",
					bson.A{
						bson.M{"$skip": request.Start},                // Skip records based on the page
						bson.M{"$limit": request.End - request.Start}, // Limit records based on the page size
					},
				},
				{
					"pagination",
					bson.A{
						bson.D{{"$count", "totalDocs"}},
					},
				},
			},
		},
	})

	return pipeline, textSearch
}
