package embed

import (
	"etcd-cp/pkg/logutil"
	"etcd-cp/pkg/transport"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/url"
	"sync"
)

const (
	ClusterStateFlagNew      = "new"
	ClusterStateFlagExisting = "existing"

	DefaultName         = "default"
	DefaultMaxSnapshots = 5
	DefaultMaxWALs      = 5

	// DefaultStrictReconfigCheck is the default value for "--strict-reconfig-check" flag.
	// 是否使用默认的 --strict-reconfig-check flag
	DefaultStrictReconfigCheck = true
	//v2版本的api开启按钮
	DefaultEnableV2 = false

	DefaultListenPeerURLs   = "http://localhost:2380"
	DefaultListenClientURLs = "http://localhost:2379"

	DefaultLogOutput = "default"
	JournalLogOutput = "systemd/journal"
	StdErrLogOutput  = "stderr"
	StdOutLogOutput  = "stdout"
)

var (
	DefaultInitialAdvertisePeerURLs = "http://localhost:2380"
	DefaultAdvertiseClientURLs      = "http://localhost:2379"
)

var (
	ErrUnsetAdvertiseClientURLsFlag = fmt.Errorf("--advertise-client-urls is required when --listen-client-urls is set explicitly")
)

type Config struct {
	Name   string `json:"name"`
	Dir    string `json:"data-dir"`
	WalDir string `json:"wal-dir"`

	LPUrls, LCUrls []url.URL
	APUrls, ACUrls []url.URL
	ClientTLSInfo  transport.TLSInfo
	ClientAutoTLS  bool
	PeerTLSInfo    transport.TLSInfo
	PeerAutoTLS    bool

	InitialCluster string `json:"initial-cluster"`
	DNSCluster     string `json:"discovery-srv"`
	DNSClusterName string `json:"discovery-srv-name"`
	Durl           string `json:"discovery"`

	//日志选项，目前只支持zap
	Logger string `json:"logger"`
	//日志级别只支持 debug, info, warn, error, panic, fatal, default
	LogLevel string `json:"log-level"`
	//日志输出，
	// - "default" os.stderr
	// - "stderr" os.stderr
	// - "stdout" os.stdout
	// - file path ，当日志是ZAP时候 ，可以支持多个选项
	LogOutputs       []string `json:"log-outputs"`
	ZapLoggerBuilder func(config *Config) error

	//日志记录器记录的服务端侧的数据， setupLogging一定要在服务启动之前被调用，禁止直接设置logger
	loggerMu *sync.RWMutex
	logger   *zap.Logger

	//这个日志配置是一个raft算法的日志配置
	//必须设置，即loggerConfig != nil || (loggerCore != nil && loggerWriteSyncer != nil)
	loggerConfig      *zap.Config
	loggerCore        zapcore.Core
	loggerWriteSyncer zapcore.WriteSyncer

	ListenMetricsUrls []url.URL
}

func NewConfig() *Config {
	lpurl, _ := url.Parse(DefaultListenPeerURLs)
	apurl, _ := url.Parse(DefaultInitialAdvertisePeerURLs)
	lcurl, _ := url.Parse(DefaultListenClientURLs)
	acurl, _ := url.Parse(DefaultAdvertiseClientURLs)

	cfg := &Config{
		Name: DefaultName,

		LPUrls: []url.URL{*lpurl},
		LCUrls: []url.URL{*lcurl},
		APUrls: []url.URL{*apurl},
		ACUrls: []url.URL{*acurl},

		logger:     nil,
		loggerMu:   new(sync.RWMutex),
		Logger:     "zap",
		LogLevel:   logutil.DefalutLogLevel,
		LogOutputs: []string{DefaultLogOutput},
	}

	return cfg

}

func (cfg *Config) Validate() error {

	return nil

}
