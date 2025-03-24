package mongodbaas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/docker/go-connections/nat"
	dbaasbase "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3"
	"github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/cache"
	dbaasbasemodel "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/model"
	"github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/model/rest"
	. "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/testutils"
	"github.com/netcracker/qubership-core-lib-go-dbaas-mongo-client/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoPort             = "27017"
	testContainerPassword = "123qwerty"
	testContainerUser     = "mongouser"
	testContainerDb       = "admin"
	wrongPassword         = "wrongPwd"
)

func (suite *DatabaseTestSuite) TestMongoClient_NewClient() {
	ctx := context.Background()
	mongoContainer := suite.prepareTestContainer(ctx)
	defer func() {
		err := mongoContainer.Terminate(ctx)
		if err != nil {
			suite.T().Fatal(err)
		}
	}()
	addr, err := mongoContainer.Endpoint(ctx, "")
	if err != nil {
		suite.T().Error(err)
	}

	AddHandler(Contains(createDatabaseV3), func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		jsonString := mongoDbaasResponseHandler(addr, testContainerPassword)
		writer.Write(jsonString)
	})

	params := model.DbParams{Classifier: ServiceClassifier, BaseDbParams: rest.BaseDbParams{}}

	mongoClient := MongoDbClientImpl{
		options:      &options.ClientOptions{},
		dbaasClient:  dbaasbase.NewDbaasClient(),
		mongodbCache: &cache.DbaaSCache{LogicalDbCache: make(map[cache.Key]interface{})},
		params:       params,
	}
	database, err := mongoClient.GetMongoDatabase(ctx)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), database)

	suite.checkConnectionIsWorking(database, ctx)
}

func (suite *DatabaseTestSuite) TestMongoClient_NewClientV3() {
	ctx := context.Background()
	mongoContainer := suite.prepareTestContainer(ctx)
	defer func() {
		err := mongoContainer.Terminate(ctx)
		if err != nil {
			suite.T().Fatal(err)
		}
	}()
	addr, err := mongoContainer.Endpoint(ctx, "")
	if err != nil {
		suite.T().Error(err)
	}

	AddHandler(Contains(createDatabaseV3), func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		jsonString := mongoDbaasResponseHandler(addr, testContainerPassword)
		writer.Write(jsonString)
	})

	params := model.DbParams{Classifier: ServiceClassifier, BaseDbParams: rest.BaseDbParams{}}

	mongoClient := MongoDbClientImpl{
		options:      &options.ClientOptions{},
		dbaasClient:  dbaasbase.NewDbaasClient(),
		mongodbCache: &cache.DbaaSCache{LogicalDbCache: make(map[cache.Key]interface{})},
		params:       params,
	}
	database, err := mongoClient.GetMongoDatabase(ctx)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), database)

	suite.checkConnectionIsWorking(database, ctx)
}

func (suite *DatabaseTestSuite) TestMongoClient_GetFromCache() {
	ctx := context.Background()
	mongoContainer := suite.prepareTestContainer(ctx)
	defer func() {
		err := mongoContainer.Terminate(ctx)
		if err != nil {
			suite.T().Fatal(err)
		}
	}()
	addr, err := mongoContainer.Endpoint(ctx, "")
	if err != nil {
		suite.T().Error(err)
	}
	counter := 0
	AddHandler(Contains(createDatabaseV3), func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		jsonString := mongoDbaasResponseHandler(addr, testContainerPassword)
		writer.Write(jsonString)
		counter++
	})

	params := model.DbParams{Classifier: ServiceClassifier, BaseDbParams: rest.BaseDbParams{}}

	mongoClient := MongoDbClientImpl{
		options:      &options.ClientOptions{},
		dbaasClient:  dbaasbase.NewDbaasClient(),
		mongodbCache: &cache.DbaaSCache{LogicalDbCache: make(map[cache.Key]interface{})},
		params:       params,
	}
	firstDatabase, err := mongoClient.GetMongoDatabase(ctx)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), firstDatabase)
	suite.checkConnectionIsWorking(firstDatabase, ctx)

	secondDatabase, err := mongoClient.GetMongoDatabase(ctx)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), secondDatabase)
	assert.Equal(suite.T(), 1, counter)
	suite.checkConnectionIsWorking(secondDatabase, ctx)
}

func (suite *DatabaseTestSuite) TestMongoClient_GetMongoDatabase_WithLogicalProvider() {
	ctx := context.Background()
	mongoContainer := suite.prepareTestContainer(ctx)
	defer func() {
		err := mongoContainer.Terminate(ctx)
		if err != nil {
			suite.T().Fatal(err)
		}
	}()
	addr, err := mongoContainer.Endpoint(ctx, "")
	mongoURI := fmt.Sprintf("mongodb://%s/%s", addr, testContainerDb)

	connectionProperties := map[string]interface{}{
		"password":   testContainerPassword,
		"url":        mongoURI,
		"username":   testContainerUser,
		"authDbName": testContainerDb,
	}

	logicalProvider := &TestLogicalDbProvider{ConnectionProperties: connectionProperties, providerCalls: 0}
	dbaasPool := dbaasbase.NewDbaaSPool(dbaasbasemodel.PoolOptions{
		LogicalDbProviders: []dbaasbasemodel.LogicalDbProvider{
			logicalProvider,
		},
	})
	client := NewClient(dbaasPool)
	database := client.ServiceDatabase()
	mongoClient, _ := database.GetMongoDbClient()
	mongoDb, err := mongoClient.GetMongoDatabase(ctx)
	assert.Nil(suite.T(), err)
	assert.NotEqual(suite.T(), 0, logicalProvider.providerCalls)
	suite.checkConnectionIsWorking(mongoDb, ctx)
}

