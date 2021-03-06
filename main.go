package main

import (
	"fmt"
	"net"
	"strconv"
	//"time"
	configuration "xqledger/gitreader/configuration"
	grpcserver "xqledger/gitreader/grpc"
	pb "xqledger/gitreader/protobuf"
	//scheduled "xqledger/gitreader/scheduled"
	utils "xqledger/gitreader/utils"

	"google.golang.org/grpc"
)

const componentMessage = "Main process"

// func startScheduledTasks(c configuration.Configuration) {
// 	methodMessage := "startScheduledTasks"
// 	for true {
// 		time.Sleep(time.Duration(c.Scheduledfreq) * time.Hour)
// 		utils.PrintLogInfo(componentMessage, methodMessage, "Scheduled action to detect expired ID Packs starts: %s")
// 		scheduled.ReviewIDPacks()
// 	}
// }

func main() {
	config := configuration.GlobalConfiguration
	// Start scheduled tasks
	/*go startScheduledTasks(config)
	go func() {
		for msg := range utils.LTRTasks {
			utils.PrintLogInfo("Main", "Start services", "LRT channel - Message: "+msg)
		}
	}()*/

	//Start gRPC service's server
	grpcPort := config.GrpcServer.Port
	listener, listenerErr := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if listenerErr != nil {
		utils.PrintLogError(listenerErr, componentMessage, "Starting", "Error")
	}
	utils.PrintLogInfo(componentMessage, "Starting", "Starting Git Reasder gRPC services on port "+strconv.Itoa(grpcPort))
	service := pb.RecordHistoryServiceServer(&grpcserver.RecordHistoryService{})
	server := grpc.NewServer()
	pb.RegisterRecordHistoryServiceServer(server, service)

	/*
	service := pb.DigitalIdentityServiceServer(&grpcserver.DigitalIdentityService{})
	server := grpc.NewServer()
	pb.RegisterDigitalIdentityServiceServer(server, service)
	*/

	if err := server.Serve(listener); err != nil {
		utils.PrintLogError(listenerErr, componentMessage, "Grpc Server start", "Error")
	}
}
