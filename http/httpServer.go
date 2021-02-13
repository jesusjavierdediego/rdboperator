package http

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"regexp"
	"path"
	"google.golang.org/grpc"
	"github.com/gin-gonic/gin"
	app "fts/sts-gateway/app"
	configuration "fts/sts-gateway/configuration"
	grpcClient "fts/sts-gateway/grpc"
	pb "fts/sts-gateway/protobuf"
	services "fts/sts-gateway/services"
	idpalapplinkStorage "fts/sts-gateway/storage_idpalapplinks"
	idpalcallbackStorage "fts/sts-gateway/storage_idpalcallbacks"
	customerStorage "fts/sts-gateway/storage_customers"
	flowstorage "fts/sts-gateway/storage_flows"
	bistorage "fts/sts-gateway/storage_business"
	utils "fts/sts-gateway/utils"
	idpal "fts/sts-gateway/idpal"
)

var config = configuration.GlobalConfiguration
const componentMessage = "HTTP Server"
const getCustomersMethodMessage = "Query for Customer record"
const internalServiceErrorMsg = "Service not available"
const externalServiceErrorMsg = "External Service not available"
const allowed = "ALLOWED"
const notAllowed = "NOT_ALLOWED"
const dateMRZFormatNotValid = "MRZ Date to parse is not valid"

func KeepAlive(c *gin.Context) {
	c.JSON(200, "ok")
}

func GetAADUserServiceClient() (pb.AADUserServiceClient, *grpc.ClientConn, error) {
	var result pb.AADUserServiceClient
	address := config.Aadconsumer.Host + ":" + strconv.Itoa(config.Aadconsumer.Port)
	conn, connErr := grpc.Dial(address, grpc.WithInsecure())
	if connErr != nil {
		utils.PrintLogError(connErr, "gRPC Client 4 AAD", "Sending new user", "Error in gRPC connection")
		return result, nil, connErr
	}

	result = pb.NewAADUserServiceClient(conn)
	return result, conn, nil
}

func GetDWUserServiceClient() (pb.WriteEventBatchServiceClient, *grpc.ClientConn, error) {
	var result pb.WriteEventBatchServiceClient
	address := config.Datawriter.Host + ":" + strconv.Itoa(config.Datawriter.Port)
	conn, connErr := grpc.Dial(address, grpc.WithInsecure())
	if connErr != nil {
		utils.PrintLogError(connErr, "gRPC Client 4 Customer Data Writing", "Sending new customer", "Error in gRPC connection")
		return result, nil, connErr
	}
	result = pb.NewWriteEventBatchServiceClient(conn)
	return result, conn, nil
}

/*
EXTERNAL CUSTOMER VALIDATION
*/
func verifyIDPalSignature(body string, signature string) bool {
    secret := []byte(config.Idpal.Pushsecret)
	payloadBytes := []byte(body)
	hash := hmac.New(sha1.New, secret)
	hash.Write(payloadBytes)
	generatedSignature := "sha1=" + hex.EncodeToString(hash.Sum(nil))
	if strings.Compare(signature, generatedSignature) == 0 {
		return true
	} else {
		return false
	}
}

func registerNewCustomerRecord(c *gin.Context, businessid string, eventBody app.IdpalCallbackEvent) {
	methodMsg := "registerNewCustomerRecord"
	bi, biErr := bistorage.GetBusinessInfoByID(businessid)
	if biErr != nil {
		utils.PrintLogError(biErr, componentMessage, methodMsg, "Error getting information about Business Info: " + businessid)
		c.JSON(403, "Business not supported")
	}
	// 1-Get the customer text info
	applinkRecord, applinkRecordErr := idpalapplinkStorage.GetApplink(eventBody.Uuid)
	if applinkRecordErr != nil {
		msg := "Applink record access failed"
		utils.PrintLogError(applinkRecordErr, componentMessage, methodMsg, msg)
		c.JSON(500, msg)
		return 
	} else {
		httpClient := idpal.GetRestClient()
		custInfo, custInfoErr := idpal.GetCustomerInformation(businessid, eventBody.Submission_id, httpClient)
		if custInfoErr != nil {
			msg := fmt.Sprintf("Error getting customer info from IDPAL - Business: %s - Submission: %s", businessid, strconv.Itoa(eventBody.Submission_id))
			utils.PrintLogError(custInfoErr, componentMessage, methodMsg, msg)
			c.JSON(404, msg)
			return 
		} else {
			utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL Customer Info retrieved - Email: "+applinkRecord.Basicuseremail)
			grpcDWClient, grpcDWConn, grpcClientErr := GetDWUserServiceClient()
			grpcAADClient, grpcAADConn, grpcAADclientErr := GetAADUserServiceClient()
			if grpcClientErr != nil || grpcAADclientErr != nil {
				msg := fmt.Sprintf("Error building clients needed the registration of user - Business: %s - Submission: %s", businessid, strconv.Itoa(eventBody.Submission_id))
				utils.PrintLogError(custInfoErr, componentMessage, methodMsg, msg)
				c.JSON(500, msg)
				return 
			}
			creq, _ := makeRegisterCustomerRequest(custInfo, applinkRecord)
			event, eventErr := makeEventWithCustomerRequest(creq)
			if eventErr != nil {
				utils.PrintLogError(eventErr, componentMessage, methodMsg, "Error marshaling RegisterCustomerRequest when building event")
			} else {
				utils.PrintLogInfo(componentMessage, methodMsg, "Event for IDPAL Customer request is made (DW + AAD)")
				registrationResponse := ProcessPostedCustomerRequest(event, &bi, grpcDWClient, grpcDWConn, grpcAADClient, grpcAADConn)
				if registrationResponse.Result.Error != "" {
					msg := "Error when processing submitted identified user - ProcessPostedCustomerRequest"
					registrationErr := errors.New(msg)
					utils.PrintLogError(registrationErr, componentMessage, methodMsg, msg)
					c.JSON(500, msg)
					return 
				} else {
					utils.PrintLogInfo(componentMessage, methodMsg, "Event for IDPAL Customer request has been registered OK")
					if bi.Business_requires_proofid {
						utils.PrintLogInfo(componentMessage, methodMsg, "Business Info requires proof of ID")
						// Get the submitted documents (business description defines what types are required. E. g. 'passport')
						for _, docType := range bi.Idpal_doctypes {
							utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL get documents")
							docFilePath, docFileErr := idpal.GetSubmissionDocument(bi.Business_name, eventBody.Submission_id, docType, httpClient)
							if docFileErr != nil {
								msg := fmt.Sprintf("Error getting document type '%s' from idpal", docType)
								utils.PrintLogError(docFileErr, componentMessage, methodMsg, msg)
							} else {
								utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL proof of ID uploading")
								uploadProofID(registrationResponse.Piiid, docFilePath)
							}
						}
						// Get the CDD report
						utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL CDD report retrieval")
						cddFilePath, cddFileErr := idpal.GetCDDReport(bi.Business_name, eventBody.Submission_id, httpClient)
						if cddFileErr != nil {
							msg := fmt.Sprintf("Error getting CDD report - Business: %s - Submission: %s - Busines", bi.Business_name, strconv.Itoa(eventBody.Submission_id))
							utils.PrintLogError(cddFileErr, componentMessage, methodMsg, msg)
						} else {
							utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL CDD report uploading")
							uploadProofID(registrationResponse.Piiid, cddFilePath)
						}
					}
					// Update the applink record
					utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL record updated to 'Verified'")
					applinkRecord.Verified = true
					applinkRecord.Piiid = registrationResponse.Piiid
					updateErr := idpalapplinkStorage.UpdateApplink(&applinkRecord)
					if updateErr != nil {
						msg := fmt.Sprintf("Error updating idpal-applink record - Business: %s - Submission: %s - Business", bi.Business_name, strconv.Itoa(eventBody.Submission_id))
						utils.PrintLogError(updateErr, componentMessage, methodMsg, msg)
					}
					c.JSON(200, "Registration of user complete")
					return 
				}
			}
		}
	}
}


