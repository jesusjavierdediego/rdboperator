package askgit

import (
	"os"
	"errors"
	"io"
	"io/ioutil"
	"bytes"
	"fmt"
	//"html/template"
	"regexp"
	"strings"
	"os/exec"
	"strconv"
	"encoding/json"
	"time"
	"math/rand"
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/src-d/go-git.v4"
	utils "xqledger/gitreader/utils"
	configuration "xqledger/gitreader/configuration"
)

var config = configuration.GlobalConfiguration
const queryTemplate = `"askgit \"$query\" --repo \"$repopath\" --format json > $exportfilename.json"`
const componentMessage = "Git Query Agent"

var trailingSpanRegex = regexp.MustCompile(`<span\s*[[:alpha:]="]*?[>]?$`)
var entityRegex = regexp.MustCompile(`&[#]*?[0-9[:alpha:]]*$`)
var (
	addedCodePrefix   = []byte(`<span class="added-code">`)
	removedCodePrefix = []byte(`<span class="removed-code">`)
	codeTagSuffix     = []byte(`</span>`)
)

func getRandomInt() int {
    rand.Seed(time.Now().UnixNano())
    return rand.Intn(100000)
}

/*
PUBLIC
Get the list of commits of a file with details as a list of commits with all fields
*/
func GetRecordHistory(file_path, reponame string) ([]Commit, error){
	commitList, err := getListOfCommitsForAFile(file_path, reponame)
	var results []Commit
	if err != nil {
		return results, err
	}
	var sqlQuery = ""
	for _, commitId := range commitList {
		sqlQuery = fmt.Sprintf(Query_commit_by_id, commitId)
		var exportfilename = fmt.Sprintf("%s_commit_by_id_%d", reponame, getRandomInt())
		commits, err := getCommits(sqlQuery, reponame, exportfilename)
		file := exportfilename + ".json"
		if err != nil{
			removeFile(file)
			return results, err
		}
		if !(len(commits)>0){
			removeFile(file)
			return results, errors.New(fmt.Sprintf("The commit with ID '%s' cannot be retrieved", commitId))
		}
		results = append(results, commits[0])
		removeFile(file)
	}
	return results, nil

}

func removeFile(exportfilename string){
	removeErr := os.Remove(exportfilename)
	if removeErr != nil {
		utils.PrintLogWarn(removeErr, componentMessage, "Removing File", "File could not be deleted: " + exportfilename)
	}
}

/*
Get the list of commit IDs of a file as a list of commit ids
*/
func getListOfCommitsForAFile(file_path, reponame string) ([]string, error){
	var emptyResult []string
	sqlQuery := fmt.Sprintf(Query_commitlist_by_file, file_path)
	var exportfilename = fmt.Sprintf("%s_commitlist_by_file_%d", reponame, getRandomInt())
	listOfCommits, err := getListOfCommits(sqlQuery, reponame, exportfilename)
	file := exportfilename + ".json"
	if err != nil {
		removeFile(file)
		return emptyResult, err
	}
	if !(len(listOfCommits)>0){
		removeFile(file)
		return emptyResult, errors.New("No commit was found")
	}
	removeFile(file)
	return listOfCommits, nil
}


/*
PUBLIC
Get the diff between two commits
Show the diff to html
Export to html
*/
func GetDiffTwoCommitsInFile(commit_id_1, commit_id_2, file_path, reponame string) (string, error){
	var result string
	var methodMsg = "GetDiffTwoCommits"
	var prettyJSON1 bytes.Buffer
	var prettyJSON2 bytes.Buffer

	olderRecord, err1 := GetContentInCommit(commit_id_1, file_path, reponame)
	if err1 != nil {
        utils.PrintLogError(err1, componentMessage, methodMsg, fmt.Sprintf("Error getting content of commit %s", commit_id_1))
        return result, err1
    }
	newerRecord, err2 := GetContentInCommit(commit_id_2, file_path, reponame)
	if err2 != nil {
        utils.PrintLogError(err2, componentMessage, methodMsg, fmt.Sprintf("Error getting content of commit %s", commit_id_2))
        return result, err2
    }

    error1 := json.Indent(&prettyJSON1, []byte(olderRecord), "", "\t")
    if error1 != nil {
        utils.PrintLogError(error1, componentMessage, methodMsg, "Error in JSON indentation of older record")
        return result, error1
    }
	error2 := json.Indent(&prettyJSON2, []byte(newerRecord), "", "\t")
	if error2 != nil {
		utils.PrintLogError(error2, componentMessage, methodMsg, "Error in JSON indentation of newer record")
        return result, error2
    }

	pt1 := string(prettyJSON1.Bytes())
	pt2 := string(prettyJSON2.Bytes())

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(pt1, pt2, false)
	fmt.Println(dmp.DiffPrettyText(diffs) + "\n")

	serializedTemplate := diffToHTML(diffs)
	utils.PrintLogInfo(componentMessage, methodMsg, "Successfully composed diff in HTML format")
	//fmt.Println(string(serializedTemplate))
	return string(serializedTemplate), nil
}


