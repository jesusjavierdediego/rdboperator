package main

import (
	"fmt"
	"net"
	"strconv"
	configuration "xqledger/rdboperator/configuration"
	grpcserver "xqledger/rdboperator/grpc"
	pb "xqledger/rdboperator/protobuf"
	"xqledger/rdboperator/kafka"
	utils "xqledger/rdboperator/utils"
	"google.golang.org/grpc"
)

const componentMessage = "Main process"

func main() {
	config := configuration.GlobalConfiguration

	utils.PrintLogInfo("RDB Reader", componentMessage, "Start listening topic with incoming successful writing events")
	go kafka.StartListeningEvents(config.Kafka.Gitactionbacktopic)

	
	//Start gRPC service's server
	grpcPort := config.GrpcServer.Port
	listener, listenerErr := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if listenerErr != nil {
		utils.PrintLogError(listenerErr, componentMessage, "Starting", "Error")
	}
	utils.PrintLogInfo(componentMessage, "Starting", "Starting RDB operator gRPC services on port "+strconv.Itoa(grpcPort))
	service := pb.RecordQueryServiceServer(&grpcserver.RecordQueryService{})
	server := grpc.NewServer()
	pb.RegisterRecordQueryServiceServer(server, service)

	if err := server.Serve(listener); err != nil {
		utils.PrintLogError(listenerErr, componentMessage, "Grpc Server start", "Error")
	}
}