func HandleExternalValidationNewSubmissionComplete(c *gin.Context){
	methodMsg := "HandleExternalValidationNewSubmissionComplete"
	signature := c.Request.Header.Get("X-IDPal-Signature")

	businessid := c.Param("businessid")
	var eventBody app.IdpalCallbackEvent

	unmarshallingError := c.Bind(&eventBody)
	if unmarshallingError != nil{
		utils.PrintLogError(unmarshallingError, componentMessage, methodMsg, "Error unmarshalling body")
		c.JSON(400, "Data in body has not expected structure")
		return
	}

	eventBodyAsJSON, payloadErr := json.Marshal(eventBody)
	if payloadErr != nil{
		utils.PrintLogError(payloadErr, componentMessage, methodMsg, "Error marshalling body")
		c.JSON(422, "Data in body cannot be processed properly")
		return
	}
    if !(len(signature)>0) || !strings.Contains(signature, "sha1=") || !verifyIDPalSignature(string(eventBodyAsJSON), signature){
		msg := "Signature not valid"
		utils.PrintLogWarn(errors.New(msg), componentMessage, methodMsg, msg)
		c.JSON(403, msg)
		return 
	}
	// Check if the submission sent in the callback is valid (Status code 4|5)
	utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL received complete submission event - Submission ID: " + strconv.Itoa(eventBody.Submission_id))
	utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL received complete submission event - Event ID: " + strconv.Itoa(eventBody.Event_id))
	utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL completed submission received for UUID: "+eventBody.Uuid)
	insertEventErr := idpalcallbackStorage.InsertIdPalEvent(eventBody)
	if insertEventErr != nil {
		utils.PrintLogError(insertEventErr, componentMessage, methodMsg, "Insertion of IDPAL event into Db failed. We'll keep going...")
	}
	httpClient := idpal.GetRestClient()
	applink, applinkErr := idpal.GetAppLinkStatus(businessid, eventBody.Uuid, httpClient)
	if applinkErr != nil {
		utils.PrintLogError(applinkErr, componentMessage, methodMsg, "Issue when checking applink submission status")
	} else {
		for _, s := range applink.Submissions {
			utils.PrintLogInfo(componentMessage, methodMsg, "IDPAL Submission status (only 4 and 5 will be processed): "+strconv.Itoa(s.Status))
			if s.Status == 4 || s.Status == 5 {
				// ALL RIGHT! PROCESS NEW VALIDATION
				registerNewCustomerRecord(c, businessid, eventBody)
			}
		}
	}
}

func uploadProofID(piiid string, filePath string) error {
	methodMessage := "uploadProofID-IDPal"
	fileName := path.Base(filePath) // name + dot + extension
	extension := path.Ext(filePath) // dot + extension
	msgInitial := fmt.Sprintf("Received file name to upload to Azure Storage: %s - We'll change name to PIIID: %s", fileName, piiid)
	utils.PrintLogInfo(componentMessage, methodMessage, msgInitial)
	fileBytes, readErr := ioutil.ReadFile(filePath)
	if readErr != nil {
		msg := fmt.Sprintf("Error reading file in filesystem: %s - PIIID: %s", fileName, piiid)
		utils.PrintLogError(readErr, componentMessage, methodMessage, msg)
		return readErr
	}
	finalFileName := piiid + extension
	msgUpload := fmt.Sprintf("Start uploading to Azure Storage - File name: %s - PIIID: %s", finalFileName, piiid)
	utils.PrintLogInfo(componentMessage, methodMessage, msgUpload)
	azStorageErr := services.StoreInAzStorageService(finalFileName, fileBytes)
	if azStorageErr != nil {
		var msg = fmt.Sprintf("Error storing uploaded file %s to Azure Storage  PIIID: %s", fileName, piiid)
		utils.PrintLogError(azStorageErr, componentMessage, methodMessage, msg)
		return readErr
	}
	utils.PrintLogInfo(componentMessage, methodMessage, fmt.Sprintf("Successfully saved file '%s' in Azure Storage", finalFileName))
	return nil
}


func makeEventWithCustomerRequest(creq app.RegisterCustomerRequest) (app.Event, error) {
	methodMsg := "makeEventWithCustomerRequest"
	var result app.Event
	var header app.EventHeader
	eventType := config.Eventids.Customerregistry
	corrID, _ := utils.GetCorrelationID(eventType)
	header.Correlation_id = corrID
	header.Event_id = eventType
	header.Created_time = utils.GetEpochNow()
	header.Priority = "HIGH"
	header.Consistency_level = "HIGH"
	body, marErr := json.Marshal(creq)
	if marErr != nil {
		utils.PrintLogError(marErr, componentMessage, methodMsg, "Error marshaling RegisterCustomerRequest")
		return result, marErr
	}
	result.Header = header
	result.Body = body
	return result, nil
}

