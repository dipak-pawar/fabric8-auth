package controller_test

import (
	"testing"

	account "github.com/fabric8-services/fabric8-auth/account/repository"
	"github.com/fabric8-services/fabric8-auth/app/test"
	. "github.com/fabric8-services/fabric8-auth/controller"
	"github.com/fabric8-services/fabric8-auth/gormtestsupport"
	"github.com/fabric8-services/fabric8-auth/resource"
	testsupport "github.com/fabric8-services/fabric8-auth/test"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceREST struct {
	gormtestsupport.DBTestSuite
	resourceID   string
	permissionID string
	policyID     string
}

func TestRunSpaceREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestSpaceREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestSpaceREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.resourceID = uuid.NewV4().String()
	rest.permissionID = uuid.NewV4().String()
	rest.policyID = uuid.NewV4().String()
}

func (rest *TestSpaceREST) SecuredController() (*goa.Service, *SpaceController) {
	identity, err := testsupport.CreateTestIdentityAndUser(rest.DB, uuid.NewV4().String(), "KC")
	require.NoError(rest.T(), err)

	svc := testsupport.ServiceAsUser("Space-Service", identity)
	return svc, NewSpaceController(svc, rest.Application, rest.Configuration, &DummyResourceManager{
		ResourceID:   &rest.resourceID,
		PermissionID: &rest.permissionID,
		PolicyID:     &rest.policyID,
	})
}

func (rest *TestSpaceREST) SecuredControllerForIdentity(identity account.Identity) (*goa.Service, *SpaceController) {
	svc := testsupport.ServiceAsUser("Space-Service", identity)
	return svc, NewSpaceController(svc, rest.Application, rest.Configuration, nil)
}

func (rest *TestSpaceREST) UnSecuredController() (*goa.Service, *SpaceController) {
	svc := goa.New("Space-Service")
	return svc, NewSpaceController(svc, rest.Application, rest.Configuration, &DummyResourceManager{
		ResourceID:   &rest.resourceID,
		PermissionID: &rest.permissionID,
		PolicyID:     &rest.policyID,
	})
}

func (rest *TestSpaceREST) UnSecuredControllerWithDeprovisionedIdentity() (*goa.Service, *SpaceController) {
	identity, err := testsupport.CreateDeprovisionedTestIdentityAndUser(rest.DB, uuid.NewV4().String())
	require.NoError(rest.T(), err)

	svc := testsupport.ServiceAsUser("Space-Service", identity)
	return svc, NewSpaceController(svc, rest.Application, rest.Configuration, &DummyResourceManager{
		ResourceID:   &rest.resourceID,
		PermissionID: &rest.permissionID,
		PolicyID:     &rest.policyID,
	})
}

func (rest *TestSpaceREST) TestFailCreateSpaceUnauthorized() {
	// given
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.CreateSpaceUnauthorized(rest.T(), svc.Context, svc, ctrl, uuid.NewV4())
}

func (rest *TestSpaceREST) TestCreateSpaceUnauthorizedDeprovisionedUser() {
	// given
	svc, ctrl := rest.UnSecuredControllerWithDeprovisionedIdentity()
	// when/then
	test.CreateSpaceUnauthorized(rest.T(), svc.Context, svc, ctrl, uuid.NewV4())
}

func (rest *TestSpaceREST) TestCreateSpaceOK() {
	// given
	svc, ctrl := rest.SecuredController()
	// when
	_, created := test.CreateSpaceOK(rest.T(), svc.Context, svc, ctrl, uuid.NewV4())
	// then
	require.NotNil(rest.T(), created.Data)
	assert.Equal(rest.T(), rest.resourceID, created.Data.ResourceID)
	assert.Equal(rest.T(), rest.permissionID, created.Data.PermissionID)
	assert.Equal(rest.T(), rest.policyID, created.Data.PolicyID)
}

func (rest *TestSpaceREST) TestFailDeleteSpaceUnauthorized() {
	// given
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.DeleteSpaceUnauthorized(rest.T(), svc.Context, svc, ctrl, uuid.NewV4())
}

