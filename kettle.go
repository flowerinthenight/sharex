package kettle

import (
	"fmt"
	"log"
	"os"

	zpubsub "github.com/NYTimes/gizmo/pubsub"
	"github.com/fatih/color"
	"github.com/go-redsync/redsync"
)

const (
	masterTimeout = 30 // seconds
	keyTTL        = 29 // seconds

	cmdReportWorkerName = "CMD_REPORT_WORKER_NAME"
	cmdStartWork        = "CMD_START_WORK"
)

var (
	red   = color.New(color.FgRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
)

type DistLocker interface {
	Lock() error
	Unlock() bool
}

type KettleOption interface {
	Apply(*kettle)
}

type withName string

func (w withName) Apply(o *kettle)   { o.name = string(w) }
func WithName(v string) KettleOption { return withName(v) }

type withVerbose bool

func (w withVerbose) Apply(o *kettle) { o.verbose = bool(w) }
func WithVerbose(v bool) KettleOption { return withVerbose(v) }

type withDistLocker struct{ dl DistLocker }

func (w withDistLocker) Apply(o *kettle)       { o.lock = w.dl }
func WithDistLocker(v DistLocker) KettleOption { return withDistLocker{v} }

type withPublisher struct{ pub zpubsub.MultiPublisher }

func (w withPublisher) Apply(o *kettle)                   { o.pub = w.pub }
func WithPublisher(v zpubsub.MultiPublisher) KettleOption { return withPublisher{v} }

type withSubscriber struct{ sub zpubsub.Subscriber }

func (w withSubscriber) Apply(o *kettle)               { o.sub = w.sub }
func WithSubscriber(v zpubsub.Subscriber) KettleOption { return withSubscriber{v} }

type kettle struct {
	name    string
	verbose bool
	lock    DistLocker
	pub     zpubsub.Publisher
	sub     zpubsub.Subscriber
}

func (s kettle) Name() string    { return s.name }
func (s kettle) IsVerbose() bool { return s.verbose }

func (s kettle) info(v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintln(v...)
	log.Printf("%s %s", green("[info]"), m)
}

func (s kettle) infof(format string, v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintf(format, v...)
	log.Printf("%s %s", green("[info]"), m)
}

func (s kettle) error(v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintln(v...)
	log.Printf("%s %s", red("[error]"), m)
}

func (s kettle) errorf(format string, v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintf(format, v...)
	log.Printf("%s %s", red("[error]"), m)
}

func (s kettle) fatal(v ...interface{}) {
	s.error(v...)
	os.Exit(1)
}

func (s kettle) fatalf(format string, v ...interface{}) {
	s.errorf(format, v...)
	os.Exit(1)
}

func New(opts ...KettleOption) (*kettle, error) {
	s := &kettle{
		name: "kettle",
	}

	for _, opt := range opts {
		opt.Apply(s)
	}

	if s.lock == nil {
		pool, err := NewRedisPool()
		if err != nil {
			return nil, err
		}

		pools := []redsync.Pool{pool}
		rs := redsync.New(pools)
		s.lock = rs.NewMutex(fmt.Sprintf("%v-distlocker", s.name))
	}

	if s.pub == nil {
		pub, err := NewPublisher(fmt.Sprintf("%v-snspublisher", s.name))
		if err != nil {
			return nil, err
		}

		s.pub = pub
	}

	return s, nil
}
