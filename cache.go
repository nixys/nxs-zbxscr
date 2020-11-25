package zbxscr

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	fslock "github.com/juju/fslock"
)

const (
	cacheTTLDefault = 60
	cacheFileName   = "cache"
)

// Cache contains data cache
type Cache struct {

	// This field indicates for ability to retrieve data from instance endpoint.
	// If true  - data successfully retrieved from from cache file (if it's actual) or exeporter without any errors.
	// If false - some errors occurs while processing data retrieve (from cache or exporter)
	InstanceAlive bool `json:"instance_alive"`

	// This field contains data was obtained from cache (if it's actual) or exporter.
	Data []byte `json:"data"`
}

// CacheGet retrieves data either from cache file if it's actual, or from exporter if cache rotted (in this case cache will be updated)
// If cache processing fails by any reasons, `InstanceAlive` field will be set to false.
// If `forceUpdate` argument is true, cache will be force updated.
func (s *Settings) CacheGet(name string, ctx interface{}, forceUpdate bool) Cache {

	// Check exporter function defined
	if s.Exporter == nil {
		s.DebugPrint("Cache processing error: null exporter function\n")
		return Cache{}
	}

	actual, c, err := s.cacheRead(s.cacheFilePath(name))
	if err != nil {
		s.DebugPrint("Cache read error: %s\n", err)
		return Cache{}
	}

	// If cache exists, actual and forceUpdate is false
	if forceUpdate == false && actual == true {
		s.DebugPrint("Cache is actual\n")
		return c
	}

	s.DebugPrint("Cache is outdated\n")
	s.DebugPrint("Calling exporter\n")

	if d, err := s.Exporter(s, ctx, c); err != nil {
		// Cleanup cache struct
		c = Cache{}
		s.DebugPrint("Exporter error: %s\n", err)
	} else {
		// Fill cache struct with new data
		c.InstanceAlive = true
		c.Data = d
		s.DebugPrint("Got data from exporter (InstanceAlive: %t)\n", c.InstanceAlive)
	}

	s.DebugPrint("Writing retrieved data to cache\n")
	if err := s.cacheWrite(name, c); err != nil {
		s.DebugPrint("Cache write error: %s\n", err)
		return Cache{}
	}

	s.DebugPrint("Return cache data (InstanceAlive: %t)\n", c.InstanceAlive)
	return c
}

// Create full cache file path string
func (s *Settings) cacheFilePath(name string) string {
	return strings.Join([]string{s.CacheRoot, name, cacheFileName}, "/")
}

// Create directory path string where cache file will be located
func (s *Settings) cacheDirPath(name string) string {
	return strings.Join([]string{s.CacheRoot, name}, "/")
}

// cacheCheckState checks cache file state (existence and actual)
func (s *Settings) cacheCheckState(file string) (bool, bool, error) {

	var ttl float64

	ttl = s.CacheTTL
	if ttl == 0 {
		ttl = cacheTTLDefault
	}

	// Get cache file stat to check last modified time
	i, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) == false {
			// If the problem is not related of the file
			// existence (e.g. permissions error)
			return false, false, err
		}

		// Cache file not exists
		return false, false, nil
	}

	// Check cache file last modified time
	if time.Now().Sub(i.ModTime()).Seconds() > ttl {
		// Cache exist but rotted
		return true, false, nil
	}

	// Cache exist and actual
	return true, true, nil
}

// cacheRead reads data from cache file
func (s *Settings) cacheRead(file string) (bool, Cache, error) {

	var c Cache

	e, a, err := s.cacheCheckState(file)
	if err != nil {
		return false, Cache{}, err
	}

	if e == false || a == false {
		return false, Cache{}, nil
	}

	// Read cache data
	d, err := ioutil.ReadFile(file)
	if err != nil {
		return false, Cache{}, err
	}

	// Unmarshal retrived data
	if err := json.Unmarshal(d, &c); err != nil {
		return false, Cache{}, err
	}

	// Success
	return a, c, nil
}

// cacheWrite writes cache to file
func (s *Settings) cacheWrite(name string, c Cache) error {

	// Marshal data
	d, err := json.Marshal(c)
	if err != nil {
		return err
	}

	// Create cache dir
	dir := s.cacheDirPath(name)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	// Write cache file
	file := s.cacheFilePath(name)

	// Tries to lock the lock until the timeout expires
	lock := fslock.New(file)
	err = lock.LockWithTimeout(time.Second * 30)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(file, d, 0640); err != nil {
		return err
	}
	defer lock.Unlock()

	// Success
	return nil
}