func makeRegisterCustomerRequest(data app.IDPalCustomerInformation, applinkRecord app.SendAppLinkResponseStorage) (app.RegisterCustomerRequest, error) {
	var idExpiryDate = ""
	if len(data.Idcard_expires) > 0 {
		idExpiryDate = data.Idcard_expires
	} else if len(data.Passport_expires) > 0 {
		idExpiryDate = data.Passport_expires
	}
	var dob string
	formatedDOB, dobErr := utils.GetFormattedDateFromIdpal(data.Dob)
	if dobErr != nil {
		dob = data.Dob
	} else {
		dob = formatedDOB
	}

	var result app.RegisterCustomerRequest
	result.FirstName = data.Firstname
	result.FamilyName = data.LastName
	result.Email = applinkRecord.Basicuseremail
	result.Phone = data.Phone
	result.DOB = dob
	result.IDNumber = ""
	result.IDExpiryDate = idExpiryDate
	result.Nationality = data.Countryofbirth
	result.AddressCountry = data.Country_name
	result.Address1 = data.Address1
	result.Address2 = data.Address2
	result.AddressCity = data.City
	result.AddressPostcode = data.Postalcode
	result.Passwordchangeurl = applinkRecord.Passwordchangeurl
	return result, nil
}



func SendExternalValidationMessage(c *gin.Context) {
	methodMsg := "SendCustomerExternalValidationMessage"
	businessid := c.Param("businessid")
	
	code, biMsg := businessSupportsExtValidation(businessid)
	if code != 200{
		c.JSON(code, biMsg)
		return
	}
	var msg app.SendAppLinkRequest
	unmarshallingError := c.Bind(&msg)
	if unmarshallingError != nil{
		utils.PrintLogError(unmarshallingError, componentMessage, methodMsg, "Error unmarshalling body")
		c.JSON(400, "Data in body is not correct")
		return
	}

	input, _ := json.Marshal(msg)
	correlationID, _ := utils.GetCorrelationID(methodMsg)
	go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Extvalidation, string(input))
	utils.PrintLogInfo(componentMessage, methodMsg, "Event I recorded in event store - " + config.Eventids.Extvalidation)
	res, sendErr := idpal.SendAppLink(businessid, msg, true, idpal.GetRestClient())
	if sendErr != nil {
		msg := "Error using ID Pal API - Send App Link - Business: " + businessid
		utils.PrintLogError(sendErr, componentMessage, methodMsg, msg)
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Extvalidation, msg)
		c.JSON(503, externalServiceErrorMsg)
		return
	}
	utils.PrintLogInfo(componentMessage, methodMsg, "ID Pal API - Send App Link successfully - Business: " + businessid)
	output, _ := json.Marshal(res)
	go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Extvalidation, string(output))
	utils.PrintLogInfo(componentMessage, methodMsg, "Event O recorded in event store - " + config.Eventids.Extvalidation)
	switch res.Status {
		case 0: { 
				utils.PrintLogInfo(componentMessage, methodMsg, "ID-Pal User Message sent")
				res.Status = 200
				c.JSON(200, res)
				return
		}
		case 1, 2: {
				msg := "ID-Pal client or access key missing. Please check post parameters"
				err := errors.New(msg)
				utils.PrintLogError(err, componentMessage, methodMsg, msg)
				c.JSON(400, msg)
				return
		}
		case 3, 4: {
				msg := "ID-Pal client or access key are invalid. Please check post parameters"
				err := errors.New(msg)
				utils.PrintLogError(err, componentMessage, methodMsg, msg)
				c.JSON(400, msg)
				return
		}
		case 5: {
				msg := "ID-Pal sent information type missing or invalid. Please check post parameters"
				err := errors.New(msg)
				utils.PrintLogError(err, componentMessage, methodMsg, msg)
				c.JSON(400, msg)
				return
		}
		default: {
				msg := "ID-Pal response cannot be processes. Please check post parameters and ID Pal availability"
				err := errors.New(msg)
				utils.PrintLogError(err, componentMessage, methodMsg, msg)
				c.JSON(503, msg)
				return
		}
	}
}

func CheckStatusExternalValidation(c *gin.Context) {
	methodMsg := "CheckCustomerExternalValidation"
	businessid := c.Param("businessid")
	uuid := c.Param("uuid")

	if !(len(businessid) > 0) || !(len(uuid) > 0){
		c.JSON(400, "Needed business id and id-pal uuid")
		return
	}

	var result app.IDPalUserStatusResponse

	code, msg := businessSupportsExtValidation(businessid)
	if code != 200{
		result.Status = code
		result.Message = msg
		c.JSON(result.Status, result)
		return
	}
	
	correlationID, _ := utils.GetCorrelationID(methodMsg)
	go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Extvalidation, businessid + "-" + uuid)

	utils.PrintLogInfo(componentMessage, methodMsg, "Event I recorded in event store - " + config.Eventids.Extvalidation)
	applinkRecord, applinkRecordErr := idpalapplinkStorage.GetApplink(uuid)
	if applinkRecordErr != nil {
		msg := "Applink record access failed"
		utils.PrintLogError(applinkRecordErr, componentMessage, methodMsg, msg)
		result.Status = 404
		result.Message = "No submission with that UUID was found"
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Extvalidation, msg)
		c.JSON(result.Status, result)
		return
	}
	if applinkRecord.Verified {
		msg := "Applink record access failed"
		result.Piiid = applinkRecord.Piiid
		result.Status = 200
		result.Message = "UUID verified and User registered"
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Extvalidation, msg)
		c.JSON(200, result)
		return
	}else{
		msg := "UUID found but not verified yet"
		result.Status = 202
		result.Message = msg
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Extvalidation, msg)
		c.JSON(202, result)
		return
	}
}

/*
CUSTOMER GET (EXPECTED PARAMS)
*/
func GetCustomers(c *gin.Context) {
	businessid := c.Param("businessid")
	familyName := c.Query("familyname")
	foreName := c.Query("forename")
	postcode := c.Query("postcode")
	dob := c.Query("dob")
	piiid := c.Query("piiid")
	if len(strings.TrimSpace(businessid)) == 0 {
		c.JSON(400, "Business ID not specified")
		return
	}
	correlationID, _ := utils.GetCorrelationID("GetCustomers")

	if piiid != "" {
			utils.PrintLogInfo(componentMessage, getCustomersMethodMessage, "PIID: "+piiid)
			go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Customerregistry, piiid)
			utils.PrintLogInfo(componentMessage, getCustomersMethodMessage, "Event I recorded in event store")
			GetCustomerRecordByPIIID(c, piiid, correlationID, businessid)
	} else if familyName != "" || foreName != "" || postcode != "" || dob != "" {
			var cq app.CustomerQuery
			cq.Familyname = familyName
			cq.Forename = foreName
			cq.Postcode = postcode
			cq.Dob = dob
			input, _ := json.Marshal(cq)
			go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Customerregistry, string(input))
			utils.PrintLogInfo(componentMessage, getCustomersMethodMessage, "Event I recorded in event store")
			utils.PrintLogInfo(componentMessage, getCustomersMethodMessage, fmt.Sprintf("Family name: %s,  Fore name: %s, Postcode: %s", familyName, foreName, postcode))
			GetCustomerRecordByCustomerRequest(c, cq, correlationID, businessid)
	} else {
			errMessage := "No parameters for search were found"
			go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Customerregistry, errMessage)
			utils.PrintLogInfo(componentMessage, getCustomersMethodMessage, "Events I/O recorded in event store")
			c.JSON(400, errMessage)
			return
	}

	
}

