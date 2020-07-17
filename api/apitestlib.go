package api

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clear-ness/qa-discussion/app"
	"github.com/clear-ness/qa-discussion/config"
	"github.com/clear-ness/qa-discussion/model"
	"github.com/clear-ness/qa-discussion/store"
	"github.com/clear-ness/qa-discussion/testlib"

	"github.com/stretchr/testify/require"
)

type TestHelper struct {
	App         *app.App
	Server      *app.Server
	ConfigStore config.Store

	Client *model.Client

	SystemAdminUser *model.User
	BasicUser       *model.User
	BasicUser2      *model.User
}

var mainHelper *testlib.MainHelper

func (me *TestHelper) CreateClient() *model.Client {
	return model.NewAPIClient(fmt.Sprintf("http://localhost:%v", me.Server.ListenAddr.Port))
}

func setupTestHelper(dbStore store.Store) *TestHelper {
	var options []app.Option
	options = append(options, app.StoreOverride(dbStore))

	s, err := app.NewServer(options...)
	if err != nil {
		panic(err)
	}

	th := &TestHelper{
		App:    s.FakeApp(),
		Server: s,
	}

	if err := th.Server.Start(); err != nil {
		panic(err)
	}

	Init(th.Server.AppOptions, th.Server.Router)

	th.Client = th.CreateClient()

	// Verify handling of the supported true/false values by randomizing on each run.
	rand.Seed(time.Now().UTC().UnixNano())
	trueValues := []string{"1", "t", "T", "TRUE", "true", "True"}
	falseValues := []string{"0", "f", "F", "FALSE", "false", "False"}
	trueString := trueValues[rand.Intn(len(trueValues))]
	falseString := falseValues[rand.Intn(len(falseValues))]
	th.Client.SetBoolString(true, trueString)
	th.Client.SetBoolString(false, falseString)

	return th
}

func Setup(tb testing.TB) *TestHelper {
	if testing.Short() {
		tb.SkipNow()
	}

	if mainHelper == nil {
		tb.SkipNow()
	}

	dbStore := mainHelper.GetStore()
	// clear tables when every test func begins
	dbStore.DropAllTables()

	return setupTestHelper(dbStore)
}

func (me *TestHelper) waitForConnectivity() {
	for i := 0; i < 1000; i++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%v", me.Server.ListenAddr.Port))
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(time.Millisecond * 20)
	}
	panic("unable to connect")
}

var initBasicOnce sync.Once
var userCache struct {
	SystemAdminUser *model.User
	BasicUser       *model.User
	BasicUser2      *model.User
}

func (me *TestHelper) InitBasic() *TestHelper {
	me.waitForConnectivity()

	// create users once and cache them because password hashing is slow
	initBasicOnce.Do(func() {
		me.SystemAdminUser = me.CreateUser()
		me.SystemAdminUser, _ = me.App.GetUser(me.SystemAdminUser.Id)
		userCache.SystemAdminUser = me.SystemAdminUser.DeepCopy()

		me.BasicUser = me.CreateUser()
		me.BasicUser, _ = me.App.GetUser(me.BasicUser.Id)
		userCache.BasicUser = me.BasicUser.DeepCopy()

		me.BasicUser2 = me.CreateUser()
		me.BasicUser2, _ = me.App.GetUser(me.BasicUser2.Id)
		userCache.BasicUser2 = me.BasicUser2.DeepCopy()
	})

	me.SystemAdminUser = userCache.SystemAdminUser.DeepCopy()
	me.BasicUser = userCache.BasicUser.DeepCopy()
	me.BasicUser2 = userCache.BasicUser2.DeepCopy()

	mainHelper.GetSQLSupplier().GetMaster().Insert(me.SystemAdminUser, me.BasicUser, me.BasicUser2)
	me.SystemAdminUser.Password = "Password1@"
	me.BasicUser.Password = "Password1@"
	me.BasicUser2.Password = "Password1@"

	// login as basic usesr
	me.LoginBasic()

	return me
}

