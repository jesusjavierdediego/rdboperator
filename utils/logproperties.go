package utils


const Event_topic_received_ok = "EVENT TOPIC RECEIVED OK"
const Event_topic_received_fail = "EVENT TOPIC RECEIVED FAIL"
const Event_topic_received_unacceptable = "EVENT TOPIC RECEIVED UNACCEPTABLE"

const Error_unmarshalling_RDB = "RDB UNMARSHAL ERROR"
const Error_inserting_record_in_RDB = "RDB INSERTION RECORD ERROR"
const Error_updating_record_in_RDB = "RDB UPDATE RECORD ERROR"
const Error_deletion_record_in_RDB = "RDB DELETE RECORD ERROR"

const Successful_insertion = "RECORD INSERTED OK - ID '%s' - Database '%s' - Collection '%s'"
const Successful_update = "RECORD UPDATED OK - ID '%s' - Database '%s' - Collection '%s'"
const Successful_delete = "RECORD DELETED OK - with ID '%s' - Database '%s' - Collection '%s'"