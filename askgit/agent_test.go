package askgit

import (
	"fmt"
	"testing"
	"html/template"
	. "github.com/smartystreets/goconvey/convey"
)

/*

| 2077eb80abf9522a163c7ce012c8b90dd87d5a8c |
+------------------------------------------+
| 30bf07336744d15b88cec5197fb0bd05991a6dfd |
+------------------------------------------+
| 61954365b9403ab6be1c5b3a0b2609e5ed5cb938 |

*/
const commitId_newer_Test = "2077eb80abf9522a163c7ce012c8b90dd87d5a8c"
const commitId_older_Test = "30bf07336744d15b88cec5197fb0bd05991a6dfd"
const commitId2Test = "30bf07336744d15b88cec5197fb0bd05991a6dfd"
const gitrepoTest = "gitrepo"
const fileNameTest = "2.json"
// askgit "SELECT contents from files where commit_id = '2077eb80abf9522a163c7ce012c8b90dd87d5a8c' AND name = '2.json'"

func TestSyntax(t *testing.T) {
	Convey("Check the syntax ", t, func() {// ([]string, error)
		So(5, ShouldBeGreaterThan, 0)
	})
}


func TestGetHistoryOfAFile(t *testing.T) {
	Convey("Get the history of a file with details", t, func() {
		list, err := GetRecordHistory(fileNameTest, gitrepoTest)
		So(err, ShouldBeNil)
		fmt.Println(len(list))
		So(len(list), ShouldBeGreaterThan, 0)
	})
}

// func TestGetListOfCommitsOfAFile(t *testing.T) {
// 	Convey("Get the list of commits in the history of a file with details", t, func() {// ([]string, error)
// 		list, err := GetListOfCommitsForAFile(fileNameTest, gitrepoTest)
// 		So(err, ShouldBeNil)
// 		So(len(list), ShouldBeGreaterThan, 0)
// 	})
// }

func TestGetContentInCommitAndFile(t *testing.T) {
	Convey("Get the content of a given commit for a file ", t, func() {
		content, err := GetContentInCommit(commitId_older_Test, fileNameTest, gitrepoTest)
		So(err, ShouldBeNil)
		So(len(content), ShouldBeGreaterThan, 0)
	})
}

func TestTheDiffBetweenTwoCommitsOfAFile(t *testing.T) {
	Convey("Get the diff between two commits for a file ", t, func() {
		serializedTemplate, err := GetDiffTwoCommitsInFile(commitId_older_Test, commitId_newer_Test, fileNameTest, gitrepoTest)
		html := template.HTML(serializedTemplate)
		fmt.Println(html)
		So(err, ShouldBeNil)
		So(len(html), ShouldBeGreaterThan, 0)
	})
}

