package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	configuration "xqledger/rdboperator/configuration"
	utils "xqledger/rdboperator/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const componentMessage = "MongoDB Client"

var config = configuration.GlobalConfiguration
var client *mongo.Client = nil

func getRDBClient() (*mongo.Client, context.Context, error) {
	methodMsg := "getRDBClient"
	if client != nil {
		utils.PrintLogInfo(componentMessage, methodMsg, "Existing MongoDB Client obtained OK")
		return client, nil, nil
	}
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/TestRepository?authSource=admin&w=majority&retryWrites=true",
		config.Rdb.Username,
		config.Rdb.Password,
		config.Rdb.Host,
		27017,
	)
	c, _ := context.WithTimeout(context.Background(), time.Duration(config.Rdb.Timeout)*time.Second)
	clientOptions := options.Client().ApplyURI(uri)
	clientOptions = clientOptions.SetMaxPoolSize(uint64(config.Rdb.Poolsize))
	client, err := mongo.Connect(c, clientOptions)
	if err != nil {
		utils.PrintLogError(err, componentMessage, methodMsg, "Error connecting to MongoDB")
		return nil, nil, err
	}
	utils.PrintLogInfo(componentMessage, methodMsg, "New MongoDB Client obtained OK")
	return client, c, nil
}

// func getID(m map[string]interface{}) string {
// 	var id = ""
// 	for k, v := range m {
// 		if k == "id" {
// 			id = fmt.Sprintf("%v", v)
// 		}
// 	}
// 	return id
// }

func HandleEvent(event utils.RecordEvent) error {
	methodMsg := "HandleEvent"
	utils.PrintLogInfo(componentMessage, methodMsg, "Event received to be handled in the RDB")
	var recordAsMap = make(map[string]interface{})
	if event.OperationType != "delete" {
		mapErr := json.Unmarshal([]byte(event.RecordContent), &recordAsMap)
		if mapErr != nil {
			utils.PrintLogError(mapErr, componentMessage, methodMsg, "Error unmarshaling record to map")
			return mapErr
		}
	}
	rdbClient, ctx, err := getRDBClient()
	if err != nil {
		utils.PrintLogError(err, componentMessage, methodMsg, "Error getting MongoDB client")
		return err
	} else {
		switch t := event.OperationType; t {
		case "new":
			_, err := insertRecord(rdbClient, ctx, event.DBName, event.Group, event.Id, recordAsMap)
			if err != nil {
				utils.PrintLogError(err, componentMessage, methodMsg, "Error inserting record in RDB")
				return err
			}
		case "update":
			err := updateRecord(rdbClient, ctx, event.DBName, event.Group, event.Id, recordAsMap)
			if err != nil {
				utils.PrintLogError(err, componentMessage, methodMsg, "Error updating record in RDB")
				return err
			}
		case "delete":
			err := deleteRecord(rdbClient, ctx, event.DBName, event.Group, event.Id)
			if err != nil {
				utils.PrintLogError(err, componentMessage, methodMsg, "Error deleting record in RDB")
				return err
			}
		default:
			utils.PrintLogInfo(componentMessage, methodMsg, fmt.Sprintf("Operation not supported: %s", t))
		}
	}
	return nil
}

func insertRecord(client *mongo.Client, ctx context.Context, dbName string, colName string, _id string, recordAsMap map[string]interface{}) (string, error) {
	methodMsg := "insertRecord"
	cleanDbName := strings.ReplaceAll(dbName, ".", "")
	if !(len(colName) > 0) {
		colName = "main"
	}
	col := client.Database(cleanDbName).Collection(colName)

	if len(_id) > 0 {
		oid, _ := primitive.ObjectIDFromHex(_id)
		recordAsMap["_id"] = oid
	} else {
		err := errors.New("ID not provided")
		return "", err
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

func updateRecord(client *mongo.Client, ctx context.Context, dbName string, colName string, _id string, recordAsMap map[string]interface{}) error {
	methodMsg := "updateRecord"
	cleanDbName := strings.ReplaceAll(dbName, ".", "")
	if !(len(colName) > 0) {
		colName = "main"
	}
	col := client.Database(cleanDbName).Collection(colName)

	if len(_id) > 0 { // Case for update
		oid, idErr := primitive.ObjectIDFromHex(_id)
		if idErr != nil {
			utils.PrintLogError(idErr, componentMessage, methodMsg, "Error converting provided id: "+_id)
			return idErr
		}
		recordAsMap["_id"] = oid
		_, replaceErr := col.ReplaceOne(ctx, bson.M{"_id": oid}, recordAsMap)
		if replaceErr != nil {
			utils.PrintLogError(replaceErr, componentMessage, methodMsg, "Error inserting record in RDB")
			return replaceErr
		}
		utils.PrintLogInfo(componentMessage, methodMsg, fmt.Sprintf("Record replaced successfully with ID '%s' - Database '%s' - Collection '%s'", _id, dbName, colName))
		return nil
	} else { // Case for new record
		err := errors.New("ID not provided")
		utils.PrintLogError(err, componentMessage, methodMsg, "ID record not provided")
		return err
	}

	return nil
}

func deleteRecord(client *mongo.Client, ctx context.Context, dbName string, colName string, _id string) error {
	methodMsg := "deleteRecord"
	if !(len(colName) > 0) {
		colName = "main"
	}
	col := client.Database(dbName).Collection(colName)
	_, delErr := col.DeleteOne(ctx, bson.M{"_id": _id})
	if delErr != nil {
		utils.PrintLogError(delErr, componentMessage, methodMsg, fmt.Sprintf("Error deleting record with ID '%s' - Database '%s' - Collection '%s'", _id, dbName, colName))
		return delErr
	}
	utils.PrintLogInfo(componentMessage, methodMsg, fmt.Sprintf("Record deleted successfully with ID '%s' - Database '%s' - Collection '%s'", _id, dbName, colName))
	return nil
}
