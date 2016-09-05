package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/imdario/mergo"
	"github.com/supergiant/supergiant/pkg/client"
	"github.com/supergiant/supergiant/pkg/model"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type NodeSize struct {
	Name     string  `json:"name"`
	RAMGIB   float64 `json:"ram_gib"`
	CPUCores float64 `json:"cpu_cores"`
}

type Settings struct {
	ConfigFilePath string

	UIEnabled   bool   `json:"ui_enabled"`
	PsqlHost    string `json:"psql_host"`
	PsqlDb      string `json:"psql_db"`
	PsqlUser    string `json:"psql_user"`
	PsqlPass    string `json:"psql_pass"`
	SQLiteFile  string `json:"sqlite_file"`
	PublishHost string `json:"publish_host"`
	HTTPPort    string `json:"http_port"`
	HTTPSPort   string `json:"https_port"`
	SSLCertFile string `json:"ssl_cert_file"`
	SSLKeyFile  string `json:"ssl_key_file"`
	LogPath     string `json:"log_file"`
	LogLevel    string `json:"log_level"`

	// NOTE these MUST be provided in ascending order by cost in order to
	// correctly provision the smallest size on Kube creation
	//
	// NodeSizes is a map of provider name (ex. "aws") and node sizes
	NodeSizes map[string][]*NodeSize `json:"node_sizes"`
}

type Core struct {
	Settings

	// NOTE we do this to prevent having to load all the cloud provider various
	// lib code everytime we load core
	AWSProvider func(map[string]string) Provider
	DOProvider  func(map[string]string) Provider

	Log *logrus.Logger

	DB *DB

	Sessions         *Sessions
	Users            *Users
	CloudAccounts    *CloudAccounts
	Kubes            *Kubes
	Apps             *Apps
	Components       *Components
	Releases         *Releases
	Instances        *Instances
	Volumes          *Volumes
	PrivateImageKeys *PrivateImageKeys
	Entrypoints      *Entrypoints
	Nodes            *Nodes

	// TODO should this be a pseudo-collection like Sessions?
	Actions *SafeMap
}

// NOTE this used to be core.New(), but due to how we load in values from the
// cli package, I needed to first actually initialize a Core struct and then
// configure.
func (c *Core) Initialize() {
	if err := c.InitializeForeground(); err != nil {
		panic(err)
	}
	c.InitializeBackground()
}

//------------------------------------------------------------------------------

// InitializeForeground sets up Log and DB on *Core.
func (c *Core) InitializeForeground() error {
	if c.ConfigFilePath != "" {
		var configFileSettings Settings
		configFile, err := os.Open(c.ConfigFilePath)
		if err != nil {
			return err
		}
		if err := json.NewDecoder(configFile).Decode(&configFileSettings); err != nil {
			return err
		}
		// Merge in command line settings (which overwrite respective config file settings)
		if err := mergo.Merge(&c.Settings, configFileSettings); err != nil {
			return err
		}
	}

	// TODO use struct tags on settings; can set defaults as well
	requiredFlags := map[string]string{
		"publish-host": c.PublishHost,
		"http-port":    c.HTTPPort,
	}
	for flag, val := range requiredFlags {
		if val == "" {
			return errors.New(flag + " required")
		}
	}

	// Logging
	c.Log = logrus.New()
	if c.LogLevel != "" {
		levelInt, err := logrus.ParseLevel(c.LogLevel)
		if err != nil {
			return err
		}
		c.Log.Level = levelInt
	}
	// db.LogMode(true)
	// guber.Log.SetLevel("debug")

	// DB
	var db *gorm.DB
	var err error

	if c.PsqlHost != "" && c.PsqlDb != "" && c.PsqlUser != "" && c.PsqlPass != "" {
		// Postgres
		options := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable", c.PsqlHost, c.PsqlDb, c.PsqlUser, c.PsqlPass)
		db, err = gorm.Open("postgres", options)

	} else if c.SQLiteFile != "" {
		// SQLite3
		db, err = gorm.Open("sqlite3", c.SQLiteFile)

	} else {
		err = errors.New("No DB configured. Must provide --psql-* options or --sqlite-file.")
	}
	if err != nil {
		return err
	}

	c.DB = &DB{c, db}
	err = c.DB.AutoMigrate(
		&model.User{},
		&model.Kube{},
		&model.CloudAccount{},
		&model.PrivateImageKey{},
		&model.App{},
		&model.Component{},
		&model.ComponentPrivateImageKey{},
		&model.Release{},
		&model.Instance{},
		&model.Volume{},
		&model.Entrypoint{},
		&model.Node{},
	).Error
	if err != nil {
		return err
	}
	c.Users = &Users{Collection{c}}
	c.Kubes = &Kubes{Collection{c}}
	c.CloudAccounts = &CloudAccounts{Collection{c}}
	c.Apps = &Apps{Collection{c}}
	c.Components = &Components{Collection{c}}
	c.Releases = &Releases{Collection{c}}
	c.Instances = &Instances{Collection{c}}
	c.Volumes = &Volumes{Collection{c}}
	c.PrivateImageKeys = &PrivateImageKeys{Collection{c}}
	c.Entrypoints = &Entrypoints{Collection{c}}
	c.Nodes = &Nodes{Collection{c}}
	c.Sessions = NewSessions(c)

	// Actions for async work
	c.Actions = NewSafeMap(c)

	return nil
}

// InitializeBackground starts Action processing and RecurringServices for *Core.
func (c *Core) InitializeBackground() {
	// Recurring services
	capacityService := &RecurringService{
		core:     c,
		service:  &CapacityService{c},
		interval: 30 * time.Second,
	}
	nodeObserver := &RecurringService{
		core:     c,
		service:  &NodeObserver{c},
		interval: 30 * time.Second,
	}
	instanceObserver := &RecurringService{
		core:     c,
		service:  &InstanceObserver{c},
		interval: 30 * time.Second,
	}
	sessionExpirer := &RecurringService{
		core:     c,
		service:  &SessionExpirer{c},
		interval: 15 * time.Second,
	}

	go capacityService.Run()
	go nodeObserver.Run()
	go instanceObserver.Run()
	go sessionExpirer.Run()
}

//------------------------------------------------------------------------------

func (c *Core) SSLEnabled() bool {
	return c.HTTPSPort != "" && c.SSLCertFile != "" && c.SSLKeyFile != ""
}

func (c *Core) HTTPSURL() string {
	return fmt.Sprintf("https://%s:%s", c.PublishHost, c.HTTPSPort)
}

func (c *Core) HTTPURL() string {
	return fmt.Sprintf("http://%s:%s", c.PublishHost, c.HTTPPort)
}

func (c *Core) BaseURL() string {
	if c.SSLEnabled() {
		return c.HTTPSURL()
	}
	return c.HTTPURL()
}

func (c *Core) APIURL() string {
	return fmt.Sprintf("%s/api/v0", c.BaseURL())
}

func (c *Core) UIURL() string {
	return fmt.Sprintf("%s/ui", c.BaseURL())
}

func (c *Core) NewAPIClient(authType string, authToken string) *client.Client {
	return client.New(c.APIURL(), authType, authToken, c.SSLCertFile)
}
