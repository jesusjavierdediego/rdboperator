package configuration

import (
	"log"
	"os"
	"reflect"
	"strings"
	"github.com/spf13/viper"
)

var GlobalConfiguration Configuration

type Configuration struct {
	Profile        string
	Obfuscationkey string
	Eventid string
	Scheduledfreq int
	Messagebustopic messagebustopic
	Pki		pki
	GrpcServer     grpcServer
	Cosmosdb cosmosdb
	Azkv azkv
	Azstoreusers 	azstoreusers
	Azstoreca 	azstoreca
	MaintenanceStatus maintenanceStatus
	Grpcresponse grpcresponse
}


type grpcresponse struct {
	Ok string
	Cancelled string
  	Invalid string
  	Notfound string
	Permissiondenied string
	Unauthenticated string
	Notimplemented string
	Internal string
	Notavailable string
	Alreadyexists string 
}

type cosmosdb struct {
	Connstring       string
	Database string
	Pki       string
	Events     string
	Customers   string
}

type messagebustopic struct {
	Namespace string
	Keyname string
	Keyvalue string
	Topicname string
}

type pki struct {
	Entitymodeenabled bool
	Certexpirationtimeyears    int
	Keybits    int
	Keyalgorithm string
	Caremotelocation string
	Rootcertfilename    string
	Cacertfilename    string
	Cakeyfilename    string
	Cacrlfilename string
	Timeformat string
	Standardissuername string
	Standardissuercountry string
	Standardissuerpostcode string
	Standardissuercounty string
	Standardissuercity string
	Standardissueraddress string
	Alloweddomains []string
	Allowedauthtokensbasic []string
	Allowedauthtokensadvanced []string
}

type maintenanceStatus struct {
	Free    string
	OnMaintenance    string
}

type grpcServer struct {
	Port    int
	Name    string
	Timeout int32
	Mode    string
	Path    string
}


type azkv struct {
	Baseurl string
  	Subscriptionid string
  	Tenant string
  	Directoryid string
  	Azureappclientid string
	Azureappclientsecret string
}

type azstoreusers struct {
	Account       string
	Key       string
	Name     string
}

type azstoreca struct {
	Account       string
	Key       string
	Name     string
}

func init() {
	GlobalConfiguration = initConfiguration()
}

/*
InitConfiguration Return the configuration
Read the default configuration from application.yml. If PROFILE=dev then use application.dev.xml

To override default param must be run with system ENV, follow the same structure of of yaml, but points is replace by _
example:

	server:
  		port: 8080
  		name: "API-endpoint"
  		timeout: 5
  		key:
		mode: debug

To override the port SERVER_PORT=8181

SERVER_PORT=8181 go run main.go
*/
func initConfiguration() Configuration {
	var configuration Configuration

	profile := os.Getenv("PROFILE")
	//ENV VARS
	viper.AutomaticEnv()                                   // Automatic binding from enviroment
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // to charge enviroment
	viper.SetConfigName("application.dev")

	if strings.ToLower(profile) == "dev" {
		viper.SetConfigName("application.dev")
	} else {
		viper.SetConfigName("application")
	}

	viper.SetConfigType("yaml")
	path := calculatePath("resources")

	viper.AddConfigPath(path)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("ERROR: Error reading config file, %s", err)
	} else {
		err := viper.Unmarshal(&configuration)
		if err != nil {
			log.Fatalf("ERROR: unable to decode into struct, %v", err)
		} else {
			log.Printf("Internal configuration loaded OK")
		}
	}
	return configuration
}
func Reload() {
	GlobalConfiguration = initConfiguration()
}

/*
calculatePath get the configuration path relative to package of configuration and the currentDir of execution
*/
func calculatePath(resourcesPath string) string {

	configurationPatch := reflect.TypeOf(Configuration{}).PkgPath()
	index := strings.LastIndex(configurationPatch, "/")
	configurationPatch = configurationPatch[0:index]

	currentDir, _ := os.Getwd()
	index = strings.LastIndex(currentDir, configurationPatch)
	if index > 0 {
		currentDir = currentDir[0:index]
	}
	currentDir = currentDir + configurationPatch + "/" + resourcesPath
	fileInfo, _ := os.Lstat(currentDir)
	if fileInfo == nil {
		currentDir = "/" + resourcesPath
		fileInfo, _ = os.Lstat(currentDir)
		if fileInfo == nil {
			currentDir = "/"
		}
	}

	return currentDir
}