func (suite *DatabaseTestSuite) TestMongoDbClient_GetMongoDatabase_UpdatePassword() {
	ctx := context.Background()
	mongoContainer := suite.prepareTestContainer(ctx)
	defer func() {
		err := mongoContainer.Terminate(ctx)
		if err != nil {
			suite.T().Fatal(err)
		}
	}()
	addr, err := mongoContainer.Endpoint(ctx, "")
	if err != nil {
		suite.T().Error(err)
	}

	AddHandler(matches(createDatabaseV3), func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		jsonString := mongoDbaasResponseHandler(addr, wrongPassword)
		writer.Write(jsonString)
	})
	AddHandler(matches(getDatabaseV3), func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		jsonString := mongoDbaasResponseHandler(addr, testContainerPassword)
		writer.Write(jsonString)
	})

	params := model.DbParams{Classifier: ServiceClassifier, BaseDbParams: rest.BaseDbParams{}}
	mongoClient := MongoDbClientImpl{
		options:      &options.ClientOptions{},
		dbaasClient:  dbaasbase.NewDbaasClient(),
		mongodbCache: &cache.DbaaSCache{LogicalDbCache: make(map[cache.Key]interface{})},
		params:       params,
	}
	database, err := mongoClient.GetMongoDatabase(ctx)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), database)
	suite.checkConnectionIsWorking(database, ctx)
}

func (suite *DatabaseTestSuite) prepareTestContainer(ctx context.Context) testcontainers.Container {
	env := make(map[string]string)
	env["MONGO_INITDB_ROOT_USERNAME"] = testContainerUser
	env["MONGO_INITDB_ROOT_PASSWORD"] = testContainerPassword

	port, _ := nat.NewPort("tcp", mongoPort)
	req := testcontainers.ContainerRequest{
		Image:        "mongo:7.0",
		ExposedPorts: []string{port.Port()},
		SkipReaper:   true,
		WaitingFor:   wait.ForListeningPort(port),
		Env:          env,
	}
	mongoContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		suite.T().Fatal(err)
	}
	return mongoContainer
}

func matches(submatch string) func(string) bool {
	return func(path string) bool {
		return strings.EqualFold(path, submatch)
	}
}

func (suite *DatabaseTestSuite) checkConnectionIsWorking(database *mongo.Database, ctx context.Context) {
	title := "one"
	collection := database.Collection("podcasts")
	_, errInsert := collection.InsertOne(ctx, bson.D{
		{Key: "title", Value: title},
	})
	assert.Nil(suite.T(), errInsert)
	cursor, errSelect := collection.Find(ctx, bson.M{})
	defer cursor.Close(ctx)
	assert.Nil(suite.T(), errSelect)
	var episodes []bson.M
	errCopy := cursor.All(ctx, &episodes)
	assert.Nil(suite.T(), errCopy)
	actualTitle := episodes[0]
	assert.Equal(suite.T(), title, actualTitle["title"])
	_, errDelete := collection.DeleteMany(ctx, bson.M{"title": title})
	assert.Nil(suite.T(), errDelete)
}

func mongoDbaasResponseHandler(address, password string) []byte {
	mongoURI := fmt.Sprintf("mongodb://%s/%s", address, testContainerDb)

	connectionProperties := map[string]interface{}{
		"password":   password,
		"url":        mongoURI,
		"username":   testContainerUser,
		"authDbName": testContainerDb,
	}
	dbResponse := dbaasbasemodel.LogicalDb{
		Id:                   "123",
		ConnectionProperties: connectionProperties,
		Name:                 testContainerDb,
	}
	jsonResponse, _ := json.Marshal(dbResponse)
	return jsonResponse
}

type TestLogicalDbProvider struct {
	ConnectionProperties map[string]interface{}
	providerCalls        int
}

func (p *TestLogicalDbProvider) GetOrCreateDb(dbType string, classifier map[string]interface{}, params rest.BaseDbParams) (*dbaasbasemodel.LogicalDb, error) {
	p.providerCalls++
	return &dbaasbasemodel.LogicalDb{
		Id:                   "123",
		ConnectionProperties: p.ConnectionProperties,
	}, nil
}

func (p *TestLogicalDbProvider) GetConnection(dbType string, classifier map[string]interface{}, params rest.BaseDbParams) (map[string]interface{}, error) {
	p.providerCalls++
	return p.ConnectionProperties, nil
}
