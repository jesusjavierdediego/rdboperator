package grpc

import (
	"testing"
	"time"
	mock "xqledger/gitreader/grpc/grpcmock"
	pb "xqledger/gitreader/protobuf"
	utils "xqledger/gitreader/utils"

	gomock "github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
)

var req = utils.GetIDR4Test()
var res = utils.GetIDRes4Test()
var errorRes pb.Empty

func TestGRPCDigitalIdentityService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := mock.NewMockDigitalIdentityServiceClient(ctrl)

	mockClient.EXPECT().RegisterNewDID(
		gomock.Any(),
		req,
	).Return(&errorRes, nil)

	mockClient.EXPECT().GetDID(
		gomock.Any(),
		req,
	).Return(res, nil)

	mockClient.EXPECT().PartialRevokeDID(
		gomock.Any(),
		req,
	).Return(&errorRes, nil)

	mockClient.EXPECT().CompleteRevokeDID(
		gomock.Any(),
		req,
	).Return(&errorRes, nil)

	testRegisterNewDID(t, mockClient)
	testGetDID(t, mockClient)
	testPartialRevokeDID(t, mockClient)
	testCompleteRevokeDID(t, mockClient)
}

func testRegisterNewDID(t *testing.T, client pb.DigitalIdentityServiceClient) {
	Convey("Should register a new DID through gRPC", t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := client.RegisterNewDID(ctx, req)
		So(err, ShouldBeNil)
	})
}

func testGetDID(t *testing.T, client pb.DigitalIdentityServiceClient) {
	Convey("Should retrieve a DID through gRPC", t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r0, r1 := client.GetDID(ctx, req)
		if r0.UserId.Piiid != "973e0e91-1120-4acd-8135-aad21be70d26" {
			t.Errorf("mocking failed")
		}
		So(r1, ShouldBeNil)
		So(r0.UserId.Piiid, ShouldEqual, req.UserId.Piiid)
	})
}

func testPartialRevokeDID(t *testing.T, client pb.DigitalIdentityServiceClient) {
	Convey("Should revoke an DID through gRPC", t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := client.PartialRevokeDID(ctx, req)
		if err != nil {
			t.Errorf("mocking failed")
		}
		So(err, ShouldBeNil)
	})
}

func testCompleteRevokeDID(t *testing.T, client pb.DigitalIdentityServiceClient) {
	Convey("Should revoke an DID through gRPC", t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := client.CompleteRevokeDID(ctx, req)
		if err != nil {
			t.Errorf("mocking failed")
		}
		So(err, ShouldBeNil)
	})
}
