package database

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConnObject struct {
	OrgId  string `json:"org_id" bson:"org_id"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	DbName string `json:"db_name" bson:"db_name"`
	UserId string `json:"user_id" bson:"user_id"`
	Pwd    string `json:"pwd"`
}

var DBConnections = make(map[string]*mongo.Database)

// By default create shared db connection
var SharedDB *mongo.Database

func Init() {
	SharedDB = CreateDBConnection(GetenvStr("MONGO_SHAREDDB_HOST"), GetenvInt("MONGO_SHAREDDB_PORT"), GetenvStr("MONGO_SHAREDDB_NAME"), GetenvStr("MONGO_SHAREDDB_USER"), GetenvStr("MONGO_SHAREDDB_PASSWORD"))
}

func GetConnection(orgId string) *mongo.Database {
	//Check whether organization specific connection available or not
	//if available return the same
	if connection, exists := DBConnections[orgId]; exists {
		return connection
	}
	//Connection not exist, so we need to create new connection
	var config ConnObject
	err := SharedDB.Collection("db_config").FindOne(context.Background(), bson.M{"org_id": orgId}).Decode(&config)
	fmt.Println(config)
	if err != nil {
		//if there is any problem or specific org config missing, by defualt return shared db
		return SharedDB
	}
	DBConnections[orgId] = CreateDBConnection(config.Host, config.Port, config.DbName, config.UserId, config.Pwd)
	log.Printf("New DB Connection created for %s", orgId)
	return DBConnections[orgId]
}

func CreateDBConnection(host string, port int, dbName string, userid string, pwd string) *mongo.Database {
	dbUrl := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?retryWrites=true&authSource=admin&w=majority&authMechanism=SCRAM-SHA-256", userid, pwd, host, port, dbName)
	//dbUrl := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?retryWrites=true&w=majority&authMechanism=SCRAM-SHA-256", userid, pwd, host, port, dbName)

	//fmt.Println(dbUrl)
	// credential := options.Credential{
	// 	AuthMechanism: "SCRAM-SHA-256",
	// 	AuthSource:    "admin",
	// 	Username:      userid,
	// 	Password:      pwd,
	// }
	client, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI(dbUrl),
		//.SetAuth(credential),
	)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Printf("DB Ping Error")
		log.Fatal(err)
		return nil
	}
	return client.Database(dbName)
}