func GetCustomerRecordByCustomerRequest(c *gin.Context, cq app.CustomerQuery, correlationID string, businessid string) {
	utils.PrintLogInfo("HTTP Server", "Query for Customer record", "Customer Query")
	records, recErr := customerStorage.GetCustomers(cq)
	if recErr != nil {
		utils.PrintLogError(recErr, componentMessage, getCustomersMethodMessage, recErr.Error())
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Customerregistry, recErr.Error())
		c.JSON(400, recErr.Error())
		return
	} else if len(records) < 1 {
		message := "No matching records were found"
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Customerregistry, message)
		c.JSON(404, message)
		return
	}
	output, _ := json.Marshal(records)
	go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Customerregistry, string(output))
	utils.PrintLogInfo(componentMessage, getCustomersMethodMessage, "Event O recorded in event store")
	c.JSON(200, records)
	return
}

/*
CUSTOMER GET (EXPECTED PARAM: string piiid)
*/
func GetCustomerRecordByPIIID(c *gin.Context, piiid string, correlationID string, businessid string) {
	utils.PrintLogInfo(componentMessage, "Query for Customer record", "PIID: "+piiid)
	methodMsg := "GetCustomerRecordByPIIID"
	record, recErr := customerStorage.GetCustomerByPIIID(piiid)
	if recErr != nil {
		utils.PrintLogError(recErr, componentMessage, methodMsg, recErr.Error())
		if strings.Contains(recErr.Error(), "not found"){
			msg := "404 - No matching record was found"
			c.JSON(404, msg)
		}else{
			msg := "400 - Not valid query"
			c.JSON(400, msg)
		}
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Customerregistry, recErr.Error())
		utils.PrintLogInfo(componentMessage, methodMsg, "Event O recorded in event store")
		return
	} else if record.Event_group_id == "" {
		msg := "404 - No matching record was found"
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Customerregistry, msg)
		utils.PrintLogInfo(componentMessage, methodMsg, "Event O recorded in event store")
		c.JSON(404, msg)
		return
	}
	output, _ := json.Marshal(record)
	go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Customerregistry, string(output))
	utils.PrintLogInfo(componentMessage, methodMsg, "Event O recorded in event store")
	c.JSON(200, record)
	return
}


func ProcessPostedCustomerRequest(event app.Event, bi *app.RegisteredBusinessInfo, grpcDWClient pb.WriteEventBatchServiceClient, grpcDWConn *grpc.ClientConn, grpcAADClient pb.AADUserServiceClient, grpcAADConn *grpc.ClientConn) app.RegistrationResponse {
	//Initialization
	methodMsg := "ProcessPostedCustomerRequest"
	receptionTime := utils.GetEpochNow()
	correlationID := event.Header.Correlation_id
	var result app.RegistrationResponse = app.RegistrationResponse{}
	var creq app.RegisterCustomerRequest = app.RegisterCustomerRequest{}
	var aadOutput app.RegistrationResponse = app.RegistrationResponse{}
	var isAADRequired = bi.Business_requires_aad

	// 1-Validation of the body
	jsonBody := event.Body
	if jsonBody == nil {
		noContentErr := errors.New("Empty body in event")
		utils.PrintLogError(noContentErr, componentMessage, "New customer -> handling events in event batch request.", "Error in event: "+event.Header.Correlation_id)
		result = buildFailedRegistrationResponse(correlationID, receptionTime, noContentErr.Error())
		return result
	}
	unmarshalErr := json.Unmarshal(jsonBody, &creq)
	if unmarshalErr != nil {
		utils.PrintLogError(unmarshalErr, componentMessage, "New customer -> handling events in event batch request. Unmarshaling body.", "Error in event: "+event.Header.Correlation_id)
		result = buildFailedRegistrationResponse(correlationID, receptionTime, unmarshalErr.Error())
		return result
	}

	// 2-Find possible duplicates
	var customerQuery app.CustomerQuery
	customerQuery.Forename = creq.FirstName
	customerQuery.Familyname = creq.FamilyName
	customerQuery.Postcode = creq.AddressPostcode
	customerQuery.Dob = creq.DOB
	msg := fmt.Sprintf("Query to detect duplicates - Forename: %s - Family Name: %s - Postcode: %s - DOB: %s", customerQuery.Forename, customerQuery.Familyname, customerQuery.Postcode, customerQuery.Dob)
	utils.PrintLogInfo(componentMessage, "Detecting duplicates", msg)
	probableDuplicates, probableDuplicatesErr := customerStorage.GetCustomers(customerQuery)
	if probableDuplicatesErr != nil {
		switch probableDuplicatesErr.Error() {
			case dateMRZFormatNotValid: {
				utils.PrintLogError(probableDuplicatesErr, componentMessage, methodMsg, dateMRZFormatNotValid)
			}
			default: {
				utils.PrintLogError(probableDuplicatesErr, componentMessage, methodMsg, probableDuplicatesErr.Error())
			}
		}
	} else {

		var finalPIIID = ""
		if len(probableDuplicates) == 1 {
			finalPIIID = probableDuplicates[0].Event_group_id
		} else if len(probableDuplicates) > 1 {
			msg := fmt.Sprintf("Found probable duplicates - Forename: %s - Family Name: %s - Postcode: %s - DOB: %s", customerQuery.Forename, customerQuery.Familyname, customerQuery.Postcode, customerQuery.Dob)
			utils.PrintLogError(probableDuplicatesErr, componentMessage, "New customer -> handling events in event batch request.", msg)
			result = buildFailedRegistrationResponse(correlationID, receptionTime, msg)
			return result
		} 
		// 3-Write to STS Customer Registry (ODB) and get PIIID
		if !(len(finalPIIID) > 0) {
			newPIIID, postErr := services.PostRecordToCustomerRegistry(bi, event.Header.Correlation_id, creq, grpcDWClient)
			if postErr != nil {
				utils.PrintLogError(postErr, componentMessage, "New customer -> Validation", "Error when inserting new Customer Request")
				result = buildFailedRegistrationResponse(correlationID, receptionTime, postErr.Error())
				return result
			}
			finalPIIID = newPIIID
		}
		
		// 4-This business description requires to register a new user in AAD
		if isAADRequired {
			output, aadErr := grpcClient.PostCustomerAsNewAADUser(event.Header.Correlation_id, finalPIIID, bi, creq, grpcAADClient)
			if aadErr != nil {
				var msg = fmt.Sprintf("Error registering new user in AAD: %s", aadErr.Error())
				utils.PrintLogError(aadErr, componentMessage, "New customer -> Register to AAD", msg)
				result = buildFailedRegistrationResponse(correlationID, receptionTime, msg)
				return result
			}
			if grpcAADConn != nil {
				grpcAADConn.Close()
			}
			aadOutput = output
		}
		aadOutput.Piiid = finalPIIID
	}
	
	return aadOutput
}

