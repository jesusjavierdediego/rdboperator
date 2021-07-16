package mongodb

import (
	"context"
	"fmt"
	"time"
	"encoding/json"
	"strings"
	utils "xqledger/rdboperator/utils"
	configuration "xqledger/rdboperator/configuration"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	//"go.mongodb.org/mongo-driver/bson/primitive"
)

const componentMessage = "MongoDB Client"

var config = configuration.GlobalConfiguration
var client *mongo.Client = nil
var ctx context.Context


func getRDBClient() (*mongo.Client, error) {
	methodMsg := "getRDBClient"
	if client != nil {
		utils.PrintLogInfo(componentMessage, methodMsg, "Existing MongoDB Client obtained OK")
		return client, nil
	}
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/admin?authSource=admin",
		config.Rdb.Username,
		config.Rdb.Password,
		config.Rdb.Host,
		27017,
	)
	if ctx == nil {
		c, _ := context.WithTimeout(context.Background(), time.Duration(config.Rdb.Timeout) * time.Second)
		ctx = c
	}
	clientOptions := options.Client().ApplyURI(uri)
	clientOptions = clientOptions.SetMaxPoolSize(uint64(config.Rdb.Poolsize))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		utils.PrintLogError(err, componentMessage, methodMsg, "Error connecting to MongoDB")
		return nil, err
	}
	utils.PrintLogInfo(componentMessage, methodMsg, "New MongoDB Client obtained OK")
	return client, nil
}

/*
The result is returned in the shape of an array of maps (key: string, value: any type)
*/
func RunQuery(dbName string, colName string, query string) ([]map[string]interface{}, error) {
	methodMsg := "RunQuery"
	rdbClient, err := getRDBClient()
	if err != nil {
		utils.PrintLogError(err, componentMessage, methodMsg, "Error getting MongoDB client")
		return nil, err
	}
	col := rdbClient.Database(dbName).Collection(colName)
	cursor, err := col.Find(context.TODO(), bson.D{})
	if err != nil {
		utils.PrintLogError(err, componentMessage, methodMsg, "Finding all documents error")
		defer cursor.Close(ctx)
	}
	var resultSet []map[string]interface{}

	for cursor.Next(ctx) {
		var result bson.M
		err := cursor.Decode(&result)
		if err != nil {
			utils.PrintLogError(err, componentMessage, methodMsg, "Reading cursor decoding error")
		} else {
			//mongoId := result["_id"]
			//mongoIdAsStr := mongoId.(primitive.ObjectID).Hex()
			var r = make(map[string]interface{})
			for k, v := range result {
				// if k == "_id" {
				// 	r[k] = mongoIdAsStr
				// } else {
				// 	r[k] = v
				// }
				r[k] = v
			}
			resultSet = append(resultSet, r)
		}
	}
	defer cursor.Close(ctx)
	utils.PrintLogInfo(componentMessage, methodMsg, fmt.Sprintf("Records in collection %s, database %s obtained OK", colName, dbName))
	return resultSet, nil
}

func getID(m map[string]interface{}) string {
	var id = ""
	for k, v := range m {
		if k == "_id" {
			id = fmt.Sprintf("%v", v)
		} 
	}
	return id
}

func HandleEvent(event utils.RecordEvent) {
	methodMsg := "HandleEvent"
	utils.PrintLogInfo(componentMessage, methodMsg, "Event received to be handled in the RDB")
	var recordAsMap = make(map[string]interface{})
	mapErr := json.Unmarshal([]byte(event.RecordContent), &recordAsMap)
	if mapErr != nil {
		utils.PrintLogError(mapErr, componentMessage, methodMsg, "Error unmarshaling record to map")
	} else {
		mapErr := json.Unmarshal([]byte(event.RecordContent), &recordAsMap)
		if mapErr != nil {
			utils.PrintLogError(mapErr, componentMessage, methodMsg, "Error unmarshaling record to map")
		}
		switch t := event.OperationType; t {
			case "new":
				insertRecord(event.DBName, event.Group, "", recordAsMap)
			case "update":
				id := getID(recordAsMap)
				updateRecord(event.DBName, event.Group, id, recordAsMap)
			case "delete":
				id := getID(recordAsMap)
				deleteRecord(event.DBName, event.Group, id)
			default:
				utils.PrintLogInfo(componentMessage, methodMsg, fmt.Sprintf("Operation not supported: %s", t))
		}
	}
}

