package utils

import (
	config "xqledger/gitreader/configuration"

	logger "github.com/sirupsen/logrus"
)

var Configuration config.Configuration

func PrintLogError(err error, comp string, phase string, errorMessage string) bool {
	logger.WithFields(logger.Fields{
		"Time":      GetFormattedNow(),
		"Component": comp,
		"Phase":     phase,
		"Error":     err,
	}).Error(errorMessage)
	return true
}

func PrintLogWarn(err error, comp string, phase string, errorMessage string) bool {
	logger.WithFields(logger.Fields{
		"Time":      GetFormattedNow(),
		"Component": comp,
		"Phase":     phase,
		"Error":     err,
	}).Warn(errorMessage)
	return true
}

func PrintLogInfo(comp string, phase string, message string) bool {
	logger.WithFields(logger.Fields{
		"Time":      GetFormattedNow(),
		"Component": comp,
		"Phase":     phase,
	}).Info(message)
	return true
}
