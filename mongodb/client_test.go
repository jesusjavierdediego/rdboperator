package mongodb

import (
	"strings"
	"testing"
	utils "xqledger/rdboperator/utils"

	. "github.com/smartystreets/goconvey/convey"
)

const repo = "GitOperatorTestRepo"
const id = "123456789123456789123456"
const email = "testorchestrator@gmail.com"
const recordTime = int64(1636570869)

func getEvent()utils.RecordEvent{
	record := utils.RecordEvent{}
	record.Id = id
	record.Group = ""
	record.DBName = repo
	record.User = email
	record.OperationType = "new"
	record.SendingTime = recordTime
	record.ReceptionTime = recordTime
	record.ProcessingTime = recordTime
	record.Priority = "MEDIUM"
	record.RecordContent = "{\"browsers\":{\"firefox\":{\"name\":\"Firefox\",\"pref_url\":\"about:config\",\"releases\":{\"1\":{\"release_date\":\"2004-11-09\",\"status\":\"retired\",\"engine\":\"Gecko\",\"engine_version\":\"1.7\"}}}}}"
	record.Status = "PENDING"
	return record
}

func getUpdateEvent()utils.RecordEvent{
	record := utils.RecordEvent{}
	record.Id = id
	record.Group = ""
	record.DBName = repo
	record.User = email
	record.OperationType = "update"
	record.SendingTime = recordTime
	record.ReceptionTime = recordTime
	record.ProcessingTime = recordTime
	record.Priority = "MEDIUM"
	record.RecordContent = "{\"browsers\":{\"firefox\":{\"name\":\"Firefox\",\"pref_url\":\"about:config\",\"releases\":{\"1\":{\"release_date\":\"2004-12-23\",\"status\":\"retired\",\"engine\":\"Gecko\",\"engine_version\":\"1.8\"}}}}}"
	record.Status = "PENDING"
	return record
}

func getDeleteEvent()utils.RecordEvent{
	record := utils.RecordEvent{}
	record.Id = id
	record.Group = ""
	record.DBName = repo
	record.User = email
	record.OperationType = "delete"
	record.SendingTime = recordTime
	record.ReceptionTime = recordTime
	record.ProcessingTime = recordTime
	record.Priority = "MEDIUM"
	record.Status = "PENDING"
	return record
}



func TestHandleNewEvent(t *testing.T) {

	Convey("Check event new record", t, func() {
		var err error
		err = HandleEvent(getEvent())
		if err != nil && strings.Contains(err.Error(), "duplicate key") {
			err = nil
		}
		So(err, ShouldBeNil)
	})

}

func TestHandleUpdateEvent(t *testing.T) {

	Convey("Check event update record", t, func() {
		err := HandleEvent(getUpdateEvent())
		So(err, ShouldBeNil)
	})

}

func TestHandleDeleteEvent(t *testing.T) {

	Convey("Check event delete record", t, func() {
		err := HandleEvent(getDeleteEvent())
		So(err, ShouldBeNil)
	})

}