package grpc

import (
	"errors"
	"fmt"
	"strings"
	configuration "xqledger/gitreader/configuration"
	pb "xqledger/gitreader/protobuf"
	utils "xqledger/gitreader/utils"
	askgit "xqledger/gitreader/askgit"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const componentMessage = "GRPC Server"
var config = configuration.GlobalConfiguration

type RecordHistoryService struct {
	query *pb.Query
}

func NewRecordHistoryService(query *pb.Query) *RecordHistoryService {
	return &RecordHistoryService{query: query}
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

func validateRecordHistoryQuery(query *pb.Query) error {
	correlationID, _ := utils.GetCorrelationID(query.FilePath + query.RepoName)
	var msg = ""
	if !(len(query.FilePath) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating RecordHistoryQuery - FilePath is missing", correlationID)
	}
	if !(len(query.RepoName) > 0) {
		msg = fmt.Sprintf("-Code: %s - Error validating RecordHistoryQuery - RepoName is missing", correlationID)
	}
	if len(msg) > 0 {
		return errors.New(msg)
	}
	return nil
}

func commitTpPBCommit(commit askgit.Commit) *pb.Commit {
	var c pb.Commit
	c.Id= commit.Id
	c.Message = commit.Message
	c.AuthorEmail = commit.Author_email
	c.AuthorName = commit.Author_name
	c.AuthorWhen = commit.Author_when
	c.CommitterEmail = commit.Committer_email
	c.CommitterName = commit.Committer_name
	c.CommitterWhen = commit.Committer_when
	c.ParentCount = int32(commit.Parent_count)
	c.ParentId = commit.Parent_id
	c.Summary = commit.Summary
	return &c
}

func (s *RecordHistoryService) GetRecordHistory(ctx context.Context, query *pb.Query) (*pb.RecordHistory, error) {
	var result pb.RecordHistory
	var finalErr error
	methodMsg := "GetRecordHistory"
	validErr := validateRecordHistoryQuery(query)
	if validErr != nil {
		finalErr = getErrorResponseMessage(validErr)
		return &result, finalErr
	}
	commits, err := askgit.GetRecordHistory(query.FilePath, query.RepoName)
	if err != nil {
		finalErr = getErrorResponseMessage(err)
		utils.PrintLogError(err, componentMessage, methodMsg, finalErr.Error())
		return &result, finalErr
	} 
	var list []*pb.Commit
	for _, commit := range commits {
		pbc := commitTpPBCommit(commit)
		list = append(list, pbc)
	}
	result.Commits = list

	return &result, nil
}

func (s *RecordHistoryService) GetContentInCommit(ctx context.Context, query *pb.Query) (*pb.CommitContent, error) {
	var result pb.CommitContent
	var finalErr error
	methodMsg := "GetContentInCommit"
	validErr := validateRecordHistoryQuery(query)
	if validErr != nil {
		finalErr = getErrorResponseMessage(validErr)
		return &result, finalErr
	}
	content, err := askgit.GetContentInCommit(query.CommitIdOld, query.FilePath, query.RepoName)
	if err != nil {
		finalErr = getErrorResponseMessage(err)
		utils.PrintLogError(err, componentMessage, methodMsg, finalErr.Error())
		return &result, finalErr
	} 
	result.Content = content
	return &result, nil
}

func (s *RecordHistoryService) GetDiffTwoCommitsInFile(ctx context.Context, query *pb.Query) (*pb.DiffHtml, error) {
	var result pb.DiffHtml
	var finalErr error
	methodMsg := "GetDiffTwoCommitsInFile"
	validErr := validateRecordHistoryQuery(query)
	if validErr != nil {
		finalErr = getErrorResponseMessage(validErr)
		return &result, finalErr
	}
	htmlDiff, err := askgit.GetDiffTwoCommitsInFile(query.CommitIdOld, query.CommitIdNew, query.FilePath, query.RepoName)
	if err != nil {
		finalErr = getErrorResponseMessage(err)
		utils.PrintLogError(err, componentMessage, methodMsg, finalErr.Error())
		return &result, finalErr
	} 
	result.Html = htmlDiff
	return &result, nil
}