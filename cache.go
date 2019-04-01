package zbxscr

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
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
	InstanceAlive bool `yaml:"instance_alive"`

	// This field contains data was obtained from cache (if it's actual) or exporter.
	Data []byte `yaml:"data"`
}

type cacheIO struct {
	InstanceAlive bool   `yaml:"instance_alive"`
	Data          string `yaml:"data"`
}

// CacheGet retrieves data either from cache file if it's actual, or from exporter if cache rotted (in this case cache will be updated)
// If cache processing fails by any reasons, `InstanceAlive` field will be set to false.
// If `forceUpdate` argument is true, cache will be force updated.
func (s *Settings) CacheGet(name string, ctx interface{}, forceUpdate bool) Cache {

	var c Cache

	// Check exporter function defined
	if s.Exporter == nil {
		s.DebugPrint("Cache processing error: null exporter function\n")
		c.InstanceAlive = false
		return c
	}

	cacheFile := s.cacheFilePath(name)

	if forceUpdate == false {
		s.DebugPrint("Checking whether cache actual\n")
		actual, err := s.cacheCheckActual(cacheFile)
		if err != nil {
			c.InstanceAlive = false
			return c
		}

		if actual == true {
			s.DebugPrint("Cache is actual\n")
			s.DebugPrint("Reading cache\n")
			c, err = s.cacheRead(cacheFile)
			if err != nil {
				c.InstanceAlive = false
			} else {
				s.DebugPrint("Data successfully retrieved from cache (InstanceAlive: %t)\n", c.InstanceAlive)
			}
			return c
		}

		s.DebugPrint("Cache is outdated\n")
	}

	s.DebugPrint("Calling exporter\n")
	if d, err := s.Exporter(ctx); err != nil {
		s.DebugPrint("Exporter error: %v\n", err)
		c.InstanceAlive = false
	} else {
		c.InstanceAlive = true
		c.Data = d
		s.DebugPrint("Got data from exporter (InstanceAlive: %t)\n", c.InstanceAlive)
	}

	s.DebugPrint("Writing retrieved data to cache\n")
	if err := s.cacheWrite(name, c); err != nil {
		c.InstanceAlive = false
	}

	s.DebugPrint("Return data (InstanceAlive: %t)\n", c.InstanceAlive)
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

// cacheCheckActual checks cache file last modified time
// If cache is actual true will be returned
func (s *Settings) cacheCheckActual(file string) (bool, error) {

	var ttl float64

	if ttl = s.CacheTTL; ttl == 0 {
		ttl = cacheTTLDefault
	}

	// Get cache file stat to check last modified time
	i, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) == false {
			// If the problem is not related of the file
			// existence (e.g. permissions error)
			if s.debugMode {
				fmt.Printf("Can't stat cache data file: %v", err)
			}
			return false, err
		}

		// Cache file not exists
		return false, nil
	}

	// Check cache file last modified time
	if time.Now().Sub(i.ModTime()).Seconds() > ttl {
		// Cache is rotted
		return false, nil
	}

	// Cache is actual
	return true, nil
}

// cacheRead reads data from cache file
func (s *Settings) cacheRead(file string) (Cache, error) {

	var (
		c   Cache
		cio cacheIO
	)

	// Read cache data
	d, err := ioutil.ReadFile(file)
	if err != nil {
		if s.debugMode {
			fmt.Printf("Can't read cache: %v", err)
		}
		return c, err
	}

	// Unmarshal retrived data
	if err := yaml.Unmarshal(d, &cio); err != nil {
		if s.debugMode {
			fmt.Printf("Can't parse cache: %v", err)
		}
		return c, err
	}

	c.InstanceAlive = cio.InstanceAlive
	c.Data, err = base64.StdEncoding.DecodeString(cio.Data)
	if err != nil {
		s.DebugPrint("Can't decode cache data: %v\n", err)
		return c, err
	}

	// Success
	return c, nil
}

// cacheWrite writes cache to file
func (s *Settings) cacheWrite(name string, c Cache) error {

	// Marshal data
	d, err := yaml.Marshal(cacheIO{
		InstanceAlive: c.InstanceAlive,
		Data:          base64.StdEncoding.EncodeToString(c.Data),
	})
	if err != nil {
		if s.debugMode {
			fmt.Printf("Can't serialize cache: %v", err)
		}
		return err
	}

	// Create cache dir
	dir := s.cacheDirPath(name)
	if err := os.MkdirAll(dir, 0750); err != nil {
		if s.debugMode {
			fmt.Printf("Can't create cache dir: %v", err)
		}
		return err
	}

	// Write cache file
	file := s.cacheFilePath(name)
	if err := ioutil.WriteFile(file, d, 0640); err != nil {
		if s.debugMode {
			fmt.Printf("Can't write cache: %v", err)
		}
		return err
	}

	// Success
	return nil
}
