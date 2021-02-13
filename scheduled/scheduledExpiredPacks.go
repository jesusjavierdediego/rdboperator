package scheduled

import (
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"strconv"
	configuration "xqledger/gitreader/configuration"
	model "xqledger/gitreader/model"
	pki "xqledger/gitreader/pki"
	servicebus "xqledger/gitreader/servicebus"
	storagepki "xqledger/gitreader/storage_PKI"
	utils "xqledger/gitreader/utils"
)

const componentMessage = "Scheduled PKI Service"
const schComponentMessage = "*** Scheduled process for PKI Records checking"

var config = configuration.GlobalConfiguration

func ReviewIDPacks() {
	methodMessage := "ReviewIDPacks"

	records, dbErr := storagepki.GetAllActive()
	if dbErr != nil {
		utils.PrintLogError(dbErr, schComponentMessage, methodMessage, "Error when retrieving all PKI records")
		return
	}
	if len(records) > 0 {
		storagepki.UpdateAll4Maintenance(config.MaintenanceStatus.Free, config.MaintenanceStatus.OnMaintenance)
		now := utils.GetEpochNow()
		for _, r := range records {
			scheduledProcessPKIRecord(r, now)
		}
		storagepki.UpdateAll4Maintenance(config.MaintenanceStatus.OnMaintenance, config.MaintenanceStatus.Free)
	} else {
		noRecordsErr := errors.New("No PKI records found free for maintenance")
		utils.PrintLogError(noRecordsErr, schComponentMessage, methodMessage, "Error when retrieving all PKI records")
		return
	}
}

// Revoke the certificates entries when they expire
func scheduledProcessPKIRecord(r *model.PKIRecord, now int64) {
	if r.Expiry_time < now {
		methodMessage := "scheduledProcessPKIRecord"
		// Update PKI DB
		revocation, revokeDBErr := pki.RevokeIDPackInPKIDB(r.PIIID, r.CompanyID, r.Serial_number, r.Expiry_time)
		if revokeDBErr != nil {
			msg := fmt.Sprintf("Error in revocation of ID pack in PKI DB. PIIID: %s - Entity ID: %s", r.PIIID, r.CompanyID)
			utils.PrintLogError(revokeDBErr, componentMessage, methodMessage, msg)
		}
		// Update CRL
		var revocationList []pkix.RevokedCertificate
		revocationList = append(revocationList, revocation)
		if len(revocationList) > 0 {
			for _, revocation := range revocationList {
				updateCRLErr := pki.UpdateCRL(revocation)
				if updateCRLErr != nil {
					msg := fmt.Sprintf("Error when updating CRL (revocation) with PIIID %s and EntityID %s", r.PIIID, strconv.FormatInt(r.Expiry_time, 10))
					utils.PrintLogError(updateCRLErr, schComponentMessage, methodMessage, msg)
				}
			}
		}
		msg := fmt.Sprintf("Digital Identity for user '%s' and company '%s' has expired. The status now is 'Expired' and revoked", r.AADUserName, r.CompanyID)
		sendNotificationErr := servicebus.SendTopicMessage(msg)
		if sendNotificationErr != nil {
			utils.PrintLogError(sendNotificationErr, schComponentMessage, methodMessage, fmt.Sprintf("Error sending message of expiration for DID  with user '%s' and EntityID '%s'", r.AADUserName, r.CompanyID))
		}
		utils.PrintLogInfo(schComponentMessage, methodMessage, fmt.Sprintf("Successfully sent message of expiration for DID  with user '%s' and EntityID '%s'", r.AADUserName, r.CompanyID))
	}
}
