package grpcmock

import (
	pb "xqledger/gitreader/protobuf"

	gomock "github.com/golang/mock/gomock"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

type MockDigitalIdentityServiceClient struct {
	ctrl     *gomock.Controller
	recorder *_MockDigitalIdentityServiceClientRecorder
}

type _MockDigitalIdentityServiceClientRecorder struct {
	mock *MockDigitalIdentityServiceClient
}

func NewMockDigitalIdentityServiceClient(ctrl *gomock.Controller) *MockDigitalIdentityServiceClient {
	mock := &MockDigitalIdentityServiceClient{ctrl: ctrl}
	mock.recorder = &_MockDigitalIdentityServiceClientRecorder{mock}
	return mock
}

func (_m *MockDigitalIdentityServiceClient) EXPECT() *_MockDigitalIdentityServiceClientRecorder {
	return _m.recorder
}

func (_m *MockDigitalIdentityServiceClient) GetDID(_param0 context.Context, _param1 *pb.IDR, _param2 ...grpc.CallOption) (*pb.IDRes, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "GetDID", _s...)
	ret0, _ := ret[0].(*pb.IDRes)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockDigitalIdentityServiceClientRecorder) GetDID(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetDID", _s...)
}

func (_m *MockDigitalIdentityServiceClient) RegisterNewDID(_param0 context.Context, _param1 *pb.IDR, _param2 ...grpc.CallOption) (*pb.Empty, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "RegisterNewDID", _s...)
	ret0, _ := ret[0].(*pb.Empty)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockDigitalIdentityServiceClientRecorder) RegisterNewDID(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "RegisterNewDID", _s...)
}

func (_m *MockDigitalIdentityServiceClient) PartialRevokeDID(_param0 context.Context, _param1 *pb.IDR, _param2 ...grpc.CallOption) (*pb.Empty, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "PartialRevokeDID", _s...)
	ret0, _ := ret[0].(*pb.Empty)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockDigitalIdentityServiceClientRecorder) PartialRevokeDID(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "PartialRevokeDID", _s...)
}

func (_m *MockDigitalIdentityServiceClient) CompleteRevokeDID(_param0 context.Context, _param1 *pb.IDR, _param2 ...grpc.CallOption) (*pb.Empty, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "CompleteRevokeDID", _s...)
	ret0, _ := ret[0].(*pb.Empty)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockDigitalIdentityServiceClientRecorder) CompleteRevokeDID(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "CompleteRevokeDID", _s...)
}
