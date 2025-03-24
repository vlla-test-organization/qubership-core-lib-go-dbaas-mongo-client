package mongodbaas

import (
	"context"

	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	dbaasbase "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3"
	"github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/cache"
	"github.com/netcracker/qubership-core-lib-go-dbaas-mongo-client/v3/model"
)

var logger logging.Logger

func init() {
	logger = logging.GetLogger("mongodbaas")
}

const (
	propMicroserviceName = "microservice.name"
	DB_TYPE              = "mongodb"
)

type DbaaSMongoDbClient struct {
	mongoClientCache cache.DbaaSCache
	pool             *dbaasbase.DbaaSPool
}

func NewClient(pool *dbaasbase.DbaaSPool) *DbaaSMongoDbClient {
	localCache := cache.DbaaSCache{
		LogicalDbCache: make(map[cache.Key]interface{}),
	}
	return &DbaaSMongoDbClient{
		mongoClientCache: localCache,
		pool:             pool,
	}
}

func (d *DbaaSMongoDbClient) ServiceDatabase(params ...model.DbParams) Database {
	return &mongoDatabase{
		params:     d.buildServiceDbParams(params),
		dbaasPool:  d.pool,
		mongoCache: &d.mongoClientCache,
	}
}

func (d *DbaaSMongoDbClient) TenantDatabase(params ...model.DbParams) Database {
	return &mongoDatabase{
		params:     d.buildTenantDbParams(params),
		dbaasPool:  d.pool,
		mongoCache: &d.mongoClientCache,
	}
}

func ServiceClassifier(ctx context.Context) map[string]interface{} {
	classifier := dbaasbase.BaseServiceClassifier(ctx)
	classifier["dbClassifier"] = "default"
	return classifier
}

func TenantClassifier(ctx context.Context) map[string]interface{} {
	classifier := dbaasbase.BaseTenantClassifier(ctx)
	classifier["dbClassifier"] = "default"
	return classifier
}

func (d *DbaaSMongoDbClient) buildServiceDbParams(params []model.DbParams) model.DbParams {
	localParams := model.DbParams{}
	if params != nil {
		localParams = params[0]
	}
	if localParams.Classifier == nil {
		localParams.Classifier = ServiceClassifier
	}
	return localParams
}

func (d *DbaaSMongoDbClient) buildTenantDbParams(params []model.DbParams) model.DbParams {
	localParams := model.DbParams{}
	if params != nil {
		localParams = params[0]
	}
	if localParams.Classifier == nil {
		localParams.Classifier = TenantClassifier
	}
	return localParams
}
