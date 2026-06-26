package config

import (
	"bytes"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// expandEnv expands ${VAR} and ${VAR:-default} in s using os.Expand.
func expandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		if idx := strings.Index(key, ":-"); idx >= 0 {
			varName, defaultVal := key[:idx], key[idx+2:]
			if val, ok := os.LookupEnv(varName); ok && val != "" {
				return val
			}
			return defaultVal
		}
		return os.Getenv(key)
	})
}

type ServiceConfig struct {
	Name         string `mapstructure:"name"           validate:"required"`
	Profile      string `mapstructure:"profile"        validate:"required,oneof=internal external"`
	Version      string `mapstructure:"version"`
	BaseURL      string `mapstructure:"base_url"`      // e.g. http://localhost:8090 (used for OIDC discovery)
	FrontendURL  string `mapstructure:"frontend_url"`  // e.g. http://localhost:3008 (auth7-ui, used for email links)
	DefaultOrgID string `mapstructure:"default_org_id"` // UUID of default org for DCR without auth context
}

type ServerConfig struct {
	Port                int64         `mapstructure:"port"                   validate:"required,min=1,max=65535"`
	RequestTimeout      time.Duration `mapstructure:"request_timeout"`
	ReadTimeout         time.Duration `mapstructure:"read_timeout"`
	WriteTimeout        time.Duration `mapstructure:"write_timeout"`
	IdleTimeout         time.Duration `mapstructure:"idle_timeout"`
	MaxRequestBodyBytes int64         `mapstructure:"max_request_body_bytes"`
}

type DatabaseConfig struct {
	Primary DatabasePoolConfig `mapstructure:"primary" validate:"required"`
	Replica ReplicaConfig      `mapstructure:"replica"`
}

type DatabasePoolConfig struct {
	DSN               string        `mapstructure:"dsn"                 validate:"required"`
	MaxConns          int32         `mapstructure:"max_conns"`
	MinConns          int32         `mapstructure:"min_conns"`
	MaxConnLifetime   time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period"`
	ConnectTimeout    time.Duration `mapstructure:"connect_timeout"`
}

type ReplicaConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	DSN     string `mapstructure:"dsn"`
}

type RedisConfig struct {
	DSN            string        `mapstructure:"dsn"            validate:"required"`
	PoolSize       int           `mapstructure:"pool_size"`
	MinIdleConns   int           `mapstructure:"min_idle_conns"`
	MaxRetries     int           `mapstructure:"max_retries"`
	DialTimeout    time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	KeepAlive      time.Duration `mapstructure:"keep_alive"`
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
}

type TokenConfig struct {
	Type     string        `mapstructure:"type"     validate:"required,oneof=jwt paseto"`
	Secret   string        `mapstructure:"secret"   validate:"required"`
	Duration time.Duration `mapstructure:"duration"`
}

type CasbinConfig struct {
	ModelPath             string        `mapstructure:"model_path"`
	PolicyRefreshInterval time.Duration `mapstructure:"policy_refresh_interval"`
}

type SecurityConfig struct {
	CSP  string     `mapstructure:"csp"`
	CORS CORSConfig `mapstructure:"cors"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	MaxAge         int      `mapstructure:"max_age"`
}

type RateLimitConfig struct {
	Enabled bool       `mapstructure:"enabled"`
	Backend string     `mapstructure:"backend" validate:"omitempty,oneof=memory redis"`
	PerIP   RateConfig `mapstructure:"per_ip"`
	PerUser RateConfig `mapstructure:"per_user"`
}

type RateConfig struct {
	Requests int           `mapstructure:"requests"`
	Window   time.Duration `mapstructure:"window"`
}

type APIConfig struct {
	Metrics     MetricsAPIConfig     `mapstructure:"metrics"`
	Diagnostics DiagnosticsAPIConfig `mapstructure:"diagnostics"`
}

type MetricsAPIConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type DiagnosticsAPIConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Secret  string `mapstructure:"secret"`
}

type OTELConfig struct {
	Enabled       bool    `mapstructure:"enabled"`
	Endpoint      string  `mapstructure:"endpoint"`
	SamplingRatio float64 `mapstructure:"sampling_ratio"`
}

type LoggingConfig struct {
	Level    string `mapstructure:"level"    validate:"required,oneof=debug info warn error"`
	Pretty   bool   `mapstructure:"pretty"`
	TimeZone string `mapstructure:"timezone"`
}

type NATSConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	URL            string        `mapstructure:"url"`
	Name           string        `mapstructure:"name"`
	Username       string        `mapstructure:"username"`
	Password       string        `mapstructure:"password"`
	CredsFile      string        `mapstructure:"creds_file"`
	ReconnectWait  time.Duration `mapstructure:"reconnect_wait"`
	MaxReconnects  int           `mapstructure:"max_reconnects"`
	PublishTimeout time.Duration `mapstructure:"publish_timeout"`
	PublishRetry   int           `mapstructure:"publish_retry"`
}

type ExternalConfig struct {
	GRPC GRPCConfig `mapstructure:"grpc"`
}

type GRPCConfig struct {
	Address  string        `mapstructure:"address"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	StartTLS bool   `mapstructure:"starttls"`
}

