package helper

import "time"

var OrgList = make(map[string]Organization)

type Organization struct {
	Id     string      `json:"_id" bson:"_id"`
	Name   string      `json:"name" bson:"name"`
	Type   string      `json:"type" bson:"type"`
	Domain string      `json:"domain" bson:"domain"`
	Group  string      `json:"group" bson:"group"`
	Style  interface{} `json:"style" bson:"style"`
}

type UserToken struct {
	UserId   string `json:"user_id" bson:"user_id"`
	UserRole string `json:"user_role" bson:"user_role"`
	OrgId    string `json:"uo_id" bson:"uo_id"`
	OrgGroup string `json:"uo_group" bson:"uo_group"`
}

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
	Column   string `json:"column" bson:"column"`
	Operator string `json:"operator" bson:"operator"`
	Type     string `json:"type" bson:"type"`
	Value    string `json:"value" bson:"value"`
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

type LookupQuery struct {
	Operation string        `json:"operation" bson:"operation"`
	ParentRef CollectionRef `json:"parent_collection" bson:"parent_collection"`
	ChildRef  CollectionRef `json:"child_collection" bson:"child_collection"`
}

type CollectionRef struct {
	Name   string   `json:"name" bson:"name"`
	Key    string   `json:"key" bson:"key"`
	Filter []Filter `json:"filter,omitempty" bson:"filter,omitempty"`
}
