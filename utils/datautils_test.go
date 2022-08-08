package utils


import (	
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

const value1 = "value1"
var array = []string {value1, "value2"}

func TestContains(t *testing.T) {
	Convey("Check if an array contains a string ", t, func() {
		result := Contains(array, value1)
		So(result, ShouldBeTrue)
	})
	Convey("Check if an array does not contain a string ", t, func() {
		result := Contains(array, "value3")
		So(result, ShouldBeFalse)
	})
}

func TestGetRDBID(t *testing.T) {
	Convey("Check GetRDBID ", t, func() {
		result, err := GetRDBID()
		So(err, ShouldBeNil)
		So(len(result), ShouldEqual, 30)//24?
	})
}

func TestGetCorrelationID(t *testing.T) {
	Convey("Check GetCorrelationID ", t, func() {
		result, err := GetCorrelationID("1234")
		So(err, ShouldBeNil)
		So(len(result), ShouldEqual, 40)
	})
}

func TestGetFormattedNow(t *testing.T) {
	Convey("Check GetFormattedNow ", t, func() {
		result := GetFormattedNow()
		So(len(result), ShouldEqual, 19)
	})
}

func TestGetEpochNow(t *testing.T) {
	Convey("Check GetEpochNow ", t, func() {
		result := GetEpochNow()
		So(result, ShouldBeGreaterThan, 0)
	})
}

func TestGetRandomSerial(t *testing.T) {
	Convey("Check GetRandomSerial ", t, func() {
		result := GetRandomSerial()
		So(len(result.String()), ShouldBeGreaterThan, 25)
	})
}

func TestTurnUnixTimestampToString(t *testing.T) {
	Convey("Check TurnUnixTimestampToString ", t, func() {
		result := TurnUnixTimestampToString(18)
		So(len(result), ShouldEqual, 29)
	})
}

func TestAddTimeToNowEpoch(t *testing.T) {
	Convey("Check AddTimeToNowEpoch ", t, func() {
		result := AddTimeToNowEpoch(1, 5, 21)
		So(result, ShouldBeGreaterThan, 0)
	})
}