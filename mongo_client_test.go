package mongodbaas

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	dbaasbase "github.com/vlla-test-organization/qubership-core-lib-go-dbaas-base-client/v3"
	"github.com/vlla-test-organization/qubership-core-lib-go-dbaas-base-client/v3/model/rest"
	"github.com/vlla-test-organization/qubership-core-lib-go-dbaas-mongo-client/v3/model"
	"github.com/vlla-test-organization/qubership-core-lib-go/v3/configloader"
	"github.com/vlla-test-organization/qubership-core-lib-go/v3/context-propagation/baseproviders/tenant"
	"github.com/vlla-test-organization/qubership-core-lib-go/v3/context-propagation/ctxmanager"
	"github.com/vlla-test-organization/qubership-core-lib-go/v3/security"
	"github.com/vlla-test-organization/qubership-core-lib-go/v3/serviceloader"
)

func init() {
	ctxmanager.Register([]ctxmanager.ContextProvider{tenant.TenantProvider{}})
	serviceloader.Register(1, &security.DummyToken{})
}

func setup() {
	os.Setenv(propMicroserviceName, "test_service")
	os.Setenv(namespaceEnvName, "test_space")
	configloader.Init(configloader.EnvPropertySource())
}

func tearDown() {
	os.Unsetenv(propMicroserviceName)
	os.Unsetenv(namespaceEnvName)
}

func TestNewServiceDbaasClient_WithoutParamsV3(t *testing.T) {
	setup()
	defer tearDown()
	dbaasPool := dbaasbase.NewDbaaSPool()
	commonClient := NewClient(dbaasPool)
	serviceDB := commonClient.ServiceDatabase()
	assert.NotNil(t, serviceDB)
	db := serviceDB.(*mongoDatabase)
	ctx := context.Background()
	assert.Equal(t, ServiceClassifier(ctx), db.params.Classifier(ctx))
}

func TestNewServiceDbaasClient_WithParamsV3(t *testing.T) {
	setup()
	defer tearDown()
	dbaasPool := dbaasbase.NewDbaaSPool()
	commonClient := NewClient(dbaasPool)
	params := model.DbParams{
		Classifier:   stubClassifierV3,
		BaseDbParams: rest.BaseDbParams{Role: "admin"},
	}
	serviceDB := commonClient.ServiceDatabase(params)
	assert.NotNil(t, serviceDB)
	db := serviceDB.(*mongoDatabase)
	ctx := context.Background()
	assert.Equal(t, stubClassifierV3(ctx), db.params.Classifier(ctx))
	assert.Equal(t, "admin", db.params.BaseDbParams.Role)
}

func TestNewTenantDbaasClient_WithoutParamsV3(t *testing.T) {
	setup()
	defer tearDown()
	dbaasPool := dbaasbase.NewDbaaSPool()
	commonClient := NewClient(dbaasPool)
	tenantDb := commonClient.TenantDatabase()
	assert.NotNil(t, tenantDb)
	db := tenantDb.(*mongoDatabase)
	ctx := createTenantContext()
	assert.Equal(t, TenantClassifier(ctx), db.params.Classifier(ctx))
}

func TestNewTenantDbaasClient_WithParamsV3(t *testing.T) {
	setup()

	defer tearDown()
	dbaasPool := dbaasbase.NewDbaaSPool()
	commonClient := NewClient(dbaasPool)
	params := model.DbParams{
		Classifier:   stubClassifierV3,
		BaseDbParams: rest.BaseDbParams{Role: "admin"},
	}
	tenantDb := commonClient.TenantDatabase(params)
	assert.NotNil(t, tenantDb)
	db := tenantDb.(*mongoDatabase)
	ctx := context.Background()
	assert.Equal(t, stubClassifierV3(ctx), db.params.Classifier(ctx))
	assert.Equal(t, "admin", db.params.BaseDbParams.Role)
}

func TestCreateServiceClassifierV3(t *testing.T) {
	setup()
	defer tearDown()
	expected := map[string]interface{}{
		"microserviceName": "test_service",
		"dbClassifier":     "default",
		"namespace":        "test_space",
		"scope":            "service",
	}
	actual := ServiceClassifier(context.Background())
	assert.Equal(t, expected, actual)
}

func TestCreateTenantClassifierV3(t *testing.T) {
	setup()
	defer tearDown()
	ctx := createTenantContext()
	expected := map[string]interface{}{
		"microserviceName": "test_service",
		"dbClassifier":     "default",
		"namespace":        "test_space",
		"tenantId":         "123",
		"scope":            "tenant",
	}
	actual := TenantClassifier(ctx)
	assert.Equal(t, expected, actual)
}

func TestCreateTenantClassifier_WithoutTenantIdV3(t *testing.T) {
	setup()
	defer tearDown()
	ctx := context.Background()

	assert.Panics(t, func() {
		TenantClassifier(ctx)
	})
}

func stubClassifierV3(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"scope":            "service",
		"microserviceName": "service_test",
	}
}

func createTenantContext() context.Context {
	incomingHeaders := map[string]interface{}{tenant.TenantHeader: "123"}
	return ctxmanager.InitContext(context.Background(), incomingHeaders)
}
