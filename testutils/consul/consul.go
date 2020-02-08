package consul

import (
	"fmt"
	"github.com/hashicorp/consul/agent"
	"github.com/hashicorp/consul/agent/config"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

type DevServer struct {
	ports        config.Ports
	t            *testing.T
	agent        *agent.Agent
	stderr       io.Writer
	client       *api.Client
	consulConfig *api.Config
}

func NewDevServer(t *testing.T) *DevServer {
	ports := config.Ports{
		DNS:     getOneFreePort(t),
		HTTP:    getOneFreePort(t),
		SerfLAN: getOneFreePort(t),
		SerfWAN: getOneFreePort(t),
		Server:  getOneFreePort(t),
		GRPC:    getOneFreePort(t),
	}
	return &DevServer{t: t, ports: ports, stderr: os.Stderr}
}

func getOneFreePort(t *testing.T) *int {
	port, err := getFreePort()
	if err != nil {
		assert.Failf(t, "failed to get free port: %s", err.Error())
		return nil
	}
	return &port
}

func (d *DevServer) Start() {
	devMode := true
	builder, err := config.NewBuilder(config.Flags{
		Config:  config.Config{Ports: d.ports},
		DevMode: &devMode,
	})
	if err != nil {
		assert.Failf(d.t, "failed to build config: %s", err.Error())
		return
	}

	cfg, err := builder.BuildAndValidate()
	if err != nil {
		assert.Failf(d.t, "failed to build and validate: %s", err.Error())
		return
	}

	logger := log.New(d.stderr, d.t.Name(), log.LstdFlags)

	devAgent, err := agent.New(&cfg, logger)
	if err != nil {
		assert.Failf(d.t, "failed to create agent: %s", err.Error())
		return
	}

	err = devAgent.Start()
	if err != nil {
		assert.Failf(d.t, "failed to start agent: %s", err.Error())
		return
	}

	defaultConfig := api.DefaultConfig()
	defaultConfig.Address = fmt.Sprintf("http://localhost:%d", d.HTTPPort())
	client, err := api.NewClient(defaultConfig)
	if err != nil {
		assert.Failf(d.t, "failed to create client for consul: %s", err.Error())
		return
	}
	d.client = client
	d.consulConfig = defaultConfig

	status := client.Status()
	waitIndex := 0
	const limit = 100
	for ; waitIndex < limit; waitIndex++ {
		logger.Printf("Checking for agent serving on localhost:%d to be up", d.HTTPPort())
		leader, err := status.Leader()
		if err != nil || leader == "" {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		break
	}
	if waitIndex == limit {
		assert.Fail(d.t, "failed to start consul agent")
		return
	}
	d.agent = devAgent
}

func (d *DevServer) ConsulConfig() *api.Config {
	return d.consulConfig
}

func (d *DevServer) HTTPPort() int {
	return *d.ports.HTTP
}

func (d *DevServer) Stop() {
	if d.agent != nil {
		d.agent.ShutdownEndpoints()
		if err := d.agent.ShutdownAgent(); err != nil {
			assert.Failf(d.t, "failed to stop agent: %s", err.Error())
		}
	}
}

func (d *DevServer) Put(basePath, key string, value []byte) error {
	key = path.Join(basePath, key)
	kvPair := &api.KVPair{
		Key:   key,
		Value: value,
	}
	_, err := d.client.KV().Put(kvPair, nil)
	if err != nil {
		return fmt.Errorf("error creating/updating key %s: %v", key, err)
	}
	return nil
}

func (d *DevServer) Delete(basePath, key string) error {
	key = path.Join(basePath, key)
	_, err := d.client.KV().Delete(key, nil)
	if err != nil {
		return fmt.Errorf("error deleting key %s: %v", key, err)
	}
	return nil
}

func (d *DevServer) Get(basePath, key string) ([]byte, error) {
	key = path.Join(basePath, key)

	kvPair, _, err := d.client.KV().Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting %s: %v", key, err)
	}
	return kvPair.Value, nil
}

func (d *DevServer) List(basePath string) ([]string, error) {
	kvPairs, _, err := d.client.KV().List(basePath, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting keys from consul: %v", err)
	}

	result := make([]string, 0, len(kvPairs))
	for _, pair := range kvPairs {
		result = append(result, strings.TrimPrefix(pair.Key, basePath))
	}
	return result, nil
}

// getFreePort returns a free available port from the system
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