func GetContentInCommit(commit_id, file_path, reponame string) (string, error){
	var emptyResult string
	var exportfilename = fmt.Sprintf("%s_contentincommit_by_file_%d", reponame, getRandomInt())
	sqlQuery := fmt.Sprintf(Query_contents_from_commit_in_file, commit_id, file_path)
	content, err := runQueryContentInCommit(sqlQuery, reponame, exportfilename)
	file := exportfilename + ".json"
	if err != nil {
		removeFile(file)
		return emptyResult, err
	}
	if !(len(content) > 0) {
		removeFile(file)
		return emptyResult, errors.New("No results where found")
	}
	removeFile(file)
	return content, nil
}

func runGitQuery(sqlquery, repopath, exportfilename string) error{
	methodMsg := "runGitQuery"
	unquoteQuery := func(s string) string{
		t, err := strconv.Unquote(s)
		if err != nil {
			fmt.Printf("Unquote(%#v): %v\n", s, err)
		} else {
			fmt.Printf("Unquote(%#v) = %v\n", s, t)
		}
		return t
	}	
	askGitCommand := getQueryText(sqlquery, repopath, exportfilename)
	unquotedQuery := unquoteQuery(askGitCommand)
	utils.PrintLogInfo(componentMessage, methodMsg, "We're gona start running the ask git command:  " + unquotedQuery)
	out, err := exec.Command("/bin/sh", "-c", unquotedQuery).Output()
    if err != nil {
		utils.PrintLogError(err, componentMessage, methodMsg, "Error executing ask git command!!")
		return err
    }
    output := string(out[:])
	utils.PrintLogInfo(componentMessage, methodMsg, "Command Successfully Executed! - Output: " + output)
	return nil
}

func getQueryText(query, repopath, exportfilename string) string{
	var result = ""
	result = strings.ReplaceAll(queryTemplate, "$query", query)
	result = strings.ReplaceAll(result, "$repopath", repopath)
	result = strings.ReplaceAll(result, "$exportfilename", exportfilename)
	//fmt.Println("QUERY: " + result)
	return result
}

func getLocalRepoPath(reponame string) (string, error) {
	var repoPath = ""
	repoPath = os.Getenv("LOCALGITBASICPATH")
	if !(len(repoPath)>0) {
		repoPath = config.Gitserver.Localreposlocation + reponame
	}
	if !(len(repoPath) > 0) || !(len(reponame) > 0){
		return "", errors.New(fmt.Sprintf("The path for the local git repo cannot be composed - event.Unit: %s - Root path in config: %s" + reponame, repoPath))
	}
	return repoPath, nil
}

func synchronizeGitRepo(reponame string) (string, error) {
	methodMsg := "synchronizeGitRepo"
	repoPath, err := getLocalRepoPath(reponame)
	if err != nil {
		utils.PrintLogError(err, componentMessage, methodMsg, "Error getting path for local clones git repository: "+repoPath)
		return "", err
	}
	var r *git.Repository
	var openErr error
	r, openErr = git.PlainOpen(repoPath)
	remoteRepoURL := config.Gitserver.Url + "/" + config.Gitserver.Username + "/" + reponame
	if openErr != nil {
		utils.PrintLogInfo(componentMessage, methodMsg, "We cannot open the local Git repository: "+repoPath)
		/*
		Error opening the local repo -> Try to clone the remote repo
		*/
		utils.PrintLogInfo(componentMessage, methodMsg, "remoteRepoURL: " + remoteRepoURL)
		utils.PrintLogInfo(componentMessage, methodMsg, "We are going to clone the remote repo if it exists - URL: " + remoteRepoURL)
		cloneErr := Clone(remoteRepoURL, repoPath)
		if cloneErr != nil {
			return "", cloneErr
		}
		r, openErr = git.PlainOpen(repoPath)
		if openErr != nil {
			return "", openErr
		}
	} else {
		utils.PrintLogInfo(componentMessage, methodMsg, fmt.Sprintf("Local repo '%s' exist. We are gonna pull the last commit", reponame))
		_, err := r.Worktree()
		if err != nil {
			utils.PrintLogError(err, componentMessage, methodMsg, "Error getting Worktree in local Git repository: "+repoPath)
			return "", err
		}
		utils.PrintLogInfo(componentMessage, methodMsg, "git pull origin")
		//w.Pull(&git.PullOptions{RemoteName: "origin"})
	}
	return repoPath, nil
}

type CommitId struct {
	Commit_id string `json:"commit_id"`
}

type CommitContent struct {
	Contents string `json:"contents"`
}

func commitIdListToStringArray(commits []CommitId) []string{
	var result []string
	for _, commit := range commits {
		result = append(result, commit.Commit_id)
	}
	return result
}

