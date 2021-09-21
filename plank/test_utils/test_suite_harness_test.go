package test_utils

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/model"
	"github.com/vmware/transport-go/plank/pkg/server"
	"github.com/vmware/transport-go/service"
	"os"
	"testing"
)

func TestGetBasicTestServerConfig(t *testing.T) {
	config := GetBasicTestServerConfig("/", "stdout", "stdout", "stderr", 999, true)
	assert.Equal(t, "/", config.RootDir)
	assert.Equal(t, 999, config.Port)
}

// define mock integration suite
type testPlankTestIntegration struct {
	PlankIntegrationTestSuite
}

func (m *testPlankTestIntegration) SetPlatformServer(s server.PlatformServer) {
	m.PlatformServer = s
}
func (m *testPlankTestIntegration) SetSysChan(c chan os.Signal) {
	m.Syschan = c
}
func (m *testPlankTestIntegration) SetChannelManager(cm bus.ChannelManager) {
	m.ChannelManager = cm
}
func (m *testPlankTestIntegration) SetBus(eventBus bus.EventBus) {
	m.EventBus = eventBus
}

func TestSetupPlankTestSuiteForTest(t *testing.T) {
	b := bus.GetBus()
	cm := b.GetChannelManager()
	pit := &PlankIntegrationTestSuite{
		Suite:          suite.Suite{},
		PlatformServer: nil,
		Syschan:        make(chan os.Signal),
		ChannelManager: cm,
		EventBus:       b,
	}

	test := &testPlankTestIntegration{}
	SetupPlankTestSuiteForTest(pit, test)
	assert.Equal(t, cm, test.ChannelManager)
	assert.Equal(t, b, test.EventBus)
	assert.Nil(t, nil, test.PlatformServer)
}

type testService struct {
}

func (t *testService) HandleServiceRequest(rt *model.Request, c service.FabricServiceCore) {
}

func TestSetupPlankTestSuite(t *testing.T) {
	suite, err := SetupPlankTestSuite(&testService{}, "nowhere", 62986, nil)
	assert.NoError(t, err)
	assert.NotNil(t, suite)
}
