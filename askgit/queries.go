package askgit


// Get list of changes in a commit
//const Query_commitinfo_by_commitid = "SELECT * FROM commit where id='%s'"
// Get history of a file as a list of blames
//const Query_blamelist_by_file = "SELECT * FROM blame WHERE file_path ='%s'"

// Get history of a file as a list of commits
const Query_commitlist_by_file = "SELECT DISTINCT commit_id FROM blame WHERE path = '%s'"
// Get the changes in every commit in a given file
const Query_contents_from_commit_in_file = "SELECT contents FROM files WHERE commit_id = '%s' AND name = '%s'"
// Get the details of a given commit ID
const Query_commit_by_id = "SELECT * FROM commits WHERE id = '%s'"


//SELECT CONTENTS FROM files WHERE commit_id=‘<older commit>’ AND PATH = ‘FILENAMEPATH