/*
CUSTOMER POST (EXPECTED: app.EventBatch with []app.Event, each one with body containing app.RegisterCustomerRequest)
*/
func PostCustomerRecord(c *gin.Context) {
	var eb app.EventBatch
	bidFromUrl := c.Param("businessid")
	if len(strings.TrimSpace(bidFromUrl)) == 0 {
		c.JSON(400, "Business ID not specified")
		return
	}
	unmarshallingError := c.Bind(&eb)
	methodMessage := "PostCustomerRecord"
	if unmarshallingError != nil {
		utils.PrintLogError(unmarshallingError, componentMessage, "New customer -> Unmarshalling.", "Error when unmarshalling request to the expected object")
		c.JSON(400, unmarshallingError.Error())
		return
	}
	
	businessIdInEB := eb.Business_unit
	if bidFromUrl != businessIdInEB {
		msg := "The Business ID included as parameter and the business ID into the Event Batch do not match"
		err := errors.New(msg)
		utils.PrintLogError(err, componentMessage, "New customer -> Validating request.", "Error checking the business id")
		c.JSON(400, msg)
	}

	events := eb.Events
	eventType := events[0].Header.Event_id
	corrIDSeed := eventType + strconv.Itoa(utils.GetSmallRandom())
	ebCorrID, _ := utils.GetCorrelationID(corrIDSeed)
	eb.Correlation_id = ebCorrID

	bi, biErr := bistorage.GetBusinessInfoByID(bidFromUrl)
	if biErr != nil {
		var msg = fmt.Sprintf("Error looking for business info '%s'.", bidFromUrl)
		utils.PrintLogError(biErr, componentMessage, "New customer -> Checking business info", msg)
		c.JSON(400, msg)
		return
	}
	if len(events) < 1 {
		noEventErr := errors.New("No events found in Event Batch with correlation ID: " + eb.Correlation_id)
		utils.PrintLogError(noEventErr, componentMessage, "New customer -> handling events in event batch request.", "Error")
		c.JSON(400, noEventErr.Error())
		return
	}
	var responsesFromEvents []app.RegistrationResponse
	for _, ev := range events {
		evCorrID, _ := utils.GetCorrelationID(strconv.FormatInt(ev.Header.Created_time, 10))
		ev.Header.Correlation_id = evCorrID
		if ev.Header.Event_id == config.Eventids.Customerregistry {
			payload, _ := json.Marshal(ev.Body)
			go utils.SaveSTSEventToEventStore(evCorrID, businessIdInEB, config.Eventids.Customerregistry, string(payload))
			utils.PrintLogInfo(componentMessage, methodMessage, "Event I recorded in event store")
			grpcDWClient, grpcDWConn, clientErr := GetDWUserServiceClient()
			if clientErr != nil {
				var msg = fmt.Sprintf("Error creating grpc client: %s", clientErr.Error())
				utils.PrintLogError(clientErr, componentMessage, "Create grpc client of DataWriter to register a new customer", msg)
			}
			grpcAADClient, grpcAADConn, clientErr := GetAADUserServiceClient()
			if clientErr != nil {
				var msg = fmt.Sprintf("Error creating grpc client: %s", clientErr.Error())
				utils.PrintLogError(clientErr, componentMessage, "New customer -> Register to AAD", msg)
			}
			registrationResponse := ProcessPostedCustomerRequest(ev, &bi, grpcDWClient, grpcDWConn, grpcAADClient, grpcAADConn)
			if registrationResponse.Result.Error != "" {
				err := errors.New(registrationResponse.Result.Error)
				utils.PrintLogError(err, componentMessage, "New customer -> Handling events.", registrationResponse.Result.Error)
				go utils.SaveSTSEventToEventStore(evCorrID, businessIdInEB, config.Eventids.Customerregistry, registrationResponse.Result.Error)
				utils.PrintLogInfo(componentMessage, methodMessage, "Event O recorded in event store")
				//c.JSON(409, registrationResponse.Result.Error)
				//return
			}else{
				utils.PrintLogInfo(componentMessage, "New customer -> Handling events.", registrationResponse.Result.User_data.DisplayName)
				payload, _ := json.Marshal(registrationResponse)
				go utils.SaveSTSEventToEventStore(evCorrID, businessIdInEB, config.Eventids.Customerregistry, string(payload))
				utils.PrintLogInfo(componentMessage, methodMessage, "Event O recorded in event store")
			}
			responsesFromEvents = append(responsesFromEvents, registrationResponse)
		} else {
			msg := fmt.Sprintf("Event type is not valid: %s. It should be: %s", ev.Header.Event_id, config.Eventids.Customerregistry)
			err := errors.New(msg)
			utils.PrintLogError(err, componentMessage, "New customer -> Handling events.", msg)
			c.JSON(400, msg)
			return
		}
	}
	responseEB := composeUserRegistrationEB(eb, eventType, responsesFromEvents)
	c.JSON(200, responseEB)
	return
}