func (me *TestHelper) LoginBasic() {
	me.LoginBasicWithClient(me.Client)
}

func (me *TestHelper) LoginBasicWithClient(client *model.Client) {
	_, resp := client.Login(me.BasicUser.Email, me.BasicUser.Password)
	if resp.Error != nil {
		panic(resp.Error)
	}
}

func (me *TestHelper) LoginBasic2() {
	me.LoginBasic2WithClient(me.Client)
}

func (me *TestHelper) LoginBasic2WithClient(client *model.Client) {
	_, resp := client.Login(me.BasicUser2.Email, me.BasicUser2.Password)
	if resp.Error != nil {
		panic(resp.Error)
	}
}

func (me *TestHelper) TearDown() {
	me.ShutdownApp()
}

func (me *TestHelper) ShutdownApp() {
	done := make(chan bool)
	go func() {
		me.Server.Shutdown()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		// panic instead of fatal to terminate all tests in this package, otherwise the
		// still running App could spuriously fail subsequent tests.
		panic("failed to shutdown App within 30 seconds")
	}
}

func (me *TestHelper) GenerateTestEmail() string {
	return strings.ToLower(model.NewId() + "@localhost")
}

func GenerateTestUsername() string {
	return "fakeuser" + model.NewRandomString(10)
}

func (me *TestHelper) CreateUserWithClient(client *model.Client) *model.User {
	user := &model.User{
		Email:    me.GenerateTestEmail(),
		Username: GenerateTestUsername(),
		Password: "Password1@",
	}

	ruser, response := client.CreateUser(user)
	if response.Error != nil {
		panic(response.Error)
	}

	ruser.Password = "Password1@"
	_, err := me.Server.Store.User().VerifyEmail(ruser.Id, ruser.Email)
	if err != nil {
		return nil
	}
	return ruser
}

func (me *TestHelper) CreateUser() *model.User {
	return me.CreateUserWithClient(me.Client)
}

func checkHTTPStatus(t *testing.T, resp *model.Response, expectedStatus int, expectError bool) {
	t.Helper()

	require.NotNilf(t, resp, "Unexpected nil response, expected http:%v, expectError:%v", expectedStatus, expectError)
	if expectError {
		require.NotNil(t, resp.Error, "Expected a non-nil error and http status:%v, got nil, %v", expectedStatus, resp.StatusCode)
	} else {
		require.Nil(t, resp.Error, "Expected no error and http status:%v, got %q, http:%v", expectedStatus, resp.Error, resp.StatusCode)
	}
	require.Equalf(t, expectedStatus, resp.StatusCode, "Expected http status:%v, got %v (err: %q)", expectedStatus, resp.StatusCode, resp.Error)
}

func CheckOKStatus(t *testing.T, resp *model.Response) {
	t.Helper()
	checkHTTPStatus(t, resp, http.StatusOK, false)
}

func CheckNoError(t *testing.T, resp *model.Response) {
	t.Helper()
	require.Nil(t, resp.Error, "expected no error")
}

func CheckCreatedStatus(t *testing.T, resp *model.Response) {
	t.Helper()
	checkHTTPStatus(t, resp, http.StatusCreated, false)
}

func CheckUnauthorizedStatus(t *testing.T, resp *model.Response) {
	t.Helper()
	checkHTTPStatus(t, resp, http.StatusUnauthorized, true)
}

func CheckBadRequestStatus(t *testing.T, resp *model.Response) {
	t.Helper()
	checkHTTPStatus(t, resp, http.StatusBadRequest, true)
}

func CheckNotFoundStatus(t *testing.T, resp *model.Response) {
	t.Helper()
	checkHTTPStatus(t, resp, http.StatusNotFound, true)
}

func CheckForbiddenStatus(t *testing.T, resp *model.Response) {
	t.Helper()
	checkHTTPStatus(t, resp, http.StatusForbidden, true)
}
