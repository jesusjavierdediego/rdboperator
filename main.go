package main

import (
	configuration "xqledger/rdboperator/configuration"
	"xqledger/rdboperator/kafka"
	utils "xqledger/rdboperator/utils"
)

const componentMessage = "Main process"

func main() {
	config := configuration.GlobalConfiguration

	utils.PrintLogInfo("RDB Operator", componentMessage, "Start listening topic with incoming successful writing events")
	go kafka.StartListeningEvents(config.Kafka.Gitactionbacktopic)
}