func UploadProofID(c *gin.Context) {
	bidFromUrl := c.Param("businessid")
	if len(strings.TrimSpace(bidFromUrl)) == 0 {
		c.JSON(400, "Business ID not specified")
		return
	}
	piiid := c.Param("piiid")
	bi, biErr := bistorage.GetBusinessInfoByID(bidFromUrl)
	methodMessage := "New customer -> Upload  Proof ID"
	corrID, _ := utils.GetCorrelationID(bidFromUrl)

	if biErr != nil {
		var msg = fmt.Sprintf("Error looking for business info '%s'.", bidFromUrl)
		utils.PrintLogError(biErr, componentMessage, "New customer -> Upload  Proof ID", msg)
		c.JSON(400, msg)
		return
	}
	go utils.SaveSTSEventToEventStore(corrID, bi.Business_name, config.Eventids.Customerregistry, "Upload file: " + bidFromUrl + "-" + piiid)
	utils.PrintLogInfo(componentMessage, methodMessage, "Event I recorded in event store")

	if bi.Business_requires_proofid {
		
		file, handler, fileErr := c.Request.FormFile("proofid")
		if fileErr != nil {
			var msg = fmt.Sprintf("Invalid uploaded proof of id file %s - PIIID: %s", fileErr.Error(), piiid)
			utils.PrintLogError(fileErr, componentMessage, "New customer -> Upload  Proof ID", msg)
			go utils.SaveSTSEventToEventStore(corrID, bi.Business_name, config.Eventids.Customerregistry, msg)
			utils.PrintLogInfo(componentMessage, methodMessage, "Event O recorded in event store")
			c.JSON(400, msg)
			return
		}

		// Find the customer using the provided PIIID
		existingCust, custErr := getCustomerByPIIID(piiid)
		if custErr != nil || existingCust.Familyname == "" {
			var msg = fmt.Sprintf("The customer does not exist. The proof of id has a wrong PIIID assigned.")
			go utils.SaveSTSEventToEventStore(corrID, bi.Business_name, config.Eventids.Customerregistry, msg)
			utils.PrintLogInfo(componentMessage, methodMessage, "Event O recorded in event store")
			c.JSON(400, msg)
			return
		}

		filename := handler.Filename
		var extension = filepath.Ext(filename)
		var finalFilePath = piiid + extension

		utils.PrintLogInfo(componentMessage, "New customer -> Upload  Proof ID", fmt.Sprintf("File name: %+v\n", handler.Filename))
		utils.PrintLogInfo(componentMessage, "New customer -> Upload  Proof ID", fmt.Sprintf("File Size: %+v\n", handler.Size))
		utils.PrintLogInfo(componentMessage, "New customer -> Upload  Proof ID", fmt.Sprintf("MIME Header: %+v\n", handler.Header))

		// tempFile, temErr := ioutil.TempFile("temp-images", finalFilePath)
		// if temErr != nil {
		// 	fmt.Println(temErr)
		// }
		// defer tempFile.Close()

		fileBytes, readErr := ioutil.ReadAll(file)
		if readErr != nil {
			fmt.Println(readErr)
		}
		azStorageErr := services.StoreInAzStorageService(finalFilePath, fileBytes)
		if azStorageErr != nil {
			var msg = "Error storing uploaded file to Encrypted Storage. Please try later or contact with the Fexco API support."
			utils.PrintLogError(azStorageErr, componentMessage, "New customer -> Upload  Proof ID", msg)
			go utils.SaveSTSEventToEventStore(corrID, bi.Business_name, config.Eventids.Customerregistry, msg)
			utils.PrintLogInfo(componentMessage, methodMessage, "Event O recorded in event store")
			c.JSON(500, msg)
			return
		}
		go utils.SaveSTSEventToEventStore(corrID, bi.Business_name, config.Eventids.Customerregistry, "Uploaded file OK: " + piiid)
		utils.PrintLogInfo(componentMessage, methodMessage, "Event O recorded in event store")
		utils.PrintLogInfo(componentMessage, "New customer -> Upload  Proof ID", fmt.Sprintf("Successfully saved at: %+v\n", finalFilePath))
		c.JSON(200, piiid)
		return
	} else {
		msg := fmt.Sprintf("Operation not supported for this business: %s ", bidFromUrl)
		utils.PrintLogWarn(errors.New(msg), componentMessage, methodMessage, msg)
		c.JSON(403, msg)
		return
	}
}

/*
COMPLIANCE FLOWS MANAGEMENT
*/
func GetComplianceFlow(c *gin.Context) {
	businessid := c.Param("businessid")
	if len(strings.TrimSpace(businessid)) == 0 {
		c.JSON(400, "Business ID not specified")
		return
	}
	complianceFlow, flowErr := getFlow(businessid)
	if flowErr != nil {
		if flowErr.Error() == "404" {
			c.JSON(404, "Flow not found for that business ID")
		} else if flowErr.Error() == "500" {
			c.JSON(500, "Error when creating the DataAccesLayer component")
		} else {
			c.JSON(423, "Operation Not Available")
		}
	} else {
		c.JSON(200, complianceFlow)
	}
	return
}

func PostComplianceFlow(c *gin.Context) {
	var flowRequest app.ComplianceFlow
	unmarshallingError := c.Bind(&flowRequest)
	if unmarshallingError != nil {
		utils.PrintLogWarn(unmarshallingError, componentMessage, "New compliance flow. Unmarshalling", "Error when unmarshalling new compliance flow request to expected object")
		c.JSON(400, unmarshallingError.Error())
		return
	} else {
		validationError := app.ValidateComplianceFlow(flowRequest)
		if validationError != nil {
			utils.PrintLogWarn(validationError, componentMessage, "New Compliance Flow. Validation", "Error when validating new Compliance Flow payload")
			c.JSON(400, validationError.Error())
			return
		} else {
			correlationID, _ := utils.GetCorrelationID("PostComplianceFlow")
			input, _ := json.Marshal(flowRequest)
			go utils.SaveSTSEventToEventStore(correlationID, flowRequest.Business.BusinessName, config.Eventids.Flow, string(input))
			err := flowstorage.InsertFlow(flowRequest)
			if err != nil {
				msg := "Error when inserting new Compliance Flow"
				go utils.SaveSTSEventToEventStore(correlationID, flowRequest.Business.BusinessName, config.Eventids.Flow, msg)
				utils.PrintLogError(err, componentMessage, "New Compliance Flow. Insertion", msg)
				c.JSON(500, err.Error())
				return
			} else {
				msg := "New Compliance flow inserted successfully"
				go utils.SaveSTSEventToEventStore(correlationID, flowRequest.Business.BusinessName, config.Eventids.Flow, msg)
				c.JSON(204, msg)
				return
			}
		}
	}
}

func PutComplianceFlow(c *gin.Context) {
	var flowRequest app.ComplianceFlow
	methodMsg := "PutComplianceFlow"
	unmarshallingError := c.Bind(&flowRequest)
	if unmarshallingError != nil {
		utils.PrintLogWarn(unmarshallingError, componentMessage, methodMsg, "Error when Unmarshalling updated Compliance Flow payload")
		c.JSON(400, unmarshallingError)
		return
	} else {
		validationError := app.ValidateComplianceFlow(flowRequest)
		if validationError != nil {
			utils.PrintLogWarn(validationError, componentMessage, methodMsg, "Error when validating updated Compliance Flow payload")
			c.JSON(400, validationError)
			return
		} else {
			correlationID, _ := utils.GetCorrelationID("PutComplianceFlow")
			input, _ := json.Marshal(flowRequest)
			go utils.SaveSTSEventToEventStore(correlationID, flowRequest.Business.BusinessName, config.Eventids.Flow, string(input))
			err := flowstorage.UpdateFlow(flowRequest)
			if err != nil {
				msg := "Error when updating Compliance Flow with ID: "+flowRequest.FlowID
				go utils.SaveSTSEventToEventStore(correlationID, flowRequest.Business.BusinessName, config.Eventids.Flow, msg)
				utils.PrintLogError(err, componentMessage, methodMsg, msg)
				if err.Error() == "not found" {
					c.JSON(404, err.Error())
					return
				} else {
					c.JSON(500, internalServiceErrorMsg)
					return
				}
			} else {
				msg := "Compliance flow with ID "+flowRequest.FlowID+" updated successfully"
				go utils.SaveSTSEventToEventStore(correlationID, flowRequest.Business.BusinessName, config.Eventids.Flow, msg)
				c.JSON(204, msg)
				return
			}
		}
	}
}

