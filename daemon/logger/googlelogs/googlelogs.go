// Package googlelogs provides the logdriver for forwarding container logs to Google Cloud Logging.
package googlelogs

import (
	"fmt"
	"log"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/cloud"
	"google.golang.org/cloud/compute/metadata"
	"google.golang.org/cloud/logging"
)

const (
	name         = "googlelogs"
	logStreamKey = "googlelogs-stream"
)

type logStream struct {
	client       logging.Client
	hostname     string
	instanceName string
	zone         string
	closed       bool
}

// init registers the googlelogs driver and sets the default region, if provided
func init() {
	if err := logger.RegisterLogDriver(name, New); err != nil {
		logrus.Fatal(err)
	}
	if err := logger.RegisterLogOptValidator(name, ValidateLogOpt); err != nil {
		logrus.Fatal(err)
	}
}

// New creates an googlelogs logger using the configuration passed in on the
// context.
func New(ctx logger.Context) (logger.Logger, error) {
	logStreamName := ctx.ContainerID
	if ctx.Config[logStreamKey] != "" {
		logStreamName = ctx.Config[logStreamKey]
	}
	projID, err := metadata.ProjectID()
	if projID == "" {
		log.Printf("Error getting project ID: %v", err)
		return nil, err
	}
	hc, err := google.DefaultClient(oauth2.NoContext)
	if err != nil {
		log.Printf("Error creating default GCE OAuth2 client: %v", err)
		return nil, err
	}
	logClient, err := logging.NewClient(cloud.NewContext(projID, hc), projID, logStreamName)
	if err != nil {
		log.Printf("Error creating Google logging client: %v", err)
		return nil, err
	}
	hostname, err := metadata.Hostname()
	if hostname == "" {
		log.Printf("Error getting hostname: %v", err)
		return nil, err
	}
	instanceName, err := metadata.InstanceName()
	if hostname == "" {
		log.Printf("Error getting instance name: %v", err)
		return nil, err
	}
	zone, err := metadata.Zone()
	if hostname == "" {
		log.Printf("Error getting zone: %v", err)
		return nil, err
	}
	stream := &logStream{
		client:       *logClient,
		hostname:     hostname,
		instanceName: instanceName,
		zone:         zone,
	}
	return stream, nil
}

// Name returns the name of the googlelogs logging driver
func (l *logStream) Name() string {
	return name
}

// Log submits messages for logging by an instance of the googlelogs logging driver
func (l *logStream) Log(msg *logger.Message) error {
	entry := logging.Entry{
		Time: msg.Timestamp,
		Labels: map[string]string{
			"ContainerId":  msg.ContainerID,
			"Source":       msg.Source,
			"Hostname":     l.hostname,
			"InstanceName": l.instanceName,
			"Zone":         l.zone,
		},
		Payload: msg.Line,
	}
	if !l.closed {
		l.client.Log(entry)
	}
	return nil
}

// Close closes the instance of the googlelogs logging driver
func (l *logStream) Close() error {
	return nil
}

// ValidateLogOpt looks for googlelogs-specific log options
// googlelogs-group, and googlelogs-stream
func ValidateLogOpt(cfg map[string]string) error {
	for key := range cfg {
		switch key {
		case logStreamKey:
		default:
			return fmt.Errorf("unknown log opt '%s' for %s log driver", key, name)
		}
	}
	return nil
}
