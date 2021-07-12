package grpc

import (
	"encoding/json"
	"xqledger/rdboperator/utils"
	configuration "xqledger/rdboperator/configuration"
	"xqledger/rdboperator/mongodb"
	//"xqledger/rdboperator/kafka"
	pb "xqledger/rdboperator/protobuf"

	"golang.org/x/net/context"
	"google.golang.org/grpc/status"
)

const componentMessage = "GRPC Server"

var config = configuration.GlobalConfiguration

type RecordQueryService struct {
	query *pb.RDBQuery
}

func NewRecordQueryService(query *pb.RDBQuery) *RecordQueryService {
	return &RecordQueryService{query: query}
}




func (s *RecordQueryService) GetRDBRecords(ctx context.Context, query *pb.RDBQuery) (*pb.RecordSet, error) {
	methodMessage := "GetRDBRecords"
	resultSet, err := mongodb.RunQuery(query.DatabaseName, "main", query.Query) // hardcoded for now, singlecoll per reepo
	if err != nil {
		utils.PrintLogError(err, componentMessage, methodMessage, "Error querying RDB")
		return nil, status.New(14, "Error querying RDB - Reason: "+err.Error()).Err()
	}
	var arrayOfRecordsStr []string
	for _, record := range resultSet {
		recordStr, err := json.Marshal(record)
		if err != nil {
			utils.PrintLogError(err, componentMessage, methodMessage, "Response content cannot be marshaled properly")
			return nil, status.New(15, "Response content cannot be marshaled properly - Reason: "+err.Error()).Err()
		}
		arrayOfRecordsStr = append(arrayOfRecordsStr, string(recordStr))
	}
	result := pb.RecordSet{Records: arrayOfRecordsStr}
	return &result, nil

	return nil, err
}
