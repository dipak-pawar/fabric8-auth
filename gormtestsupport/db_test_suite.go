package gormtestsupport

import (
	"context"
	"os"
	"testing"

	"github.com/fabric8-services/fabric8-auth/application"
	"github.com/fabric8-services/fabric8-auth/application/factory/wrapper"
	config "github.com/fabric8-services/fabric8-auth/configuration"
	"github.com/fabric8-services/fabric8-auth/gormapplication"
	"github.com/fabric8-services/fabric8-auth/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-auth/log"
	"github.com/fabric8-services/fabric8-auth/migration"
	"github.com/fabric8-services/fabric8-auth/resource"

	"github.com/fabric8-services/fabric8-auth/test/graph"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // need to import postgres driver
	"github.com/stretchr/testify/suite"
)

var _ suite.SetupAllSuite = &DBTestSuite{}
var _ suite.TearDownAllSuite = &DBTestSuite{}

// NewDBTestSuite instantiates a new DBTestSuite
func NewDBTestSuite() DBTestSuite {
	return DBTestSuite{}
}

// DBTestSuite is a base for tests using a gorm db
type DBTestSuite struct {
	suite.Suite
	Configuration *config.ConfigurationData
	DB            *gorm.DB
	Application   application.Application
	CleanTest     func()
	CleanSuite    func()
	Ctx           context.Context
	Graph         *graph.TestGraph
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.Database)
	configuration, err := config.GetConfigurationData()
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the configuration")
	}
	s.Configuration = configuration
	if _, c := os.LookupEnv(resource.Database); c != false {
		s.DB, err = gorm.Open("postgres", s.Configuration.GetPostgresConfigString())
		if err != nil {
			log.Panic(nil, map[string]interface{}{
				"err":             err,
				"postgres_config": configuration.GetPostgresConfigString(),
			}, "failed to connect to the database")
		}
	}
	// configures the log mode for the SQL queries (by default, disabled)
	s.DB.LogMode(s.Configuration.IsDBLogsEnabled())
	s.Application = gormapplication.NewGormDB(s.DB, configuration)
	s.Ctx = migration.NewMigrationContext(context.Background())
	s.PopulateDBTestSuite(s.Ctx)
	s.CleanSuite = cleaner.DeleteCreatedEntities(s.DB)
}

// SetupTest implements suite.SetupTest
func (s *DBTestSuite) SetupTest() {
	s.CleanTest = cleaner.DeleteCreatedEntities(s.DB)
	g := s.NewTestGraph(s.T())
	s.Graph = &g
	s.ResetFactories()
}

// TearDownTest implements suite.TearDownTest
func (s *DBTestSuite) TearDownTest() {
	// in some cases, we might need to keep the test data in the DB for inspecting/reproducing
	// the SQL queries. In that case, the `AUTH_CLEAN_TEST_DATA` env variable should be set to `false`.
	// By default, test data will be removed from the DB after each test
	if s.Configuration.IsCleanTestDataEnabled() {
		s.CleanTest()
	}
	s.Graph = nil
}

// PopulateDBTestSuite populates the DB with common values
func (s *DBTestSuite) PopulateDBTestSuite(ctx context.Context) {
}

// TearDownSuite implements suite.TearDownAllSuite
func (s *DBTestSuite) TearDownSuite() {
	// in some cases, we might need to keep the test data in the DB for inspecting/reproducing
	// the SQL queries. In that case, the `AUTH_CLEAN_TEST_DATA` env variable should be set to `false`.
	// By default, test data will be removed from the DB after each test
	if s.Configuration.IsCleanTestDataEnabled() {
		s.CleanSuite()
	}
	s.DB.Close()
}

// DisableGormCallbacks will turn off gorm's automatic setting of `created_at`
// and `updated_at` columns. Call this function and make sure to `defer` the
// returned function.
//
//    resetFn := DisableGormCallbacks()
//    defer resetFn()
func (s *DBTestSuite) DisableGormCallbacks() func() {
	gormCallbackName := "gorm:update_time_stamp"
	// remember old callbacks
	oldCreateCallback := s.DB.Callback().Create().Get(gormCallbackName)
	oldUpdateCallback := s.DB.Callback().Update().Get(gormCallbackName)
	// remove current callbacks
	s.DB.Callback().Create().Remove(gormCallbackName)
	s.DB.Callback().Update().Remove(gormCallbackName)
	// return a function to restore old callbacks
	return func() {
		s.DB.Callback().Create().Register(gormCallbackName, oldCreateCallback)
		s.DB.Callback().Update().Register(gormCallbackName, oldUpdateCallback)
	}
}

func (s *DBTestSuite) NewTestGraph(t *testing.T) graph.TestGraph {
	return graph.NewTestGraph(t, s.Application, s.Ctx, s.DB)
}

// ReplaceFactory replaces a default factory with the specified factory.  This function is recommended to be used
// during tests where the default behaviour of a factory needs to be overridden
func (s *DBTestSuite) WrapFactory(identifier string, constructor wrapper.FactoryWrapperConstructor, initializer wrapper.FactoryWrapperInitializer) {
	s.Application.(*gormapplication.GormDB).WrapFactory(identifier, constructor, initializer)
}

// ResetFactories resets all factories to default, and resets all overridden factory configurations.
func (s *DBTestSuite) ResetFactories() {
	s.Application.(*gormapplication.GormDB).ResetFactories()
}
