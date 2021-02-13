package grpc

import (
	"errors"
	"fmt"
	"strings"
	configuration "xqledger/gitreader/configuration"
	pki "xqledger/gitreader/pki"
	pb "xqledger/gitreader/protobuf"
	utils "xqledger/gitreader/utils"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const componentMessage = "GRPC Server"
const unauthenticatedMsg = "Authorization token for IDM is not valid"

var config = configuration.GlobalConfiguration
var restClient = utils.GetStandardRestClient()

type DigitalIdentityService struct {
	request *pb.IDR
}

func NewDigitalIdentityService(request *pb.IDR) *DigitalIdentityService {
	return &DigitalIdentityService{request: request}
}

/*
  ok: OK
  cancelled: CANCELLED
  invalid: INVALID_ARGUMENT
  notfound: NOT_FOUND
  permissiondenied: PERMISSION_DENIED
  unauthenticated: UNAUTHENTICATED
  notimplemented: UNIMPLEMENTED
  internal: INTERNAL
  notavailable: UNAVAILABLE
*/
func getErrorResponseMessage(outputErr error) error {
	var finalErr error
	errText := outputErr.Error()
	i := strings.Index(errText, "-Code:")
	if strings.Contains(errText, config.Grpcresponse.Invalid) || strings.Contains(errText, "missing") {
		finalErr = status.Error(codes.InvalidArgument, string(errText[i:]))
	} else if strings.Contains(errText, config.Grpcresponse.Notfound) {
		finalErr = status.Error(codes.NotFound, string(errText[i:]))
	} else if strings.Contains(errText, config.Grpcresponse.Permissiondenied) {
		finalErr = status.Error(codes.PermissionDenied, string(errText[i:]))
	} else if strings.Contains(errText, config.Grpcresponse.Unauthenticated) {
		finalErr = status.Error(codes.Unauthenticated, string(errText[i:]))
	} else if strings.Contains(errText, config.Grpcresponse.Notimplemented) {
		finalErr = status.Error(codes.Unimplemented, string(errText[i:]))
	} else if strings.Contains(errText, config.Grpcresponse.Internal) {
		finalErr = status.Error(codes.Internal, string(errText[i:]))
	} else {
		finalErr = status.Error(codes.Unavailable, string(errText[i:]))
	}
	return finalErr
}

func validateIDR(idr *pb.IDR) error {
	correlationID, _ := utils.GetCorrelationID(idr.String())
	var msg = ""
	if !(len(idr.UserId.CompleteName) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - CompleteName is missing", correlationID)
	}
	if !(len(idr.UserId.EntityId) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - EntityId is missing", correlationID)
	}
	if !(len(idr.UserId.Country) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - Country is missing", correlationID)
	}
	if !(len(idr.UserId.Postcode) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - Postcode is missing", correlationID)
	}
	if !(len(idr.UserId.Address) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - Address is missing", correlationID)
	}
	if !(len(idr.UserId.City) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - City is missing", correlationID)
	}
	if !(len(idr.UserId.Province) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - Province is missing", correlationID)
	}
	if !(len(idr.UserId.AadUserName) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - AadUserName is missing", correlationID)
	}
	if !(len(idr.Purpose) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - Purpose is missing", correlationID)
	}
	if !(len(idr.Context) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - Context is missing", correlationID)
	}
	if !(len(idr.BusinessId) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - BusinessId is missing", correlationID)
	}
	if !(len(idr.Authtoken) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating IDR - Authtoken is missing", correlationID)
	}
	if len(msg) > 0 {
		return errors.New(msg)
	}
	return nil
}

func (c *DigitalIdentityService) RegisterNewDID(ctx context.Context, idRequest *pb.IDR) (*pb.Empty, error) {
	var empty pb.Empty
	var finalErr error
	methodMsg := "RegisterNewDID"
	validErr := validateIDR(idRequest)
	if validErr != nil {
		finalErr = getErrorResponseMessage(validErr)
		return &empty, finalErr
	}
	if utils.Contains(config.Pki.Allowedauthtokensbasic, idRequest.Authtoken) || utils.Contains(config.Pki.Allowedauthtokensadvanced, idRequest.Authtoken) {
		outputErr := pki.GenerateUserCompanyPack(idRequest, restClient)
		if outputErr != nil {
			finalErr = getErrorResponseMessage(outputErr)
			return &empty, finalErr
		} else {
			return &empty, nil
		}
	} else {
		utils.PrintLogInfo(componentMessage, methodMsg, unauthenticatedMsg)
		finalErr = status.Error(codes.Unauthenticated, unauthenticatedMsg)
		return &empty, finalErr
	}
}

func (c *DigitalIdentityService) GetDID(ctx context.Context, idRequest *pb.IDR) (*pb.IDRes, error) {
	var empty pb.IDRes
	var finalErr error
	methodMsg := "GetDID"
	validErr := validateIDR(idRequest)
	if validErr != nil {
		finalErr = getErrorResponseMessage(validErr)
		return &empty, finalErr
	}
	if utils.Contains(config.Pki.Allowedauthtokensadvanced, idRequest.Authtoken) {
		output, outputErr := pki.GetUserCompanyPack(idRequest, restClient)
		if outputErr != nil {
			finalErr = getErrorResponseMessage(outputErr)
			return &empty, finalErr
		} else {
			return output, nil
		}
	} else {
		utils.PrintLogInfo(componentMessage, methodMsg, unauthenticatedMsg)
		finalErr = status.Error(codes.Unauthenticated, unauthenticatedMsg)
		return &empty, finalErr
	}
}

func (c *DigitalIdentityService) PartialRevokeDID(ctx context.Context, idRequest *pb.IDR) (*pb.Empty, error) {
	var empty pb.Empty
	var finalErr error
	methodMsg := "PartialRevokeDID"
	validErr := validateIDR(idRequest)
	if validErr != nil {
		finalErr = getErrorResponseMessage(validErr)
		return &empty, finalErr
	}
	if utils.Contains(config.Pki.Allowedauthtokensbasic, idRequest.Authtoken) || utils.Contains(config.Pki.Allowedauthtokensadvanced, idRequest.Authtoken) {
		outputErr := pki.RevokeUserCompanyPack(idRequest)
		if outputErr != nil {
			finalErr = getErrorResponseMessage(outputErr)
			return &empty, finalErr
		} else {
			return &empty, nil
		}
	} else {
		utils.PrintLogInfo(componentMessage, methodMsg, unauthenticatedMsg)
		finalErr = status.Error(codes.Unauthenticated, unauthenticatedMsg)
		return &empty, finalErr
	}
}

func (c *DigitalIdentityService) CompleteRevokeDID(ctx context.Context, idRequest *pb.IDR) (*pb.Empty, error) {
	var empty pb.Empty
	var finalErr error
	methodMsg := "CompleteRevokeDID"
	validErr := validateIDR(idRequest)
	if validErr != nil {
		finalErr = getErrorResponseMessage(validErr)
		return &empty, finalErr
	}
	if utils.Contains(config.Pki.Allowedauthtokensadvanced, idRequest.Authtoken) {
		outputErr := pki.RevokeAllUserCompanyPacks(idRequest)
		if outputErr != nil {
			finalErr = getErrorResponseMessage(outputErr)
			return &empty, finalErr
		} else {
			return &empty, nil
		}
	} else {
		utils.PrintLogInfo(componentMessage, methodMsg, unauthenticatedMsg)
		finalErr = status.Error(codes.Unauthenticated, unauthenticatedMsg)
		return &empty, finalErr
	}
}
