package kafka

import (
	"encoding/json"
	"testing"
	utils "xqledger/rdboperator/utils"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	. "github.com/smartystreets/goconvey/convey"
)

const repo = "GitOperatorTestRepo"
const id = "123456789123456789123456"
const email = "testorchestrator@gmail.com"
const recordTime = int64(1636570869)

func getEvent() []byte{
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
	out, _ := json.Marshal(record)
	return out
}

func getUpdateEvent() []byte{
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
	out, _ := json.Marshal(record)
	return out
}

func getDeleteEvent() []byte{
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
	out, _ := json.Marshal(record)
	return out
}


//convertMessageToProcessable(msg kafka.Message) (utils.RecordEvent, error) {
func TestConvertMessageToProcessable(t *testing.T) {


	Convey("Check convert event new record", t, func() {
		msg := kafka.Message{
			Key:   []byte(uuid.New().String()),
			Value: []byte(getEvent()),
		}
		event, err := convertMessageToProcessable(msg)
		So(err, ShouldBeNil)
		So(event.OperationType, ShouldEqual, "new")
	})

	Convey("Check convert event update record", t, func() {
		msg := kafka.Message{
			Key:   []byte(uuid.New().String()),
			Value: []byte(getUpdateEvent()),
		}
		event, err := convertMessageToProcessable(msg)
		So(err, ShouldBeNil)
		So(event.OperationType, ShouldEqual, "update")
	})

	Convey("Check convert event delete record", t, func() {
		msg := kafka.Message{
			Key:   []byte(uuid.New().String()),
			Value: []byte(getDeleteEvent()),
		}
		event, err := convertMessageToProcessable(msg)
		So(err, ShouldBeNil)
		So(event.OperationType, ShouldEqual, "delete")
	})

}