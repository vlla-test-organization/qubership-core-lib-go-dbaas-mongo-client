[![Go build](https://github.com/Netcracker/qubership-core-lib-go-dbaas-mongo-client/actions/workflows/go-build.yml/badge.svg)](https://github.com/Netcracker/qubership-core-lib-go-dbaas-mongo-client/actions/workflows/go-build.yml)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?metric=coverage&project=Netcracker_qubership-core-lib-go-dbaas-mongo-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-dbaas-mongo-client)
[![duplicated_lines_density](https://sonarcloud.io/api/project_badges/measure?metric=duplicated_lines_density&project=Netcracker_qubership-core-lib-go-dbaas-mongo-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-dbaas-mongo-client)
[![vulnerabilities](https://sonarcloud.io/api/project_badges/measure?metric=vulnerabilities&project=Netcracker_qubership-core-lib-go-dbaas-mongo-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-dbaas-mongo-client)
[![bugs](https://sonarcloud.io/api/project_badges/measure?metric=bugs&project=Netcracker_qubership-core-lib-go-dbaas-mongo-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-dbaas-mongo-client)
[![code_smells](https://sonarcloud.io/api/project_badges/measure?metric=code_smells&project=Netcracker_qubership-core-lib-go-dbaas-mongo-client)](https://sonarcloud.io/summary/overall?id=Netcracker_qubership-core-lib-go-dbaas-mongo-client)

# Mongo dbaas go client

This module provides convenient way of interaction with **mongo** databases provided by dbaas-aggregator.
`Mongo dbaas go client` supports _multi-tenancy_ and can work with both _service_ and _tenant_ databases.

> **NOTE** If you want to migrate your service from go-microservice-core to new mongo-client please check our
> [migration guide](/docs/mongo-client-migration-guide.md)

- [Install](#install)
- [Usage](#usage)
    * [Get connection properties for existing database or create new one](#get-connection-for-existing-database-or-create-new-one)
    * [Find connection properties for existing database](#find-connection-for-existing-database)
    * [MongoDbClient](#mongodbclient)
    * [Mongo multiusers](#mongo-multiusers)
- [Classifier](#classifier)
- [SSL/TLS support](#ssltls-support)
- [Quick example](#quick-example)

## Install

To get `mongo dbaas client` use
```go
 go get github.com/netcracker/qubership-core-lib-go-dbaas-mongo-client@<latest released version>
```

List of all released versions may be found [here](https://github.com/netcracker/qubership-core-lib-go-dbaas-mongo-client/-/tags)

## Usage

At first, it's necessary to register security implemention - dummy or your own, the followning example shows registration of required services:
```go
import (
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/netcracker/qubership-core-lib-go/v3/security"
)

func init() {
	serviceloader.Register(1, &security.DummyToken{})
}
```

Then the user should create `DbaaSMongoDbClient`. This is a base client, which allows working with tenant and service databases.
To create instance of `DbaaSMongoDbClient` use `NewClient(pool *dbaasbase.DbaaSPool) *DbaaSMongoDbClient`.

Note that client has parameter _pool_. `dbaasbase.DbaaSPool` is a tool which stores all cached connections and
create new ones. To find more info visit [dbaasbase](https://github.com/netcracker/qubership-core-lib-go-dbaas-base-client/blob/main/README.md)

Example of client creation:
```go
pool := dbaasbase.NewDbaasPool()
client := mongodbaas.NewClient(pool)
```

_Note_:By default, `Mongo dbaas go client` supports dbaas-aggregator as databases source. But there is a possibility for user to provide another
sources (for example, zookeeper). To do so use [LogcalDbProvider](https://github.com/netcracker/qubership-core-lib-go-dbaas-base-client/blob/main/README.md#logicaldbproviders)
from dbaasbase.

Next step is to create `Database` object. `Databse` is not a mongo.Database instance. It just an interface which allows
creating mongoClient or getting connection properties from dbaas. At this step user may choose which type of database he will
work with:  `service` or `tenant`.

* To work with service databases use `ServiceDatabase(params ...model.DbParams) Database`
* To work with tenant databases use `TenantDatabase(params ...model.DbParams) Database`

Each func has `DbParams` as parameter.

DbParams store information for database creation. Note that this parameter is optional, but if user doesn't pass Classifier,
default one will be used. More about classifiers [here](#classifier)

| Name         | Description                                                                                    | type                                                                                                                                                                      |
|--------------|------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Classifier   | function which builds classifier from context. Classifier should be unique for each mongo db.  | func(ctx context.Context) map[string]interface{}                                                                                                                          |
| BaseDbParams | Specific parameters for database creation and getting connection.                              | [DbCreationParams](https://github.com/netcracker/qubership-core-lib-go-dbaas-base-client/blob/main#basedbparams)   |

Example how to create an instance of Database.
```go
 dbPool := dbaasbase.NewDbaasPool()
 client := mongodbaas.NewClient(dbPool)
 serviceDB := client.ServiceDatabase() // service Database creation 
 tenantDB := client.TenantDatabase() // tenant Database creation 
```

`Database` allows:   
* get connection for existing database or create new one;
* find connection for existing database (don't create db just return connection properties)
* get MongoDbClient, through which you can create mongo db and get `*mongo.Database` for database operation. 
`serviceDB` and `tenantDB`  instances should be singleton and it's enough to create them only once.

### Get connection for existing database or create new one

Func `GetConnectionProperties(ctx context.Context) (*DbaaSMongoDbConnection, error)`
at first will check if the desired database with _mongodb_ type and classifier exists. If it exists, function will just return
connection properties in the form of [MongoConnProperties](model/mongo_connection_properies.go).
If database with _mongo_ type and classifier doesn't exist, such database will be created and function will return
connection properties for a new created database.

_Parameters:_
* ctx - context, enriched with some headers. (See docs about context-propagation [here](https://github.com/netcracker/qubership-core-lib-go/blob/main/context-propagation/README.md)). Context object can have request scope values from which can be used to build classifier, for example tenantId.

```go
    ctx := ctxmanager.InitContext(context.Background(), propagateHeaders()) // preferred way
    // ctx := context.Background() // also possible for service client, but not recommended
    MongoDbConnectionProps, err := servcieDB.GetConnectionProperties(ctx) // no params because they are optional
```

### Find connection for existing database

Func `FindConnectionProperties(ctx context.Context) (*DbaaSMongoDbConnection, error)`
returns connection properties in the form of [MongoConnProperties](model/mongo_connection_properies.go). Unlike `GetConnectionProperties`
this function won't create database if it doesn't exist and just return nil value.

_Parameters:_
* ctx - context, enriched with some headers. (See docs about context-propagation [here](https://github.com/netcracker/qubership-core-lib-go/blob/main/context-propagation/README.md)). Context object can have request scope values from which can be used to build classifier, for example tenantId.

```go
    ctx := ctxmanager.InitContext(context.Background(), propagateHeaders()) // preferred way
    // ctx := context.Background() // also possible for service client, but not recommended
    dbMongoConnectionProps, err := client.FindConnectionProperties(ctx)
```

### MongoDbClient

MongoDbClient is a special object, which allows getting `mongo.Database` to establish connection and to operate with a database. 
`MongoDbClient` is a singleton and should be created only once.

MongoDbClient has method `GetConnection(ctx context.Context) (*mongo.Database, error)` which will return `*mongo.Database` to work with the database.
We strongly recommend not to store `mongo.Database` as singleton and get new connection for every block of code.
This is because the password in the database may changed (by dbaas or someone else) and then the connection will return an error. Every time the function
`mongoDbClient.GetConnection(ctx)`is called, the password lifetime and correctness is checked. If necessary, the password is updated.

_Note_: classifier will be created with context and function from DbParams.

To create mongoDbClient use `GetMongoDbClient(options ...*mongo.ClientOptions) (*MongoDbClient, error)`

Parameters:
* options _optional_ - user may pass desired mongo.ClientOptions or don't pass anything at all. Note that user **doesn't have to 
set connection parameters** with options, because these parameters will be received from dbaas-aggregator.

```go
    ctx := ctxmanager.InitContext(context.Background(), propagateHeaders()) // preferred way
    // ctx := context.Background() // also possible for service client, but not recommended
    mgOpts := options.Client().SetMaxPoolSize(7)
    mongoClient, err := database.GetMongoDbClient(options) // with options
    mgDB, err := mongoClient.GetMongoDatabase(ctx)
    
    collection := mgDB.Collection("qux")
    res, err := collection.InsertOne(context.Background(), bson.M{"hello": "world"})
    if err != nil { return err }
```
### Mongo multiusers
For specifying connection properties user role you should add this role in BaseDbParams structure:

```go
params := model.DbParams{
        Classifier:   Classifier, //database classifier
        BaseDbParams: rest.BaseDbParams{Role: "admin"}, //for example "admin", "rw", "ro"
    }
dbPool := dbaasbase.NewDbaaSPool()
mongoClient := mongodbaas.NewClient(dbPool)
servicetDb := mongoClient.ServiceDatabase(params) //or for tenant database - TenantDatabase(params)
mgClient, err := serviceDb.GetMongoDbClient()
mgDb, err := mgClient.GetMongoDatabase(ctx)
```
Requests to DbaaS will contain the role you specify in this structure.

## Classifier

Classifier and dbType should be unique combination for each database. Fields "tenantId" or "scope" must be into users' custom classifiers.

User can use default service or tenant classifier. It will be used if user doesn't specify Classifier in DbParams. 
This is recommended approach and and we don't recommend using custom classifier because it can lead to some problems. 
Use can be reasonable if you migrate to this module and before used custom and not default classifier.


Default service classifier looks like:
```json
{
    "dbClassifier": "default",
    "scope": "service",
    "microserviceName": "<ms-name>"
}
```

Default tenant classifier looks like

```json
{
  "scope": "tenant",
  "dbClassifier": "default",
  "tenantId": "<tenant-external-id>",
  "microserviceName": "<ms-name>"
}
```
Note, that if user doesn't set `MICROSERVICE_NAME` (or `microservice.name`) property, there will be panic during default classifier creation.
Also, if there are no tenantId in tenantContext, **panic will be thrown**.

## SSL/TLS support

This library supports work with secured connections to mongodb. Connection will be secured if TLS mode is enabled in
mongodb-adapter.

For correct work with secured connections, the library requires having a truststore with certificate.
It may be public cloud certificate, cert-manager's certificate or any type of certificates related to database.
We do not recommend use self-signed certificates. Instead, use default NC-CA.

To start using TLS feature user has to enable it on the physical database (adapter's) side and add certificate to service truststore.

#### Physical database switching
To enable TLS support in physical database redeploy mongodb with mandatory parameters
```yaml
tls.mode=requireTLS;
```

In case of using cert-manager as certificates source add extra parameters
```yaml
tls.generateCerts.enabled=true;
tls.generateCerts.clusterIssuerName=<cluster issuer name>
```

ClusterIssuerName identifies which Certificate Authority cert-manager will use to issue a certificate.
It can be obtained from the person in charge of the cert-manager on the environment.

#### Add certificate to service truststore

The platform deployer provides the bulk uploading of certificates to truststores.

In order to add required certificates to services truststore:
1. Check and get certificate which is used in mongodb.
   * In most cases certificate is located in `Secrets` -> `root-ca` -> `ca.crt`
2. Create ticket to `PSUPCDO/Configuration` and ask DevOps team to add this certificate to your deployer job.
3. After that all new deployments via configured deployer will include new certificate. Deployer creates a secret with certificate.
   Make sure the certificate is mount into your microservice.
   On bootstrapping microservice there is generated truststore with default location and password.

## Quick example

Here we create mongo tenant client, then get MongoDbClient and execute a query.

application.yaml
```yaml
  microservice.name=tenant-manager
```

```go
package main

import (
	"context"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/netcracker/qubership-core-lib-go/v3/context-propagation/ctxmanager"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
    dbaasbase "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3"
    "github.com/netcracker/qubership-core-lib-go-dbaas-base-client/v3/model/rest"
	mongodbaas "github.com/netcracker/qubership-core-lib-go-dbaas-mongo-client"
	"github.com/netcracker/qubership-core-lib-go/v3/context-propagation/baseproviders/tenant"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var logger logging.Logger

func init() {
	configloader.Init(configloader.BasePropertySources())
	logger = logging.GetLogger("main")
	ctxmanager.Register([]ctxmanager.ContextProvider{tenant.TenantProvider{}})
}

func main() {
	
	// some context initialization
	ctx := ctxmanager.InitContext(context.Background(), map[string]interface{}{tenant.TenantContextName: "123"})
	
	// mongo service client creation
	dbPool := dbaasbase.NewDbaaSPool()
	mongoClient := mongodbaas.NewClient(dbPool)
	tenantDb := mongoClient.TenantDatabase()

	// create mongoDbClient
	mgClient, err := tenantDb.GetMongoDbClient() // singleton for tenant db. This object must be used to get connection in the entire application.
	mgDb, err := mgClient.GetMongoDatabase(ctx) // now we can receive mongo.Database
	if err != nil {
		logger.Error("Error during mongo.DatabaseCreation")
	}
	logger.Info("Got such db %+v", mgDb)

    collection := mgDb.Collection("podcasts")
    _, errInsert := collection.InsertOne(ctx, bson.D{
    	{Key: "title", Value: title},
    })
    if errInsert != nil {
      logger.Error("Error during insert into collection")
    }
  logger.Info("Value was inserted")
    result, error := getAllPodcasts(mgClient, ctx)
}

func getAllPodcasts(client *mongodbaas.MongoDbClient, ctx context.Context) ([]bson.M, error) {
	mgDb, err := client.GetMongoDatabase(ctx)  
	collection := mgDb.Collection("podcasts")
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
      return nil, err
	}
	var episodes []bson.M
	if err = cursor.All(ctx, &episodes); err != nil {
      return nil, err
	}
	return episodes, nil
}

```
