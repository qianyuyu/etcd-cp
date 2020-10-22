package embed

import "net/url"

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
)

var (
	DefaultInitialAdvertisePeerURLs = "http://localhost:2380"
	DefaultAdvertiseClientURLs      = "http://localhost:2379"
)

type Config struct {
	Name   string `json:"name"`
	Dir    string `json:"data-dir"`
	WalDir string `json:"wal_dir"`

	LPUrls, LCUrls []url.URL
	APUrls, ACUrls []url.URL

	Durl string `json:"discovery"`

	ListenMetricsUrls []url.URL
}

func NewConfig() *Config {
	lpurl, _ := url.Parse(DefaultListenPeerURLs)
	apurl, _ := url.Parse(DefaultInitialAdvertisePeerURLs)
	lcurl, _ := url.Parse(DefaultListenClientURLs)
	acurl, _ := url.Parse(DefaultAdvertiseClientURLs)

	cfg := &Config{

		LPUrls: []url.URL{*lpurl},
		LCUrls: []url.URL{*lcurl},
		APUrls: []url.URL{*apurl},
		ACUrls: []url.URL{*acurl},
	}

	return cfg

}
