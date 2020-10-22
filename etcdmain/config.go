package etcdmain

import (
	"etcd-cp/embed"
	"etcd-cp/pkg/flags"
	"etcd-cp/version"
	"flag"
	"fmt"
	"go.uber.org/zap"
	"os"
	"runtime"
)

var (
	proxyFlagOff      = "off"
	proxyFlagReadonly = "readonly"
	proxyFlagOn       = "on"

	fallbackFlagExit  = "exit"
	fallbackFlagProxy = "proxy"

	ignored = []string{
		"cluster-active-size",
		"cluster-remove-delay",
		"cluster-sync-interval",
		"config",
		"force",
		"max-result-buffer",
		"max-retry-attempts",
		"peer-heartbeat-interval",
		"peer-election-timeout",
		"retry-interval",
		"snapshot",
		"v",
		"vv",
		// for coverage testing
		"test.coverprofile",
		"test.outputdir",
	}
)

type configProxy struct {
	//ProxyFailureWaitMs     uint `json:"proxy-failure-wait"`
	//ProxyRefreshIntervalMs uint `json:"proxy-refresh-interval"`
	//ProxyDialTimeoutMs     uint `json:"proxy-dial-timeout"`
	//ProxyWriteTimeoutMs    uint `json:"proxy-write-timeout"`
	//ProxyReadTimeoutMs     uint `json:"proxy-read-timeout"`
	Fallback string
	Proxy    string
	//ProxyJSON              string `json:"proxy"`
	//FallbackJSON           string `json:"discovery-fallback"`
}

// configFlags has the set of flags used for command line parsing a Config
type configFlags struct {
	flagSet      *flag.FlagSet
	clusterState *flags.SelectiveStringValue
	fallback     *flags.SelectiveStringValue
	proxy        *flags.SelectiveStringValue
}

type config struct {
	ec embed.Config
	cp configProxy
	cf configFlags

	configFile   string
	printVersion bool
	ignored      []string
}

func newConfig() *config {
	cfg := &config{
		ec: *embed.NewConfig(),
		cf: configFlags{
			flagSet:      flag.NewFlagSet("etcd", flag.ContinueOnError),
			clusterState: flags.NewSelectiveStringValue(embed.ClusterStateFlagNew, embed.ClusterStateFlagExisting),
			fallback:     flags.NewSelectiveStringValue(fallbackFlagExit, fallbackFlagProxy),
			proxy:        flags.NewSelectiveStringValue(proxyFlagOff, proxyFlagOn, proxyFlagReadonly),
		},
		cp: configProxy{
			Proxy: proxyFlagOff,
		},
		ignored: ignored,
	}

	fs := cfg.cf.flagSet
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, usageline)
	}

	//配置文件会覆盖命令行和环境变量参数
	fs.StringVar(&cfg.configFile, "config-file", "", "Path to the server configuration file. Note that if a configuration file is provided, other command line flags and environment variables will be ignored.")

	//成员配置
	fs.StringVar(&cfg.ec.Dir, "data-dir", cfg.ec.Dir, "Path to the data directory.")
	fs.StringVar(&cfg.ec.WalDir, "wal-dir", cfg.ec.WalDir, "Path to the dedicated wal directory.")

	fs.Var(flags.NewUniqueURLsWithExceptions(embed.DefaultListenPeerURLs, ""),
		"listen-peer-urls",
		"List of URLs to listen on for peer traffic.",
	)
	fs.Var(
		flags.NewUniqueURLsWithExceptions(embed.DefaultListenClientURLs, ""),
		"listen-client-urls",
		"List of URLs to listen on for client traffic.",
	)
	fs.Var(
		flags.NewUniqueURLsWithExceptions("", ""),
		"listen-metrics-urls",
		"List of URLs to listen on for the metrics and health endpoints.",
	)

	//集群参数
	fs.Var(
		flags.NewUniqueURLsWithExceptions(embed.DefaultInitialAdvertisePeerURLs, ""),
		"initial-advertise-peer-urls",
		"List of this member's peer URLs to advertise to the rest of the cluster.",
	)
	fs.Var(
		flags.NewUniqueURLsWithExceptions(embed.DefaultAdvertiseClientURLs, ""),
		"advertise-client-urls",
		"List of this member's client URLs to advertise to the public.",
	)
	fs.Var(cfg.cf.fallback, "discovery-fallback", fmt.Sprintf("Valid values include %q", cfg.cf.fallback.Valids()))
	fs.Var(cfg.cf.clusterState, "initial-cluster-state", "Initial cluster state ('new' or 'existing').")

	//版本参数
	fs.BoolVar(&cfg.printVersion, "version", false, "Print the version and exit.")

	//代理参数
	fs.Var(cfg.cf.proxy, "proxy", fmt.Sprintf("Valid values include %q", cfg.cf.proxy.Valids()))

	for _, f := range cfg.ignored {
		fs.Var(&flags.IgnoredFlag{Name: f}, f, "")
	}

	return cfg
}

