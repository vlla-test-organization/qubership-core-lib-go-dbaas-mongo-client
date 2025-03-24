package mongodbaas

import (
	"context"

	"github.com/netcracker/qubership-core-lib-go/v3/utils"
	dbaasbase "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3"
	"github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/cache"
	dbaasbasemodel "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/model"
	"github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/model/rest"
	"github.com/netcracker/qubership-core-lib-go-dbaas-mongo-client/v3/model"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

type MongoDbClient interface {
	GetMongoDatabase(ctx context.Context) (*mongo.Database, error)
}

type MongoDbClientImpl struct {
	options      *options.ClientOptions
	dbaasClient  dbaasbase.DbaaSClient
	mongodbCache *cache.DbaaSCache
	params       model.DbParams
}

type cachedMongoDatabase struct {
	mongoDb *mongo.Database
	auth    *options.Credential
}

func (m *MongoDbClientImpl) GetMongoDatabase(ctx context.Context) (*mongo.Database, error) {
	classifier := m.params.Classifier(ctx)
	key := cache.NewKey(DB_TYPE, classifier)

	rawCachedMongoDb, err := m.mongodbCache.Cache(key, m.createNewMongoDb(ctx, classifier))
	if err != nil {
		return nil, err
	}
	cachedMongoDb := rawCachedMongoDb.(cachedMongoDatabase)

	if !m.isPasswordValid(cachedMongoDb, ctx) {
		logger.Info("authentication error, try to get new password")
		connection, err := m.getNewConnectionProperties(ctx, classifier, m.params.BaseDbParams)
		if err != nil {
			return nil, err
		}
		err = m.updatePassword(ctx, &cachedMongoDb, connection)
		if err != nil {
			return nil, err
		}
		return cachedMongoDb.mongoDb, nil
	}
	return cachedMongoDb.mongoDb, nil
}

func (m *MongoDbClientImpl) createNewMongoDb(ctx context.Context, classifier map[string]interface{}) func() (interface{}, error) {
	return func() (interface{}, error) {
		logger.Debug("Create mongo database with classifier %+v", classifier)
		logicalDb, err := m.dbaasClient.GetOrCreateDb(ctx, DB_TYPE, classifier, m.params.BaseDbParams)
		if err != nil {
			return nil, err
		}
		clientOptions, err := buildMongoOptions(m.options, *logicalDb)
		if err != nil {
			logger.ErrorC(ctx, "Unable to build client clientOptions, error %+v", err)
			return nil, err
		}
		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			logger.ErrorC(ctx, "Unable to connect to mongo client, error %+v", err)
			return nil, err
		}
		cachedMongoDb := cachedMongoDatabase{
			mongoDb: client.Database(getDbName(logicalDb.ConnectionProperties)),
			auth:    clientOptions.Auth,
		}
		logger.Debugf("Build go-mongo client for database with classifier %+v", classifier)
		return cachedMongoDb, nil
	}
}

func (m *MongoDbClientImpl) updatePassword(ctx context.Context, cachedMongoDb *cachedMongoDatabase, connectionProperties *model.MongoConnProperties) error {
	err := cachedMongoDb.mongoDb.Client().Disconnect(ctx)
	if err != nil {
		logger.ErrorC(ctx, "Couldn't disconnect from existing mongoClient")
		return err
	}
	cachedMongoDb.auth.Password = connectionProperties.Password
	cachedMongoDb.auth.AuthSource = connectionProperties.AuthDbName
	newClient, err := mongo.Connect(ctx, m.options)
	if err != nil {
		logger.ErrorC(ctx, "Couldn't connect to new mongoClient")
		return err
	}
	cachedMongoDb.mongoDb = newClient.Database(connectionProperties.AuthDbName)
	return nil
}

func (m *MongoDbClientImpl) getNewConnectionProperties(ctx context.Context, classifier map[string]interface{}, params rest.BaseDbParams) (*model.MongoConnProperties, error) {
	newConnection, dbErr := m.dbaasClient.GetConnection(ctx, DB_TYPE, classifier, params)
	if dbErr != nil {
		logger.ErrorC(ctx, "Can't update connection with dbaas")
		return nil, dbErr
	}
	connectionProperties := toMongoConnProperties(newConnection)
	return &connectionProperties, nil
}

func (m *MongoDbClientImpl) isPasswordValid(db cachedMongoDatabase, ctx context.Context) bool {
	if err := db.mongoDb.Client().Ping(ctx, readpref.Primary()); err != nil {
		_, ok := err.(topology.ConnectionError).Unwrap().(*auth.Error)
		return !ok
	}
	return true
}

func buildMongoOptions(userOptions *options.ClientOptions, logicalDb dbaasbasemodel.LogicalDb) (*options.ClientOptions, error) {
	mongoConnProperties := toMongoConnProperties(logicalDb.ConnectionProperties)
	opts := new(options.ClientOptions)
	if userOptions != nil {
		opts = userOptions
	}
	if ok, tls := logicalDb.ConnectionProperties["tls"].(bool); ok && tls {
		logger.Infof("Connection to mongodb will be secured")
		opts.TLSConfig = utils.GetTlsConfig()
	}
	connString, err := connstring.ParseAndValidate(mongoConnProperties.Url)
	if err != nil {
		return nil, err
	}
	authDbName := connString.Database
	if authDbName == "" {
		authDbName = mongoConnProperties.AuthDbName
	}
	opts.ApplyURI(mongoConnProperties.Url).
		SetAuth(options.Credential{
			Username:      mongoConnProperties.Username,
			Password:      mongoConnProperties.Password,
			AuthSource:    authDbName,
			AuthMechanism: "SCRAM-SHA-1",
		})
	return opts, nil
}
