package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Output    OutputConfig    `mapstructure:"output"`
	Fetch     FetchConfig     `mapstructure:"fetch"`
	Filter    FilterConfig    `mapstructure:"filter"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Auth      AuthConfig      `mapstructure:"auth"`
	HTTP      HTTPConfig      `mapstructure:"http"`
	Browser   BrowserConfig   `mapstructure:"browser"`
}

type OutputConfig struct {
	Format string `mapstructure:"format"`
}

type FetchConfig struct {
	Count int `mapstructure:"count"`
}

type FilterConfig struct {
	Mode            string        `mapstructure:"mode"`
	TopN            int           `mapstructure:"top_n"`
	MinScore        float64       `mapstructure:"min_score"`
	ExcludeRetweets bool          `mapstructure:"exclude_retweets"`
	Weights         FilterWeights `mapstructure:"weights"`
}

type FilterWeights struct {
	Likes     float64 `mapstructure:"likes"`
	Retweets  float64 `mapstructure:"retweets"`
	Replies   float64 `mapstructure:"replies"`
	Bookmarks float64 `mapstructure:"bookmarks"`
	ViewsLog  float64 `mapstructure:"views_log"`
}

type RateLimitConfig struct {
	RequestDelay   time.Duration `mapstructure:"request_delay"`
	MaxRetries     int           `mapstructure:"max_retries"`
	RetryBaseDelay time.Duration `mapstructure:"retry_base_delay"`
	MaxCount       int           `mapstructure:"max_count"`
}

type AuthConfig struct {
	Source string `mapstructure:"source"`
	Token  string `mapstructure:"token"`
	CT0    string `mapstructure:"ct0"`
}

type HTTPConfig struct {
	GraphQLBaseURL string `mapstructure:"graphql_base_url"`
	Proxy          string `mapstructure:"proxy"`
	UserAgent      string `mapstructure:"user_agent"`
}

type BrowserConfig struct {
	RemoteDebugURL string `mapstructure:"remote_debug_url"`
	TraceTxIDFile  string `mapstructure:"trace_txid_file"`
	TraceTxIDMode  string `mapstructure:"trace_txid_mode"`
	TraceTxIDOps   string `mapstructure:"trace_txid_ops"`
	StaticSalt     string `mapstructure:"static_salt"`
}

type Flags struct {
	ConfigFile    string
	Format        string
	Verbose       bool
	Debug         bool
	AuthToken     string
	CT0           string
	Proxy         string
	TraceTxIDFile string
	TraceTxIDOps  string
}

const (
	DefaultConfigName = "config"
	DefaultConfigType = "yaml"
	DefaultFormat     = "table"
	DefaultFetchCount = 20
	DefaultMaxCount   = 200
)

func Load(flags Flags) (*Config, error) {
	v := viper.New()

	v.SetDefault("output.format", DefaultFormat)
	v.SetDefault("fetch.count", DefaultFetchCount)
	v.SetDefault("filter.mode", "topN")
	v.SetDefault("filter.top_n", 20)
	v.SetDefault("filter.min_score", 50)
	v.SetDefault("filter.exclude_retweets", false)
	v.SetDefault("filter.weights.likes", 1.0)
	v.SetDefault("filter.weights.retweets", 3.0)
	v.SetDefault("filter.weights.replies", 2.0)
	v.SetDefault("filter.weights.bookmarks", 5.0)
	v.SetDefault("filter.weights.views_log", 0.5)
	v.SetDefault("rate_limit.request_delay", "2500ms")
	v.SetDefault("rate_limit.max_retries", 3)
	v.SetDefault("rate_limit.retry_base_delay", "5s")
	v.SetDefault("rate_limit.max_count", DefaultMaxCount)
	v.SetDefault("auth.source", "browser")
	v.SetDefault("http.graphql_base_url", "https://x.com/i/api/graphql")
	v.SetDefault("http.user_agent", "x/dev")
	v.SetDefault("browser.remote_debug_url", "")
	v.SetDefault("browser.trace_txid_file", "")
	v.SetDefault("browser.trace_txid_mode", "writes")
	v.SetDefault("browser.trace_txid_ops", "")

	if flags.ConfigFile != "" {
		v.SetConfigFile(flags.ConfigFile)
	} else {
		v.AddConfigPath(getConfigDir())
		v.SetConfigName(DefaultConfigName)
		v.SetConfigType(DefaultConfigType)
	}

	v.SetEnvPrefix("X")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	if flags.Format != "" {
		v.Set("output.format", flags.Format)
	}
	if flags.AuthToken != "" {
		v.Set("auth.token", flags.AuthToken)
	}
	if flags.CT0 != "" {
		v.Set("auth.ct0", flags.CT0)
	}
	if flags.Proxy != "" {
		v.Set("http.proxy", flags.Proxy)
	}
	if flags.TraceTxIDFile != "" {
		v.Set("browser.trace_txid_file", flags.TraceTxIDFile)
	}
	if flags.TraceTxIDOps != "" {
		v.Set("browser.trace_txid_ops", flags.TraceTxIDOps)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.Fetch.Count <= 0 {
		cfg.Fetch.Count = DefaultFetchCount
	}
	if cfg.RateLimit.MaxCount <= 0 {
		cfg.RateLimit.MaxCount = DefaultMaxCount
	}
	if cfg.Output.Format == "" {
		cfg.Output.Format = DefaultFormat
	}

	return &cfg, nil
}

func getConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "x")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	return filepath.Join(home, ".config", "x")
}