func DeleteComplianceFlow(c *gin.Context) {
	businessid := c.Param("businessid")
	if len(strings.TrimSpace(businessid)) == 0 {
		c.JSON(400, "Business ID not specified")
		return
	}
	correlationID, _ := utils.GetCorrelationID("DeleteComplianceFlow")
	go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Flow, businessid)
	err := flowstorage.DeleteFlow(businessid)
    if err != nil {
		msg := "Failed deletion of the Compliance Flow with Business ID: "+businessid
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Flow, msg)
		utils.PrintLogWarn(err, componentMessage, "Delete Compliance Flow. Data", msg)
		c.JSON(404, err.Error())
		return
	} else {
		msg := "Compliance flow for Business ID "+businessid+" deleted successfully"
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Flow, msg)
		c.JSON(204, msg)
		return
	}
}

/*
UK ADDRESSES FROM POSTCODE
*/
func GetAddressesFromPostcodeUK(c *gin.Context) {
	postcode := c.Param("postcode")
	businessid := c.Param("businessid")
	if len(strings.TrimSpace(businessid)) == 0 {
		c.JSON(400, "Business ID not specified")
		return
	}
	client := services.GetRestClient()
	methodMessage := "GetAddressesFromPostcodeUK"
	correlationID, _ := utils.GetCorrelationID("GetAddressesFromPostcodeUK")
	go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Postcodeuk, businessid)

	utils.PrintLogInfo(componentMessage, getCustomersMethodMessage, "Event I recorded in event store")
	postcodeResultSet, processErr := services.ProcessPostcodeUKRequest(postcode, client)
	if processErr != nil {
		msg := "Error when getting the list of addresses for the postcode: "+postcode
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Postcodeuk, msg)
		utils.PrintLogError(processErr, componentMessage, methodMessage, msg)
		c.JSON(503, processErr.Error())
		return
	} else {
		payload, _ := json.Marshal(postcodeResultSet)
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Postcodeuk, string(payload))
		utils.PrintLogInfo(componentMessage, methodMessage, "Success postcode: "+postcode)
		c.JSON(200, postcodeResultSet)
		return
	}
}

/*
 CARD VALIDATION
*/
func PostCardValidation(c *gin.Context) {
	businessid := c.Param("businessid")
	var cardValidationRequest app.CardValidationRequest
	methodMessage := "Validation of card"
	unmarshallingError := c.Bind(&cardValidationRequest)
	if unmarshallingError != nil {
		utils.PrintLogWarn(unmarshallingError, componentMessage, methodMessage, "Error when unmarshaling the card validation request")
		c.JSON(400, unmarshallingError.Error())
		return
	} else {
		input, _ := json.Marshal(cardValidationRequest)
		stringInput := string(input)
		correlationID, _ := utils.GetCorrelationID(stringInput)
		go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Postcodeuk, stringInput)
		client := services.GetRestClient()
		cardValidationResult, processErr := services.GetCardValidationData(cardValidationRequest, client)
		if processErr == nil {
			output, _ := json.Marshal(cardValidationResult)
			go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Postcodeuk, string(output))
			c.JSON(200, cardValidationResult)
			return
		} else {
			go utils.SaveSTSEventToEventStore(correlationID, businessid, config.Eventids.Postcodeuk, processErr.Error())
			if(strings.Contains(processErr.Error(), "Error invoking external service")){
				utils.PrintLogError(processErr, componentMessage, methodMessage, "External card validation service not available")
				c.JSON(503, externalServiceErrorMsg)
				return
			}else{
				utils.PrintLogError(processErr, componentMessage, methodMessage, "Passed card validation request is not valid")
				c.JSON(422, "Passed card validation request is not valid")
				return
			}
		}
	}
}


/*
CLIENT VALIDATION
*/
func validateIPValidationBody(c *gin.Context, req app.ClientRequestTransaction) {
	if len(req.PublicIP) == 0 || len(req.UserAgent) == 0 || len(req.Domain) == 0{
		err := errors.New("IP Transaction Validation Request is not valid")
		utils.PrintLogError(err, componentMessage, "PostIPSetValidation", err.Error())
		c.JSON(400, err.Error())
		return
	}
}

func validateHash(hash string) error {
	if !(len(hash) > 0){
		msg := "Hash is empty"
		return errors.New(msg)
	}
	decryptedHash, tokenErr := utils.AESdecrypt4TxValidation(hash)
	if tokenErr != nil {
		return tokenErr
	}
	if decryptedHash != config.Ip.ClientToken {
		msg := "Hash is not valid"
		return errors.New(msg)
	}
	return nil
}
/*
* Rules for a valid IP:
* 1-IP not in blacklist
* 2-TODO IP in whitelist
*
*/
func GetAndValidateIP(c *gin.Context){
	ip := c.Param("ip")
	var res app.ValidationResult
	if len(strings.TrimSpace(ip)) == 0 {
		res.Result = ""
		res.Message = "IP_NOT_INCLUDED"
		c.JSON(400, res)
		return
	}
	hash := c.GetHeader("Portal-Hash")
	hashErr := validateHash(hash)
	if hashErr != nil {
		res.Result = "NOT_AUTHORIZED"
		res.Message = hashErr.Error()
		c.JSON(401, res)
		return
	}

	httpClient := services.GetRestClient()
	err := services.CheckBlacklist(ip, httpClient)
	if err != nil {
		res.Result = "NOT_AVAILABLE"
		res.Message = err.Error()
		c.JSON(500, res)
		return
	}
	res.Result = allowed
	c.JSON(200, res)
	return
}

