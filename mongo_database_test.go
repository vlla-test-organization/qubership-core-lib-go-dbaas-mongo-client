package mongodbaas

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	dbaasbase "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3"
	"github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/model"
	. "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dbaasAgentUrlEnvName     = "dbaas.agent"
	namespaceEnvName         = "microservice.namespace"
	testServiceName          = "service_test"

	createDatabaseV3         = "/api/v3/dbaas/test_namespace/databases"
	getDatabaseV3            = "/api/v3/dbaas/test_namespace/databases/get-by-classifier/mongodb"
	username                 = "service_test"
	password                 = "qwerty127"
	testToken                = "test-token"
	testTokenExpiresIn       = 300
)

type DatabaseTestSuite struct {
	suite.Suite
	database Database
}

func (suite *DatabaseTestSuite) SetupSuite() {
	StartMockServer()
	os.Setenv(dbaasAgentUrlEnvName, GetMockServerUrl())
	os.Setenv(namespaceEnvName, "test_namespace")
	os.Setenv(propMicroserviceName, testServiceName)

	yamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(yamlParams))
}

func (suite *DatabaseTestSuite) TearDownSuite() {
	os.Unsetenv(dbaasAgentUrlEnvName)
	os.Unsetenv(namespaceEnvName)
	os.Unsetenv(propMicroserviceName)
	StopMockServer()
}

func (suite *DatabaseTestSuite) BeforeTest(suiteName, testName string) {
	suite.T().Cleanup(ClearHandlers)
	dbaasPool := dbaasbase.NewDbaaSPool()
	client := NewClient(dbaasPool)
	suite.database = client.ServiceDatabase()
}

func (suite *DatabaseTestSuite) TestServiceDbaasMongoClient_FindDbaasMongoDbConnection() {
	AddHandler(Contains(getDatabaseV3), defaultDbaasResponseHandler)

	ctx := context.Background()
	actualResponse, err := suite.database.FindConnectionProperties(ctx)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), password, actualResponse.Password)
	assert.Equal(suite.T(), username, actualResponse.Username)
}

func (suite *DatabaseTestSuite) TestServiceDbaasMongoClient_FindDbaasMongoDbConnection_ConnectionNotFound() {
	yamlParams := configloader.YamlPropertySourceParams{ConfigFilePath: "testdata/application.yaml"}
	configloader.InitWithSourcesArray(configloader.BasePropertySources(yamlParams))
	AddHandler(Contains(getDatabaseV3), func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	})
	ctx := context.Background()

	if _, err := suite.database.FindConnectionProperties(ctx); assert.Error(suite.T(), err) {
		assert.IsType(suite.T(), model.DbaaSCreateDbError{}, err)
		assert.Equal(suite.T(), 404, err.(model.DbaaSCreateDbError).HttpCode)
	}
}

func (suite *DatabaseTestSuite) TestServiceDbaasMongoClient_GetDbaaSMongoDbConnection() {
	AddHandler(Contains(createDatabaseV3), defaultDbaasResponseHandler)
	ctx := context.Background()
	actualResponse, err := suite.database.GetConnectionProperties(ctx)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), password, actualResponse.Password)
	assert.Equal(suite.T(), username, actualResponse.Username)
}

func (suite *DatabaseTestSuite) TestServiceDbaasMongoClient_GetDbaaSMongoDbConnection_ConnectionNotFound() {
	AddHandler(Contains(createDatabaseV3), func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	})
	ctx := context.Background()

	if _, err := suite.database.GetConnectionProperties(ctx); assert.Error(suite.T(), err) {
		assert.IsType(suite.T(), model.DbaaSCreateDbError{}, err)
		assert.Equal(suite.T(), 404, err.(model.DbaaSCreateDbError).HttpCode)
	}
}

func (suite *DatabaseTestSuite) TestServiceDbaasMongoClient_GetMongoDbClient_WithoutOptions() {
	actualPgClient, err := suite.database.GetMongoDbClient()
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), actualPgClient)
}

func (suite *DatabaseTestSuite) TestServiceDbaasMongoClient_GetMongoDbClient_WithOptions() {
	testOpts := options.Client().SetMaxPoolSize(7)
	actualPgClient, err := suite.database.GetMongoDbClient(testOpts)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), actualPgClient)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}

func defaultDbaasResponseHandler(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusOK)
	connectionProperties := map[string]interface{}{
		"password":   "qwerty127",
		"url":        "mongodb://mongodb0.example.com:27017/name",
		"username":   "service_test",
		"authDbName": "name",
	}
	dbResponse := model.LogicalDb{
		Id:                   "123",
		ConnectionProperties: connectionProperties,
	}
	jsonResponse, _ := json.Marshal(dbResponse)
	writer.Write(jsonResponse)
}

