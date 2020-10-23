package embed

import (
	"etcd-cp/etcdserver/v3compactor"
	"etcd-cp/pkg/logutil"
	"etcd-cp/pkg/transport"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net"
	"net/url"
	"strings"
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

	maxElectionMs = 50000
)

var (
	DefaultInitialAdvertisePeerURLs = "http://localhost:2380"
	DefaultAdvertiseClientURLs      = "http://localhost:2379"
)

var (
	//CompactorModePeriodic是“ Config.AutoCompactionMode”字段的定期压缩模式。
	//如果“ AutoCompactionMode”为CompactorModePeriodic，
	//并且“ AutoCompactionRetention”为“ 1h”，则它将每小时自动压缩一次压缩存储
	CompactorModePeriodic = v3compactor.ModePeriodic

	//Config.AutoCompactionMode”字段的基于修订的压缩模式。
	//在此模式下，“AutoCompactionRetention”为“ 1000”，
	//则当当前修订为6000时，它将压缩修订5000上的日志。如果已处理足够的日志，则每5分钟运行一次。
	CompactorModeRevision = v3compactor.ModeRevision
)

var (
	ErrConflictBootstrapFlags = fmt.Errorf("multiple discovery or bootstrap flags are set. " +
		"Choose one of \"initial-cluster\", \"discovery\" or \"discovery-srv\"")
	ErrUnsetAdvertiseClientURLsFlag = fmt.Errorf("--advertise-client-urls is required when --listen-client-urls is set explicitly")
)

type Config struct {
	Name   string `json:"name"`
	Dir    string `json:"data-dir"`
	WalDir string `json:"wal-dir"`

	//心跳间隔时间是集群配置，非单点配置，要统一
	//TODO 解耦心跳间隔时间 和心跳周期
	TickMs     uint `json:"heartbeat-interval"`
	ElectionMs uint `json:"election-timeout"`

	LPUrls, LCUrls []url.URL
	APUrls, ACUrls []url.URL
	ClientTLSInfo  transport.TLSInfo
	ClientAutoTLS  bool
	PeerTLSInfo    transport.TLSInfo
	PeerAutoTLS    bool

	ClusterState   string `json:"initial-cluster-state"`
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

	// 自动压缩模式 可以只可以为Periodic和revision
	AutoCompactionMode string `json:"auto-compaction-mode"`
	//自动压缩的单位，可以是"5m"5分钟这种格式——周期时间
	//5000 ————条数
	//如果没提供单位 则默认是小时单位
	AutoCompactionRetention string `json:"auto-compaction-retention"`

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

		TickMs:     100,
		ElectionMs: 1000,

		logger:     nil,
		loggerMu:   new(sync.RWMutex),
		Logger:     "zap",
		LogLevel:   logutil.DefalutLogLevel,
		LogOutputs: []string{DefaultLogOutput},
	}

	return cfg

}

func (cfg *Config) Validate() error {
	err := cfg.setupLogging()
	if err != nil {
		return err
	}

	if err := checkBindUrls(cfg.LPUrls); err != nil {
		return err
	}
	if err := checkBindUrls(cfg.LCUrls); err != nil {
		return err
	}
	if err := checkBindUrls(cfg.APUrls); err != nil {
		addrs := cfg.getAPURLs()
		return fmt.Errorf(`--initial-advertise-peer-urls %q must be "host:port" (%v)`, strings.Join(addrs, ","), err)
	}
	if err := checkHostURLs(cfg.ACUrls); err != nil {
		addrs := cfg.getACURLs()
		return fmt.Errorf(`--advertise-client-urls %q must be "host:port" (%v)`, strings.Join(addrs, ","), err)
	}
	if err := checkBindUrls(cfg.ListenMetricsUrls); err != nil {
		return err
	}

	nSet := 0
	for _, v := range []bool{cfg.Durl != "", cfg.InitialCluster != "", cfg.DNSCluster != ""} {
		if v {
			nSet++
		}
	}

	if cfg.ClusterState != ClusterStateFlagExisting && cfg.ClusterState != ClusterStateFlagNew {
		return fmt.Errorf("unexpected clusterState %q", cfg.ClusterState)
	}

	if nSet > 1 {
		return ErrConflictBootstrapFlags
	}

	if cfg.TickMs == 0 {
		return fmt.Errorf("--heartbeat-interval must be >0 (set to %dms)", cfg.TickMs)
	}
	if cfg.ElectionMs == 0 {
		return fmt.Errorf("--election-timeout must be >0 (set to %dms)", cfg.ElectionMs)
	}
	if 5*cfg.TickMs > cfg.ElectionMs {
		return fmt.Errorf("--election-timeout[%vms] should be at least as 5 times as --heartbeat-interval[%vms]", cfg.ElectionMs, cfg.TickMs)
	}
	if cfg.ElectionMs > maxElectionMs {
		return fmt.Errorf("--election-timeout[%vms] is too long, and should be set less than %vms", cfg.ElectionMs, maxElectionMs)
	}

	//代理模式下忽略该错误  在服务模式下 设置客户端监听url 则必须设置推荐url
	if cfg.LCUrls != nil && cfg.ACUrls == nil {
		return ErrUnsetAdvertiseClientURLsFlag
	}

	if cfg.AutoCompactionMode != "" && cfg.AutoCompactionMode != CompactorModePeriodic && cfg.AutoCompactionMode != CompactorModeRevision {
		return fmt.Errorf("unknown auto-compaction-mode %q", cfg.AutoCompactionMode)
	}

	return nil

}

func (cfg Config) getLCURLs() []string {
	res := make([]string, len(cfg.LCUrls))
	for i := range cfg.LCUrls {
		res[i] = cfg.LCUrls[i].String()
	}
	return res
}

func (cfg Config) getACURLs() []string {
	res := make([]string, len(cfg.ACUrls))
	for i := range cfg.ACUrls {
		res[i] = cfg.ACUrls[i].String()
	}
	return res
}

func (cfg Config) getLPURLs() []string {
	res := make([]string, len(cfg.LPUrls))
	for i := range cfg.LPUrls {
		res[i] = cfg.LPUrls[i].String()
	}
	return res
}

func (cfg Config) getAPURLs() []string {
	res := make([]string, len(cfg.APUrls))
	for i := range cfg.APUrls {
		res[i] = cfg.APUrls[i].String()
	}
	return res
}

func checkBindUrls(urls []url.URL) error {
	for _, u := range urls {
		if u.Scheme == "unix" || u.Scheme == "unixs" {
			continue
		}

		host, _, err := net.SplitHostPort(u.String())
		if err != nil {
			return err
		}
		if host == "localhost" {
			//TODO 支持/etc/host
			continue
		}

		if net.ParseIP(host) == nil {
			return fmt.Errorf("expected IP in URL for binding (%s)", u.String())
		}

	}

	return nil
}

func checkHostURLs(urls []url.URL) error {
	for _, v := range urls {
		host, _, err := net.SplitHostPort(v.String())
		if err != nil {
			return err
		}
		if host == "" {
			return fmt.Errorf("unexpected empty host (%s)", v.String())
		}
	}

	return nil
}
