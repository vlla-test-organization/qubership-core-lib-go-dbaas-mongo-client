package mongodbaas

import (
	"context"

	dbaasbase "github.com/vlla-test-organization/qubership-core-lib-go-dbaas-base-client/v3"
	"github.com/vlla-test-organization/qubership-core-lib-go-dbaas-base-client/v3/cache"
	"github.com/vlla-test-organization/qubership-core-lib-go-dbaas-mongo-client/v3/model"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database interface {
	GetMongoDbClient(options ...*options.ClientOptions) (MongoDbClient, error)
	GetConnectionProperties(ctx context.Context) (*model.MongoConnProperties, error)
	FindConnectionProperties(ctx context.Context) (*model.MongoConnProperties, error)
}

type mongoDatabase struct {
	dbaasPool  *dbaasbase.DbaaSPool
	params     model.DbParams
	mongoCache *cache.DbaaSCache
}

func (d mongoDatabase) GetMongoDbClient(opts ...*options.ClientOptions) (MongoDbClient, error) {
	clientOptions := &options.ClientOptions{}
	if opts != nil {
		clientOptions = opts[0]
	}
	return &MongoDbClientImpl{
		options:      clientOptions,
		dbaasClient:  d.dbaasPool.Client,
		mongodbCache: d.mongoCache,
		params:       d.params,
	}, nil
}

func (d mongoDatabase) GetConnectionProperties(ctx context.Context) (*model.MongoConnProperties, error) {
	classifier := d.params.Classifier(ctx)
	mongoLogicalDb, err := d.dbaasPool.GetOrCreateDb(ctx, DB_TYPE, classifier, d.params.BaseDbParams)
	if err != nil {
		logger.Error("Error acquiring connection properties from DBAAS: %v", err)
		return nil, err
	}

	mongoConnProperties := toMongoConnProperties(mongoLogicalDb.ConnectionProperties)
	return &mongoConnProperties, nil
}

func (d mongoDatabase) FindConnectionProperties(ctx context.Context) (*model.MongoConnProperties, error) {
	classifier := d.params.Classifier(ctx)
	responseBody, err := d.dbaasPool.GetConnection(ctx, DB_TYPE, classifier, d.params.BaseDbParams)
	if err != nil {
		logger.ErrorC(ctx, "Error finding connection properties from DBAAS: %v", err)
		return nil, err
	}
	mongoConnProperties := toMongoConnProperties(responseBody)
	return &mongoConnProperties, err
}

func toMongoConnProperties(connParams map[string]interface{}) model.MongoConnProperties {
	return model.MongoConnProperties{
		Url:        connParams["url"].(string),
		Username:   connParams["username"].(string),
		Password:   connParams["password"].(string),
		AuthDbName: connParams["authDbName"].(string),
		DbName:     getDbName(connParams),
	}
}

func getDbName(connection map[string]interface{}) string {
	if dbName, ok := connection["dbName"]; ok && dbName != "" {
		return dbName.(string)
	}
	return connection["authDbName"].(string)
}