func insertRecord(dbName string, colName string, _id string, recordAsMap map[string]interface{}) (string, error) {
	methodMsg := "insertRecord"
	rdbClient, err := getRDBClient()
	if err != nil {
		utils.PrintLogError(err, componentMessage, methodMsg, "Error getting MongoDB client")
		return "", err
	}
	cleanDbName := strings.ReplaceAll(dbName, ".", "")
	col := rdbClient.Database(cleanDbName).Collection("main")// Hardcoded for now, onluy one collection
	
	if len(_id) > 0 { // Case for update
		// oid, idErr := primitive.ObjectIDFromHex(_id)
		// fmt.Printf("%s %v", err, _id)
		// if err != nil {
		// 	utils.PrintLogError(idErr, componentMessage, methodMsg, "Error converting provided id: " + _id)
		// 	return "", idErr
		// }
		recordAsMap["_id"] = _id
	} else { // Case for new recvord
		newID, err := utils.GetRDBID()
		if err != nil {
			utils.PrintLogError(err, componentMessage, methodMsg, "Error composing new id")
			return "", err
		}
		recordAsMap["_id"] = newID
	}
	
	if ctx == nil {
		c, _ := context.WithTimeout(context.Background(), time.Duration(config.Rdb.Timeout) * time.Second)
		ctx = c
	}
	result, insertErr := col.InsertOne(ctx, recordAsMap)
	if insertErr != nil {
		utils.PrintLogError(insertErr, componentMessage, methodMsg, "Error inserting record in RDB")
		return "", insertErr
	}
	id := fmt.Sprintf("%v", result.InsertedID) 
	utils.PrintLogInfo(componentMessage, methodMsg, fmt.Sprintf("New record inserted successfully with ID '%s' - Database '%s' - Collection '%s'", id, dbName, colName))
	
	return id, nil
}


func updateRecord(dbName string, colName string, _id string, recordAsMap map[string]interface{}) error {
	methodMsg := "UpdateRecord"
	delErr := deleteRecord(dbName, colName, _id)
	if delErr != nil{
		return delErr
	}
	_, insertErr := insertRecord(dbName, colName, _id, recordAsMap)
	if insertErr != nil {
		utils.PrintLogError(insertErr, componentMessage, methodMsg, fmt.Sprintf("Error updating record with ID '%s' - Database '%s' - Collection '%s'", _id, dbName, colName))
		return insertErr
	}
	utils.PrintLogInfo(componentMessage, methodMsg, fmt.Sprintf("Record inserted successfully with ID '%s' - Database '%s' - Collection '%s'", _id, dbName, colName))
	return nil
}

func deleteRecord(dbName string, colName string, _id string) error {
	methodMsg := "deleteRecord"
	rdbClient, err := getRDBClient()
	if err != nil {
		utils.PrintLogError(err, componentMessage, methodMsg, "Error getting MongoDB client")
		return err
	}
	col := rdbClient.Database(dbName).Collection(colName)
	_, delErr := col.DeleteOne(ctx, _id)
	if delErr != nil {
		utils.PrintLogError(delErr, componentMessage, methodMsg, fmt.Sprintf("Error deleting record with ID '%s' - Database '%s' - Collection '%s'", _id, dbName, colName))
		return err
	}
	utils.PrintLogInfo(componentMessage, methodMsg, fmt.Sprintf("Record deleted successfully with ID '%s' - Database '%s' - Collection '%s'", _id, dbName, colName))
	return nil
}