func (cfg *config) parse(arguments []string) error {
	perr := cfg.cf.flagSet.Parse(arguments)
	switch perr {
	case nil:
	case flag.ErrHelp:
		fmt.Println(flagsline)
		os.Exit(0)
	default:
		os.Exit(2)
	}

	//检查是否有不在集合中标识的flag  会报警
	if len(cfg.cf.flagSet.Args()) != 0 {
		return fmt.Errorf("'%s' is not a valid flag", cfg.cf.flagSet.Arg(0))
	}

	if cfg.printVersion {
		fmt.Printf("etcd Version: %s\n", version.Version)
		fmt.Printf("Git SHA: %s\n", version.GitSHA)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	var err error

	if cfg.configFile == "" {
		cfg.configFile = os.Getenv(flags.FlagToEnv("ETCD", "config-file"))
	}

	if cfg.configFile != "" {

	} else {
		err = cfg.configFromCmd()
	}

	return err
}

func (cfg *config) configFromCmd() error {
	lg, err := zap.NewProduction()
	if err != nil {
		return err
	}

	verKey := "ETCD_VERSION"
	if verVal := os.Getenv(verKey); verVal != "" {
		err := os.Unsetenv(verKey)
		if err != nil {
			panic(err)
		}

		lg.Warn(
			"cannot set special environment variable",
			zap.String("key", verKey),
			zap.String("value", verVal),
		)
	}

	err = flags.SetFlagsFromEnv(lg, "ETCD", cfg.cf.flagSet)
	if err != nil {
		return nil
	}

	//cmd+env 更新ec, 注意env不会覆盖cmd中的，只会补充cmd没有设置的（默认的也会被env覆盖）
	cfg.ec.LPUrls = flags.UniqueURLsFromFlag(cfg.cf.flagSet, "listen-peer-urls")
	cfg.ec.APUrls = flags.UniqueURLsFromFlag(cfg.cf.flagSet, "initial-advertise-peer-urls")
	cfg.ec.LCUrls = flags.UniqueURLsFromFlag(cfg.cf.flagSet, "listen-client-urls")
	cfg.ec.ACUrls = flags.UniqueURLsFromFlag(cfg.cf.flagSet, "advertise-client-urls")
	cfg.ec.ListenMetricsUrls = flags.UniqueURLsFromFlag(cfg.cf.flagSet, "listen-metrics-urls")

	cfg.cp.Fallback = cfg.cf.fallback.String()
	cfg.cp.Proxy = cfg.cf.proxy.String()

	//设置了listen-client-urls，但未设置advertise-client-urls, 切不可能是代理模式，则将ACUrls清空
	missingAC := flags.IsSet(cfg.cf.flagSet, "listen-client-urls") && !flags.IsSet(cfg.cf.flagSet, "advertise-client-urls")
	if !cfg.mayBeProxy() && missingAC {
		cfg.ec.ACUrls = nil
	}

	//关闭集群模式，如果发现模式打开切集群模式关闭着
	if (cfg.ec.Durl != "" || cfg.ec.DNSCluster != "" || cfg.ec.DNSClusterName != "") && flags.IsSet(cfg.cf.flagSet, "initial-cluster") {
		cfg.ec.InitialCluster = ""
	}

	return cfg.validate()
}

func (cfg *config) configFromFile() error {

	return nil
}

//什么情况下回退回到proxy模式
func (cfg *config) mayBeProxy() bool {
	//discovery不为空（应该是一个url），切可回退到代理模式， 标记为可回退至代理模式
	mayFallBackToProxy := cfg.ec.Durl != "" && cfg.cp.Fallback == fallbackFlagProxy
	//切代理模式不为关闭状态也有可能是代理模式
	return cfg.cp.Proxy != proxyFlagOff || mayFallBackToProxy
}

func (cfg *config) validate() error {
	err := cfg.ec.Validate()

	if err == embed.ErrUnsetAdvertiseClientURLsFlag && cfg.mayBeProxy() {
		return nil
	}

	return err
}

func (cfg config) isProxy() bool               { return cfg.cf.proxy.String() != proxyFlagOff }
func (cfg config) isReadOnlyProxy() bool       { return cfg.cf.proxy.String() == proxyFlagReadonly }
func (cfg config) shouldFallBackToProxy() bool { return cfg.cf.fallback.String() == fallbackFlagProxy }