/*
* Rules for a valid transaction:
* 1-IP repeated exceeding number of Transactions in certain periods of time
*
*/
func PostAndValidateIPTx(c *gin.Context){
	methodMessage := "PostAndValidateIPTx"
	var req app.ClientRequestTransaction
	var res app.ValidationResult
	unmarshallingError := c.Bind(&req)
	if unmarshallingError != nil {
		utils.PrintLogError(unmarshallingError, componentMessage, methodMessage, "Error when unmarshalling request to the expected object")
		res.Result = notAllowed
		res.Message = unmarshallingError.Error()
		c.JSON(400, res)
		return
	}
	validateIPValidationBody(c, req)
	req.Time = utils.GetEpochNow()
	validationIPErr := services.ValidateIPTx(&req)
	if validationIPErr != nil {
		res.Result = notAllowed
		res.Message = validationIPErr.Error()
		c.JSON(401, res)
		return
	}
	res.Result = allowed
	c.JSON(200, res)
	return
}

// 2-Email Address Validation (well-formed, no temp)
func GetEmailValidation(c *gin.Context){
	methodMessage := "GetEmailValidation"
	var res app.ValidationResult
	encodedEmail := c.Param("email")
	encodedAt := "%40"

	if !(len(encodedEmail) > 0) {
		msg := "Empty email"
		utils.PrintLogInfo(componentMessage, methodMessage, msg)
		res.Result = ""
		res.Message = msg
		c.JSON(400, res)
		return
	}
	email := strings.Replace(encodedEmail, encodedAt, "@", 1) 

	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	if !(re.MatchString(email)){
		msg := "Not valid email address"
		utils.PrintLogInfo(componentMessage, methodMessage, msg)
		res.Result = ""
		res.Message = msg
		c.JSON(400, res)
		return
	}
	httpClient := services.GetRestClient()
	isTemp, err := services.CheckTempEmail(email, httpClient)
	if err != nil {
		res.Result = ""
		res.Message = err.Error()
		c.JSON(500, res)
		return
	}
	if isTemp {
		res.Result = notAllowed
		res.Message = "Email address in blacklist"
	} else {
		res.Result = allowed
	}
	c.JSON(200, res)
	return
}



/*
PRIVATE FUNCTIONS
*/
func composeUserRegistrationEB(eb app.EventBatch, eventType string, responses []app.RegistrationResponse) app.EventBatch {
	var result app.EventBatch
	result.Correlation_id = eb.Correlation_id
	result.Tenant_id = eb.Tenant_id
	result.Business_unit = eb.Business_unit
	result.Location_id = eb.Location_id
	result.Pos_id = eb.Pos_id
	result.User_id = eb.User_id
	result.Client_id = eb.Client_id
	result.Created_time = eb.Created_time
	var events []app.Event
	for _, res := range responses {
		var ev app.Event
		var evh app.EventHeader
		evh.Correlation_id = res.Correlation_id
		now := utils.GetEpochNow()
		evh.Created_time = now
		evh.Event_id = eventType
		jsonResponse, resErr := json.Marshal(res)
		if resErr == nil {
			ev.Body = jsonResponse
		}
		ev.Header = evh
		events = append(events, ev)
	}
	result.Events = events
	return result
}

func businessSupportsExtValidation(businessid string) (int, string){
	methodMsg := "businessSupportsExtValidation"
	if len(businessid) < 1 {
		return 400, "Business ID not specified"
	}
	bi, biErr := bistorage.GetBusinessInfoByID(businessid)
	if biErr != nil {
		utils.PrintLogError(biErr, componentMessage, methodMsg, "Error getting information about Business Info: " + businessid)
		return 400, "Business not supported"
	}
	if !bi.Business_requires_idpal {
		utils.PrintLogInfo(componentMessage, methodMsg, "Business Info does not support external validation of customers: " + businessid)
		return 400, "Business does not support external validation"
	}
	return 200, ""
}

func getFlow(businessid string) (*app.ComplianceFlow, error) {
	var emptyResult *app.ComplianceFlow
	complianceFlow, err := flowstorage.GetFlow(businessid)
	if err != nil {
		notFoundErr := errors.New("404")
		utils.PrintLogError(err, componentMessage, "Get Compliance Flow. Get Data", "Error when searching for the Compliance Flow by Business")
		return emptyResult, notFoundErr
	}
	return complianceFlow, nil
}

func buildFailedRegistrationResponse(correlation_id string, receptionTime int64, errMsg string) app.RegistrationResponse {
	var aadOutput app.AADNewUserOutput
	var registration app.Registration
	registration.Result = false
	registration.User_data = aadOutput
	registration.Error = errMsg
	var regResponse app.RegistrationResponse
	regResponse.Result = registration
	now := utils.GetEpochNow()
	regResponse.Processing_time = now
	regResponse.Reception_time = receptionTime
	return regResponse
}


func getCustomerByPIIID(piiid string) (app.CustomerSlim, error) {
	var emptyResult app.CustomerSlim
	cust, dataErr := customerStorage.GetCustomerByPIIID(piiid)
	if dataErr != nil {
		utils.PrintLogError(dataErr, componentMessage, "New Customer -> Query for Customer. Get Data", "Error")
		return emptyResult, dataErr
	}
	return cust, nil
}




/*
NZ ADDRESSES FROM POSTCODE
*/
/*func GetAddressesFromPostcodeNZ(c *gin.Context) {
	address := c.Param("address")
	client := services.GetRestClient()
	postcodeResultSet, processErr := services.ProcessPostcodeNZRequest(address, client)
	if processErr != nil {
		utils.PrintLogError(processErr, componentMessage, "Getting addresses from a postcode (NZ)", "Error when getting the list of addresses for the address: "+address)
		c.JSON(503, processErr.Error())
		return
	} else {
		utils.PrintLogInfo(componentMessage, "Getting addresses from an address (NZ)", "Success postcode: "+address)
		c.JSON(200, postcodeResultSet)
		return
	}
}*/

/*
COMPLIANCE REQUEST

func PostComplianceRequest(c *gin.Context) {
	var compReq app.ComplianceRequest
	unmarshallingError := c.Bind(&compReq)
	if unmarshallingError != nil {
		utils.PrintLogWarn(unmarshallingError, componentMessage, "Received compliance query. Unmarshalling", "Error when unmarshalling compliance request to the expected object")
		c.JSON(400, unmarshallingError.Error())
		return
	} else {
		cr, validationError := app.ValidateComplianceRequest(compReq)
		if validationError != nil {
			utils.PrintLogWarn(validationError, componentMessage, "Received compliance query. Validation", "Error in the validation of compliance request payload")
			c.JSON(400, validationError.Error())
			return
		} else {
			complianceResult, processErr := services.ProcessComplianceRequest(cr)
			if processErr != nil {
				utils.PrintLogError(processErr, componentMessage, "Received compliance query. Processing", "Error in the processing of validated compliance request")
				c.JSON(500, processErr.Error())
				return
			} else {
				utils.PrintLogInfo(componentMessage, "Received compliance query. Processing", "Successfully processed compliance query with result")
				c.JSON(200, complianceResult)
				return
			}
		}
	}
}
*/