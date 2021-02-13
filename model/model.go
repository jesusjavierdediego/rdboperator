package model

/*
STS Event Store
*/
type STSEvent struct {
	Correlation_id  string       `json:"correlation_id" binding:"required"`
	Tenant_id      string  `json:"tenant_id" binding:"required"`
	Business_unit  string  `json:"business_unit"  binding:"required"`
	Created_time   int64     `json:"created_time" binding:"required"`
	Event_id          string          `json:"event_id" binding:"required"`
	Body string `json:"body" binding:"required"`
}

type CustomerSlim struct {
	Forename   string `json:"forename"`
	FamilyName string `json:"familyname"`
	Secondfamilyname string `json:"secondfamilyname"`
	Postcode string `json:"Postcode"`
	Dob string `json:"dob"`
	Event_group_id        string `json:"event_group_id"`
}

type CompanySlim struct {
	EntityID   string `json:"EntityID"`
	Name string `json:"Name"`
}

/*
Each record represents an ID Pack for a given user/entity relationship.
A pack consists of:
	- X509 certificate
	- Private Key
	- Pfx file (containing cert and PK)
*/
type PKIRecord struct {
	AADUserName   string `json:"aad_username"` // <user>@<domain>
	PIIID   string `json:"piiid"` // Use the PIIID as the unique ID for the Az Key Vault
	CompleteName string `json:"complete_name"`
	CompanyID string `json:"companyid"` // The cardinality is: 1 Person / 1 Company / 1 Cert / 1 PK
	Isrevoked bool `json:"isrevoked"`
	Revocation_time int64 `json:"revocation_time"`
	Creation_time int64 `json:"creation_time"`
	Expiry_time int64 `json:"expiry_time"`
	Serial_number string `json:"serial_number"`
	Unique_reference string `json:"unique_reference"`
	Purposes []string `json:"purposes"`
	Kty string `json:"kty"`
	Remarks string `json:"remarks"`
	MaintenanceStatus string `json:"maintenance_status"`
}

type CertRequest struct {
	AADUserName   string `json:"aad_username"` // <user>@<domain>
	PIIID   string `json:"piiid"`
	Organization string `json:"organization"`
	OrganizationalUnit string `json:"organizational_unit"`
	Country string `json:"country"`
	Province string `json:"province"`
	City string `json:"city"`
	Address string `json:"address"`
	Postcode string `json:"postcode"`
	CommonName string `json:"common_name"`
	Type string `json:"type"`
}

// AKV
type TokenResponse struct {
	Token_type   string `json:"token_type"`
	Expires_in   int32 `json:"expires_in"`
	Ext_expires_in   int32 `json:"ext_expires_in"`
	Access_token   string `json:"access_token"`
}

type GenericResponseAttributes struct {
	enabled   bool `json:"enabled"`
	Nbf   int64 `json:"nbf"`
	Exp   int64 `json:"exp"`
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`
	RecoveryLevel   string `json:"recoveryLevel"`
}

type SecretRequest struct {
	Value   string `json:"value"`
}

type SecretResponse struct {
	Id   string `json:"id"`
	Value   string `json:"value"`
	Attributes GenericResponseAttributes `json:"attributes"`
}

/*
NOT USED
*/
/*
type Key struct {// Rference: https://docs.microsoft.com/en-us/rest/api/keyvault/getkey/getkey#jsonwebkey
	Kid   string `json:"kid"` // Key identifier
	Kty   string `json:"kty"` // JsonWebKey Key Type (kty), as defined in https://tools.ietf.org/html/draft-ietf-jose-json-web-algorithms-40
	Key_ops   []string `json:"key_ops"` // Supported key operations
	N   string `json:"n"` // RSA modulus
	E   string `json:"e"` //RSA public exponent
}
type PrivateKeyResponse struct {
	Key   Key `json:"key"`
	Attributes GenericResponseAttributes `json:"attributes"`
}
type CertificateResponse struct {
	Id   string `json:"id"`
	Kid   string `json:"kid"`
	Sid   string `json:"sid"`
	X5t   string `json:"x5t"`
	Cer   string `json:"cer"`
	Attributes GenericResponseAttributes `json:"attributes"`
}
*/