package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	. "github.com/smartystreets/goconvey/convey"
	app "fts/sts-gateway/app"
)

func getCardValidationRequest() app.CardValidationRequest {
	var result app.CardValidationRequest
	result.CardNumber = "4548034981104011"
	result.Cvv = "444"
	result.Month = "11"
	result.Priority = "HIGH"
	result.Year = "19"
	return result
}

func TestHttpServices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("profile", "testprofile")

	Convey("Should check the keep alive service works OK", t, func() {
		KeepAlive(c)
		So(c.Writer.Status(), ShouldEqual, 200)
	})
}

/*
resp := httptest.NewRecorder()
gin.SetMode(gin.TestMode)
c, r := gin.CreateTestContext(resp)
c.Set("profile", "myfakeprofile")
r.GET("/test", func(c *gin.Context) {
    _, found := c.Get("profile")
    // found is always false
    t.Log(found)
    c.Status(200)
})
c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
r.ServeHTTP(resp, c.Request)

 func TestValidationManager(t *testing.T) {
	Convey("Should validate a valid card validation request", t, func() {
		cvr := getCardValidationRequest()
		result, err := validationManager(cvr)
		So(err, ShouldBeNil)
		So(result.CorrelationID, ShouldEqual, "001")
	})
} */