func (rest *TestSpaceREST) TestDeleteSpaceUnauthorizedDeprovisionedUser() {
	// given
	svc, ctrl := rest.UnSecuredControllerWithDeprovisionedIdentity()
	// when/then
	test.DeleteSpaceUnauthorized(rest.T(), svc.Context, svc, ctrl, uuid.NewV4())
}

func (rest *TestSpaceREST) TestDeleteSpaceOK() {
	// given
	svc, ctrl := rest.SecuredController()
	id := uuid.NewV4()
	// when
	test.CreateSpaceOK(rest.T(), svc.Context, svc, ctrl, id)
	// then
	test.DeleteSpaceOK(rest.T(), svc.Context, svc, ctrl, id)
}

func (rest *TestSpaceREST) TestDeleteSpaceIfUserIsNotSpaceOwnerForbidden() {
	// given
	svcOwner, ctrlOwner := rest.SecuredController()
	svcNotOwner, ctrlNotOwner := rest.SecuredController()
	id := uuid.NewV4()
	// when
	test.CreateSpaceOK(rest.T(), svcOwner.Context, svcOwner, ctrlOwner, id)
	// then
	test.DeleteSpaceForbidden(rest.T(), svcNotOwner.Context, svcNotOwner, ctrlNotOwner, id)
}

/*
* This test will attempt to list teams for a space
 */
func (rest *TestSpaceREST) TestListTeamOK() {
	g := rest.DBTestSuite.NewTestGraph()
	g.CreateTeam(g.ID("t1"), g.CreateSpace(g.ID("space")).
		AddAdmin(g.CreateUser(g.ID("admin"))).
		AddContributor(g.CreateUser(g.ID("contributor"))).
		AddViewer(g.CreateUser(g.ID("viewer"))))

	g.CreateTeam(g.ID("t2"), g.SpaceByID("space"))

	service, controller := rest.SecuredControllerForIdentity(*g.UserByID("admin").Identity())

	_, teams := test.ListTeamsSpaceOK(rest.T(), service.Context, service, controller, g.SpaceByID("space").SpaceID())

	require.Equal(rest.T(), 2, len(teams.Data))
	t1Found := false
	t2Found := false

	for i := range teams.Data {
		if teams.Data[i].ID == g.TeamByID("t1").TeamID().String() {
			t1Found = true
			require.Equal(rest.T(), g.TeamByID("t1").TeamName(), teams.Data[i].Name)
		} else if teams.Data[i].ID == g.TeamByID("t2").TeamID().String() {
			t2Found = true
			require.Equal(rest.T(), g.TeamByID("t2").TeamName(), teams.Data[i].Name)
		}
	}

	require.True(rest.T(), t1Found)
	require.True(rest.T(), t2Found)

	service, controller = rest.SecuredControllerForIdentity(*g.UserByID("contributor").Identity())
	_, teams = test.ListTeamsSpaceOK(rest.T(), service.Context, service, controller, g.SpaceByID("space").SpaceID())
	require.Equal(rest.T(), 2, len(teams.Data))

	service, controller = rest.SecuredControllerForIdentity(*g.UserByID("viewer").Identity())
	_, teams = test.ListTeamsSpaceOK(rest.T(), service.Context, service, controller, g.SpaceByID("space").SpaceID())
	require.Equal(rest.T(), 2, len(teams.Data))
}

func (rest *TestSpaceREST) TestListTeamUnauthorized() {
	g := rest.DBTestSuite.NewTestGraph()
	g.CreateTeam(g.ID("t1"), g.CreateSpace(g.ID("space")))
	g.CreateTeam(g.ID("t2"), g.SpaceByID("space"))

	service, controller := rest.SecuredControllerForIdentity(*g.CreateUser().Identity())
	test.ListTeamsSpaceForbidden(rest.T(), service.Context, service, controller, g.SpaceByID("space").SpaceID())

	service, controller = rest.UnSecuredController()
	test.ListTeamsSpaceUnauthorized(rest.T(), service.Context, service, controller, g.SpaceByID("space").SpaceID())
}