func runQueryContentInCommit(sqlquery, reponame, exportfilename string) (string, error){
	var result string
	repoPath, syncErr := synchronizeGitRepo(reponame)
	if syncErr != nil {
		return result, syncErr
	}
	runCommandErr := runGitQuery(sqlquery, repoPath, exportfilename)
	if runCommandErr != nil {
		return result, runCommandErr
	}
	file, err := ioutil.ReadFile(exportfilename + ".json")
	if err != nil {
		return result, err
	}
	decodedContent := json.NewDecoder(strings.NewReader(string(file)))
	var data []CommitContent
	for {
		var payload CommitContent
		decodeErr := decodedContent.Decode(&payload)
		if decodeErr == io.EOF {
			// all done
			break
		} 
		if decodeErr != nil {
			return result, decodeErr
		}
		data = append(data, payload)
	}
	content := data[0].Contents
	return content, nil
}


func getListOfCommits(sqlquery, reponame, exportfilename string) ([]string, error){
	var result []string
	repoPath, syncErr := synchronizeGitRepo(reponame)
	if syncErr != nil {
		return result, syncErr
	}
	runCommandErr := runGitQuery(sqlquery, repoPath, exportfilename)
	if runCommandErr != nil {
		return result, runCommandErr
	}
	file, err := ioutil.ReadFile(exportfilename + ".json")
	if err != nil {
		return result, err
	}
	decodedContent := json.NewDecoder(strings.NewReader(string(file)))
	var data []CommitId
	for {
		var payload CommitId
		decodeErr := decodedContent.Decode(&payload)
		if decodeErr == io.EOF {
			// all done
			break
		} 
		if decodeErr != nil {
			return result, decodeErr
		}
		data = append(data, payload)
	}
	listOfCommits := commitIdListToStringArray(data)
	return listOfCommits, nil
}

func shouldWriteInline(diff diffmatchpatch.Diff) bool {
	if true &&
		diff.Type == diffmatchpatch.DiffEqual ||
		diff.Type == diffmatchpatch.DiffInsert ||
		diff.Type == diffmatchpatch.DiffDelete {
		return true
	}
	return false
}

func diffToHTML(diffs []diffmatchpatch.Diff) []byte {
	buf := bytes.NewBuffer(nil)
	match := ""

	for _, diff := range diffs {
		if shouldWriteInline(diff) {
			if len(match) > 0 {
				diff.Text = match + diff.Text
				match = ""
			}

			m := trailingSpanRegex.FindStringSubmatchIndex(diff.Text)
			if m != nil {
				match = diff.Text[m[0]:m[1]]
				diff.Text = strings.TrimSuffix(diff.Text, match)
			}
			m = entityRegex.FindStringSubmatchIndex(diff.Text)
			if m != nil {
				match = diff.Text[m[0]:m[1]]
				diff.Text = strings.TrimSuffix(diff.Text, match)
			}
			// Print an existing closing span first before opening added/remove-code span so it doesn't unintentionally close it
			if strings.HasPrefix(diff.Text, "</span>") {
				buf.WriteString("</span>")
				diff.Text = strings.TrimPrefix(diff.Text, "</span>")
			}
			// If we weren't able to fix it then this should avoid broken HTML by not inserting more spans below
			// The previous/next diff section will contain the rest of the tag that is missing here
			if strings.Count(diff.Text, "<") != strings.Count(diff.Text, ">") {
				buf.WriteString(diff.Text)
				continue
			}
		}
		switch {
		case diff.Type == diffmatchpatch.DiffEqual:
			buf.WriteString(diff.Text)
		case diff.Type == diffmatchpatch.DiffInsert:
			buf.Write(addedCodePrefix)
			buf.WriteString(diff.Text)
			buf.Write(codeTagSuffix)
		case diff.Type == diffmatchpatch.DiffDelete:
			buf.Write(removedCodePrefix)
			buf.WriteString(diff.Text)
			buf.Write(codeTagSuffix)
		}
	}
	return buf.Bytes()
}


type Commit struct {
	Id   string `json:"id"`
	Message string `json:"message"`
	Summary string `json:"summary"`
	Author_name string `json:"author_name"`
	Author_email string `json:"author_email"`
	Author_when string `json:"author_when"`
	Committer_name string `json:"committer_name"`
	Committer_email string `json:"committer_email"`
	Committer_when string `json:"committer_when"`
	Parent_id string `json:"parent_id"`
	Parent_count int `json:"parent_count"`
}


func getCommits(sqlquery, reponame, exportfilename string) ([]Commit, error){
	var data []Commit
	repoPath, syncErr := synchronizeGitRepo(reponame)
	if syncErr != nil {
		return data, syncErr
	}
	runCommandErr := runGitQuery(sqlquery, repoPath, exportfilename)
	if runCommandErr != nil {
		return data, runCommandErr
	}
	file, err := ioutil.ReadFile(exportfilename + ".json")
	if err != nil {
		return data, err
	}
	decodedContent := json.NewDecoder(strings.NewReader(string(file)))
	for {
		var commit Commit
		decodeErr := decodedContent.Decode(&commit)
		if decodeErr == io.EOF {
			// all done
			break
		} 
		if decodeErr != nil {
			return data, decodeErr
		}
		data = append(data, commit)
	}
	return data, nil
}
