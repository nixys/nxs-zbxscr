package zbxscr

import (
	"encoding/json"
	"fmt"
	"os/user"
	"strconv"
	"syscall"
)

import (
	//#include <unistd.h>
	//#include <errno.h>
	"C"
)

const (
	// MsgNotSupported - error message from zabbix agent
	MsgNotSupported = "ZBX_NOTSUPPORTED"

	// UserDefault is a default username wich set as UID
	UserDefault = "zabbix"

	// GroupDefault is a default group wich set as GID
	GroupDefault = "zabbix"

	checkConfFail    = "0"
	checkConfSuccess = "1"

	checkAliveFail    = "0"
	checkAliveSuccess = "1"
)

// DiscoveryFunc defines the process to service discovery.
// This function must return the slice of elements to be sent to Zabbix server to
// automatically create items, triggers, and graphs for different entities.
type DiscoveryFunc func(s *Settings, ctx interface{}) (interface{}, error)

// CheckConfFunc defines the process to check config for syntax errors, instances duplicates, etc.
type CheckConfFunc func(s *Settings, ctx interface{}) error

// CheckAliveFunc defines the process to check alive for specified instance.
type CheckAliveFunc func(s *Settings, ctx interface{}) bool

// MetricFunc defines the process to obtain specified metric
type MetricFunc func(s *Settings, ctx interface{}) (string, error)

// ExporterFunc defines the process to obtain all the needed data from monitored service.
// Usually this function automatically called from cache process, when the cache data is outdated,
// or from `MetricFunc` when cache disabled.
type ExporterFunc func(s *Settings, ctx interface{}, c Cache) ([]byte, error)

// Settings is struct to store settings
type Settings struct {

	// Directory path to save all cache
	CacheRoot string

	// Directory path to save instances cache
	CacheSubRoot string

	// Cache TTL in seconds
	CacheTTL float64

	// Whether or not to check user and group of running application
	CheckGUIDDisable bool

	// User to SUID. Available only for root
	User string

	// Group to SGID. Available only for root
	Group string

	// See functions description above for details
	DiscoveryAction  DiscoveryFunc
	CheckConfAction  CheckConfFunc
	CheckAliveAction CheckAliveFunc
	MetricAction     MetricFunc
	Exporter         ExporterFunc

	// Whether or not print debug message
	debugMode bool
}

type discovery struct {
	Data interface{} `json:"data"`
}

// Action is package entrypoint function
func (s *Settings) Action(action string, ctx interface{}) string {

	if err := s.checkGUID(); err != nil {
		s.DebugPrint("Error while checking uid or gid for process: %v (try to use 'sudo')\n", err)
		return fmt.Sprintf("%s: %v", MsgNotSupported, err)
	}

	switch action {
	case "discovery":

		if s.DiscoveryAction == nil {
			s.DebugPrint("Action 'discovery' not implemented\n")
			return fmt.Sprintf("%s: action 'discovery' not implemented", MsgNotSupported)
		}

		d, err := s.DiscoveryAction(s, ctx)
		if err != nil {
			s.DebugPrint("Discovery processing error: %v\n", err)
			return fmt.Sprintf("%s: %v", MsgNotSupported, err)
		}

		r := discovery{
			Data: d,
		}

		u, _ := json.Marshal(r)

		return string(u)

	case "check_conf":

		if s.CheckConfAction == nil {
			s.DebugPrint("Action 'check_conf' not implemented\n")
			return fmt.Sprintf("%s: action 'check_conf' not implemented", MsgNotSupported)
		}

		if err := s.CheckConfAction(s, ctx); err != nil {
			s.DebugPrint("Check config processing error: %v\n", err)
			return checkConfFail
		}
		return checkConfSuccess

	case "check_alive":

		if s.CheckAliveAction == nil {
			s.DebugPrint("Action 'check_alive' not implemented\n")
			return fmt.Sprintf("%s: action 'check_alive' not implemented", MsgNotSupported)
		}

		if r := s.CheckAliveAction(s, ctx); r == false {
			return checkAliveFail
		}
		return checkAliveSuccess

	case "metric":

		if s.MetricAction == nil {
			s.DebugPrint("Action 'metric' not implemented\n")
			return fmt.Sprintf("%s: action 'metric' not implemented", MsgNotSupported)
		}

		r, err := s.MetricAction(s, ctx)
		if err != nil {
			s.DebugPrint("Get metric processing error: %v\n", err)
			return fmt.Sprintf("%s: %v", MsgNotSupported, err)
		}
		return r
	}

	return fmt.Sprintf("%s: unknown metric", MsgNotSupported)
}

// DebugSet toggles the debug messages to stdout
func (s *Settings) DebugSet(toggle bool) {
	s.debugMode = toggle
}

// DebugGet returns current debug toggle
func (s *Settings) DebugGet() bool {
	return s.debugMode
}

// DebugPrint prints the message if debug toggle true
func (s *Settings) DebugPrint(format string, a ...interface{}) {
	if s.debugMode == true {
		fmt.Printf(format, a...)
	}
}

func (s *Settings) checkGUID() error {

	if s.CheckGUIDDisable == true {
		return nil
	}

	uid, gid, usename, groupname, err := s.getGUID()
	if err != nil {
		return err
	}

	if syscall.Getuid() != uid {
		return fmt.Errorf("'%s' user expected", usename)
	}

	if syscall.Getgid() != gid {
		return fmt.Errorf("'%s' group expected", groupname)
	}

	return nil
}

func (s *Settings) getGUID() (int, int, string, string, error) {

	// Check and set default username
	usename := UserDefault
	if len(s.User) > 0 {
		usename = s.User
	}

	// Check and set default groupname
	groupname := GroupDefault
	if len(s.User) > 0 {
		groupname = s.Group
	}

	// Determine UID by specified username
	u, err := user.Lookup(usename)
	if err != nil {
		return 0, 0, "", "", err
	}
	uid, _ := strconv.Atoi(u.Uid)

	// Determine GID by specified groupname
	g, err := user.LookupGroup(groupname)
	if err != nil {
		return 0, 0, "", "", err
	}
	gid, _ := strconv.Atoi(g.Gid)

	return uid, gid, usename, groupname, nil
}
