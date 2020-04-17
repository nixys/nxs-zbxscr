package zbxscr

import (
	"fmt"
	"testing"

	"github.com/nixys/nxs-go-conf"
	"github.com/tidwall/gjson"
)

const (
	confPathDefault = "nxs-zbxscr_test.conf"
)

type Instance struct {
	Name string `conf:"name" conf_extraopts:"required"`
	URL  string `conf:"url" conf_extraopts:"required"`
}

type confOpts struct {
	CacheRoot string     `conf:"cache_root" conf_extraopts:"default=/tmp/nxs-zbxscr_test"`
	CacheTTL  float64    `conf:"cache_ttl" conf_extraopts:"default=30"`
	User      string     `conf:"user" conf_extraopts:"default=zabbix"`
	Group     string     `conf:"group" conf_extraopts:"default=zabbix"`
	Instances []Instance `conf:"instances"`
}

type appContext struct {
	instance string
	metric   string
	conf     contextConf
}

type contextConf struct {
	opts confOpts
	err  error
}

type testDiscovery struct {
	Name string `json:"{#NAME}"`
	URL  string `json:"{#URL}"`
}

func confRead(confPath string) (confOpts, error) {

	var c confOpts

	err := conf.Load(&c, conf.Settings{
		ConfPath:    confPath,
		ConfType:    conf.ConfigTypeYAML,
		UnknownDeny: true,
	})

	// Check duplicates for instances
	for i, e1 := range c.Instances {
		for j := i + 1; j < len(c.Instances); j++ {
			if e1.Name == c.Instances[j].Name {
				return c, fmt.Errorf("Config parse error: duplicated values for name (name: %s)", e1.Name)
			}
		}
	}

	return c, err
}

func discoveryAction(s *Settings, ctx interface{}) (interface{}, error) {

	var d []testDiscovery

	appCtx := ctx.(appContext)

	// Check config errors
	if appCtx.conf.err != nil {
		return "", appCtx.conf.err
	}

	// Preparing data for discovery
	for _, e := range appCtx.conf.opts.Instances {
		d = append(d, testDiscovery{
			Name: e.Name,
			URL:  e.URL,
		})
	}

	return d, nil
}

func metricAction(s *Settings, ctx interface{}) (string, error) {

	appCtx := ctx.(appContext)

	// Check config errors
	if appCtx.conf.err != nil {
		return "", appCtx.conf.err
	}

	d := s.CacheGet(appCtx.instance, ctx, false)
	if d.InstanceAlive == false {
		return "", fmt.Errorf("InstanceAlive: fail")
	}

	j := gjson.Get(string(d.Data), appCtx.metric)

	if j.Exists() == true {
		return j.String(), nil
	}

	return "", fmt.Errorf("Can't find specified metric (metric: %s)", appCtx.metric)
}

func checkAliveAction(s *Settings, ctx interface{}) bool {

	appCtx := ctx.(appContext)

	// Check config errors
	if appCtx.conf.err != nil {
		return false
	}

	return s.CacheGet(appCtx.instance, ctx, false).InstanceAlive
}

func checkConfAction(s *Settings, ctx interface{}) error {
	return ctx.(appContext).conf.err
}

func exporter(s *Settings, ctx interface{}) ([]byte, error) {

	appCtx := ctx.(appContext)

	// Search instance by specified name
	instance := instanceLookup(appCtx)
	if instance == nil {
		return nil, fmt.Errorf("Instance `%s` not exists", appCtx.instance)
	}

	return connectImitation(instance.URL)
}

// instanceLookup finds context instance by specified in opt args name
func instanceLookup(ctx appContext) *Instance {

	for _, i := range ctx.conf.opts.Instances {
		if i.Name == ctx.instance {
			return &i
		}
	}

	return nil
}

func connectImitation(url string) ([]byte, error) {

	switch url {
	case "https://url1.org":
		return []byte(`{"key1_1":{"key1_2":"val1_2"}}`), nil
	case "https://url2.org":
		return []byte(`{"key2_1":{"key2_2":"val2_2"}}`), nil
	case "https://url3.org":
		return []byte(`{"key3_1":{"key3_2":"val3_2"}}`), nil
	}

	return nil, fmt.Errorf("Connection timeout")
}

func TestScript(t *testing.T) {

	// Load config file
	conf, err := confRead(confPathDefault)

	// Fill context
	ctx := appContext{
		instance: "instance1",
		metric:   "key1_1.key1_2",
		conf: contextConf{
			opts: conf,
			err:  err,
		},
	}

	s := Settings{
		DiscoveryAction:  discoveryAction,
		CheckAliveAction: checkAliveAction,
		CheckConfAction:  checkConfAction,
		MetricAction:     metricAction,
		Exporter:         exporter,
	}

	if ctx.conf.err == nil {
		s.CacheRoot = ctx.conf.opts.CacheRoot
		s.CacheTTL = ctx.conf.opts.CacheTTL
		s.User = ctx.conf.opts.User
		s.Group = ctx.conf.opts.Group
		s.CheckGUIDDisable = true
	}

	s.DebugSet(true)

	m := s.Action("metric", ctx)

	if m != "val1_2" {
		t.Fatalf("Wrong metric value, got: '%v'", m)
	}
}