func (c SMTPConfig) IsConfigured() bool {
	return c.Host != "" && c.Port > 0
}

type Config struct {
	Service    ServiceConfig    `mapstructure:"service"`
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Token      TokenConfig      `mapstructure:"token"`
	Casbin     CasbinConfig     `mapstructure:"casbin"`
	Security   SecurityConfig   `mapstructure:"security"`
	RateLimit  RateLimitConfig  `mapstructure:"rate_limit"`
	API        APIConfig        `mapstructure:"api"`
	OTEL       OTELConfig       `mapstructure:"otel"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	NATS       NATSConfig       `mapstructure:"nats"`
	External   ExternalConfig   `mapstructure:"external"`
	SMTP       SMTPConfig       `mapstructure:"smtp"`
	Audit7     Audit7Config     `mapstructure:"audit7"`
	Policy7    Policy7Config    `mapstructure:"policy7"`
}

// Policy7Config configures consumption of policy7 parameters as ABAC context.
// Currently used for time-based access control (operational_hours). Secrets
// (service_id, api_key) must be supplied via env (${POLICY7_*}); never inline.
type Policy7Config struct {
	// Enabled turns on time-based ABAC enforcement. When false the time-gate is
	// skipped entirely (no policy7 calls, all actions time-allowed).
	Enabled bool `mapstructure:"enabled"`
	// APIURL is the policy7 base URL, e.g. http://policy7:8080.
	APIURL string `mapstructure:"api_url"`
	// ServiceID / APIKey are the M2M credentials policy7 expects on
	// X-Service-ID / X-API-Key.
	ServiceID string `mapstructure:"service_id"`
	APIKey    string `mapstructure:"api_key"`
	// ParamName is the operational_hours parameter name to fetch (effective
	// resolution applies user->role->branch->global inside policy7). Defaults to
	// "teller_operating_hours" — the aggregate weekly-schedule shape.
	ParamName string `mapstructure:"param_name"`
	// DefaultTimezone is used when the parameter value carries no timezone.
	// Accepts IANA names (Asia/Jakarta) or Indonesian abbreviations (WIB/WITA/WIT).
	DefaultTimezone string `mapstructure:"default_timezone"`
	// TimeGatedPermissions lists the permissions denied outside operational hours.
	// Config-driven so the set is tuned without code changes.
	TimeGatedPermissions []string `mapstructure:"time_gated_permissions"`
	// FailOpen controls behaviour when policy7 is unreachable or the parameter is
	// missing/unparseable: true => allow (availability-first, default), false =>
	// deny (strict). A loud warning is logged either way.
	FailOpen bool `mapstructure:"fail_open"`
}

// ParamNameOrDefault returns the configured operational_hours param name or the
// default aggregate weekly-schedule param.
func (c Policy7Config) ParamNameOrDefault() string {
	if c.ParamName != "" {
		return c.ParamName
	}
	return "teller_operating_hours"
}

// DefaultTimezoneOrFallback returns the configured default timezone or WIB.
func (c Policy7Config) DefaultTimezoneOrFallback() string {
	if c.DefaultTimezone != "" {
		return c.DefaultTimezone
	}
	return "Asia/Jakarta"
}

// Audit7Config configures forwarding of admin/workflow audit logs to the
// central audit7 service. Empty URL disables forwarding.
type Audit7Config struct {
	URL        string `mapstructure:"url"`
	ServiceKey string `mapstructure:"service_key"`
}

func Load(configPath string) (*Config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadConfig(bytes.NewReader([]byte(expandEnv(string(raw))))); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
