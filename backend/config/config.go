package config

import (
	"errors"
	"fmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/sethvargo/go-limiter/httplimit"
	"log"
	"strings"
	"time"
)

// Config is the central configuration type
type Config struct {
	Server      Server           `yaml:"server" json:"server" koanf:"server"`
	Webauthn    WebauthnSettings `yaml:"webauthn" json:"webauthn" koanf:"webauthn"`
	Passcode    Passcode         `yaml:"passcode" json:"passcode" koanf:"passcode"`
	Password    Password         `yaml:"password" json:"password" koanf:"password"`
	Database    Database         `yaml:"database" json:"database" koanf:"database"`
	Secrets     Secrets          `yaml:"secrets" json:"secrets" koanf:"secrets"`
	Service     Service          `yaml:"service" json:"service" koanf:"service"`
	Session     Session          `yaml:"session" json:"session" koanf:"session"`
	AuditLog    AuditLog         `yaml:"audit_log" json:"audit_log" koanf:"audit_log"`
	Emails      Emails           `yaml:"emails" json:"emails" koanf:"emails"`
	RateLimiter RateLimiter      `yaml:"rate_limiter" json:"rate_limiter" koanf:"rate_limiter"`
}

func Load(cfgFile *string) (*Config, error) {
	k := koanf.New(".")
	var err error
	if cfgFile == nil || *cfgFile == "" {
		*cfgFile = "./config/config.yaml"
	}
	if err = k.Load(file.Provider(*cfgFile), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("failed to load config from: %s: %w", *cfgFile, err)
	} else {
		log.Println("Using config file:", *cfgFile)
	}

	err = k.Load(env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from env vars: %w", err)
	}

	c := DefaultConfig()
	err = k.Unmarshal("", c)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return c, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: Server{
			Public: ServerSettings{
				Address: ":8000",
				Cors: Cors{
					ExposeHeaders: []string{
						httplimit.HeaderRateLimitLimit,
						httplimit.HeaderRateLimitRemaining,
						httplimit.HeaderRateLimitReset,
						httplimit.HeaderRetryAfter,
					},
				},
			},
			Admin: ServerSettings{
				Address: ":8001",
			},
		},
		Webauthn: WebauthnSettings{
			RelyingParty: RelyingParty{
				Id:          "localhost",
				DisplayName: "Hanko Authentication Service",
				Origins:     []string{"http://localhost"},
			},
			Timeout: 60000,
		},
		Passcode: Passcode{
			Smtp: SMTP{
				Port: "465",
			},
			TTL: 300,
			Email: Email{
				FromAddress: "passcode@hanko.io",
				FromName:    "Hanko",
			},
		},
		Password: Password{
			MinPasswordLength: 8,
		},
		Database: Database{
			Database: "hanko",
		},
		Session: Session{
			Lifespan: "1h",
			Cookie: Cookie{
				HttpOnly: true,
				SameSite: "strict",
				Secure:   true,
			},
		},
		AuditLog: AuditLog{
			ConsoleOutput: AuditLogConsole{
				Enabled:      true,
				OutputStream: OutputStreamStdOut,
			},
		},
		Emails: Emails{
			RequireVerification: true,
			MaxNumOfAddresses:   5,
		},
		RateLimiter: RateLimiter{
			Enabled: true,
			Store:   RATE_LIMITER_STORE_IN_MEMORY,
			PasswordLimits: RateLimits{
				Tokens:   5,
				Interval: 1 * time.Minute,
			},
			PasscodeLimits: RateLimits{
				Tokens:   3,
				Interval: 1 * time.Minute,
			},
		},
	}
}

func (c *Config) Validate() error {
	err := c.Server.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate server settings: %w", err)
	}
	err = c.Webauthn.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate webauthn settings: %w", err)
	}
	err = c.Passcode.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate passcode settings: %w", err)
	}
	err = c.Database.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate database settings: %w", err)
	}
	err = c.Secrets.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate secrets: %w", err)
	}
	err = c.Service.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate service settings: %w", err)
	}
	err = c.Session.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate session settings: %w", err)
	}
	err = c.RateLimiter.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate rate-limiter settings: %w", err)
	}
	return nil
}

// Server contains the setting for the public and admin server
type Server struct {
	Public ServerSettings `yaml:"public" json:"public" koanf:"public"`
	Admin  ServerSettings `yaml:"admin" json:"admin" koanf:"admin"`
}

func (s *Server) Validate() error {
	err := s.Public.Validate()
	if err != nil {
		return fmt.Errorf("error validating public server settings: %w", err)
	}
	err = s.Admin.Validate()
	if err != nil {
		return fmt.Errorf("error validating admin server settings: %w", err)
	}
	return nil
}

type Service struct {
	Name string `yaml:"name" json:"name" koanf:"name"`
}

func (s *Service) Validate() error {
	if len(strings.TrimSpace(s.Name)) == 0 {
		return errors.New("field name must not be empty")
	}
	return nil
}

type Password struct {
	Enabled           bool `yaml:"enabled" json:"enabled" koanf:"enabled"`
	MinPasswordLength int  `yaml:"min_password_length" json:"min_password_length" koanf:"min_password_length"`
}

type Cookie struct {
	Domain   string `yaml:"domain" json:"domain" koanf:"domain"`
	HttpOnly bool   `yaml:"http_only" json:"http_only" koanf:"http_only"`
	SameSite string `yaml:"same_site" json:"same_site" koanf:"same_site"`
	Secure   bool   `yaml:"secure" json:"secure" koanf:"secure"`
}

type ServerSettings struct {
	// The Address to listen on in the form of host:port
	// See net.Dial for details of the address format.
	Address string `yaml:"address" json:"address" koanf:"address"`
	Cors    Cors   `yaml:"cors" json:"cors" koanf:"cors"`
}

type Cors struct {
	Enabled          bool     `yaml:"enabled" json:"enabled" koanf:"enabled"`
	AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials" koanf:"allow_credentials"`
	AllowOrigins     []string `yaml:"allow_origins" json:"allow_origins" koanf:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods" json:"allow_methods" koanf:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers" json:"allow_headers" koanf:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers" json:"expose_headers" koanf:"expose_headers"`
	MaxAge           int      `yaml:"max_age" json:"max_age" koanf:"max_age"`
}

func (s *ServerSettings) Validate() error {
	if len(strings.TrimSpace(s.Address)) == 0 {
		return errors.New("field Address must not be empty")
	}
	return nil
}

// WebauthnSettings defines the settings for the webauthn authentication mechanism
type WebauthnSettings struct {
	RelyingParty RelyingParty `yaml:"relying_party" json:"relying_party" koanf:"relying_party"`
	Timeout      int          `yaml:"timeout" json:"timeout" koanf:"timeout"`
}

// Validate does not need to validate the config, because the library does this already
func (r *WebauthnSettings) Validate() error {
	return nil
}

// RelyingParty webauthn settings for your application using hanko.
type RelyingParty struct {
	Id          string `yaml:"id" json:"id" koanf:"id"`
	DisplayName string `yaml:"display_name" json:"display_name" koanf:"display_name"`
	Icon        string `yaml:"icon" json:"icon" koanf:"icon"`
	// Deprecated: Use Origins instead
	Origin  string   `yaml:"origin" json:"origin" koanf:"origin"`
	Origins []string `yaml:"origins" json:"origins" koanf:"origins"`
}

// SMTP Server Settings for sending passcodes
type SMTP struct {
	Host     string `yaml:"host" json:"host" koanf:"host"`
	Port     string `yaml:"port" json:"port" koanf:"port"`
	User     string `yaml:"user" json:"user" koanf:"user"`
	Password string `yaml:"password" json:"password" koanf:"password"`
}

func (s *SMTP) Validate() error {
	if len(strings.TrimSpace(s.Host)) == 0 {
		return errors.New("smtp host must not be empty")
	}
	if len(strings.TrimSpace(s.Port)) == 0 {
		return errors.New("smtp port must not be empty")
	}
	return nil
}

type Email struct {
	FromAddress string `yaml:"from_address" json:"from_address" koanf:"from_address"`
	FromName    string `yaml:"from_name" json:"from_name" koanf:"from_name"`
}

func (e *Email) Validate() error {
	if len(strings.TrimSpace(e.FromAddress)) == 0 {
		return errors.New("from_address must not be empty")
	}
	return nil
}

type Passcode struct {
	Email Email `yaml:"email" json:"email" koanf:"email"`
	Smtp  SMTP  `yaml:"smtp" json:"smtp" koanf:"smtp"`
	TTL   int   `yaml:"ttl" json:"ttl" koanf:"ttl"`
}

func (p *Passcode) Validate() error {
	err := p.Email.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate email settings: %w", err)
	}
	err = p.Smtp.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate smtp settings: %w", err)
	}
	return nil
}

// Database connection settings
type Database struct {
	Database string `yaml:"database" json:"database" koanf:"database"`
	User     string `yaml:"user" json:"user" koanf:"user"`
	Password string `yaml:"password" json:"password" koanf:"password"`
	Host     string `yaml:"host" json:"host" koanf:"host"`
	Port     string `yaml:"port" json:"port" koanf:"port"`
	Dialect  string `yaml:"dialect" json:"dialect" koanf:"dialect"`
	Url      string `yaml:"url" json:"url" koanf:"url"`
}

func (d *Database) Validate() error {
	if len(strings.TrimSpace(d.Url)) > 0 {
		return nil
	}
	if len(strings.TrimSpace(d.Database)) == 0 {
		return errors.New("database must not be empty")
	}
	if len(strings.TrimSpace(d.User)) == 0 {
		return errors.New("user must not be empty")
	}
	if len(strings.TrimSpace(d.Host)) == 0 {
		return errors.New("host must not be empty")
	}
	if len(strings.TrimSpace(d.Port)) == 0 {
		return errors.New("port must not be empty")
	}
	if len(strings.TrimSpace(d.Dialect)) == 0 {
		return errors.New("dialect must not be empty")
	}
	return nil
}

type Secrets struct {
	// Keys secrets are used to en- and decrypt the JWKs which get used to sign the JWTs.
	// For every key a JWK is generated, encrypted with the key and persisted in the database.
	//
	// You can use this list for key rotation: add a new key to the beginning of the list and the corresponding
	// JWK will then be used for signing JWTs. All tokens signed with the previous JWK(s) will still
	// be valid until they expire. Removing a key from the list does not remove the corresponding
	// database record. If you remove a key, you also have to remove the database record, otherwise
	// application startup will fail.
	//
	// Each key must be at least 16 characters long.
	Keys []string `yaml:"keys" json:"keys" koanf:"keys"`
}

func (s *Secrets) Validate() error {
	if len(s.Keys) == 0 {
		return errors.New("at least one key must be defined")
	}
	return nil
}

type Session struct {
	EnableAuthTokenHeader bool   `yaml:"enable_auth_token_header" json:"enable_auth_token_header" koanf:"enable_auth_token_header"`
	Lifespan              string `yaml:"lifespan" json:"lifespan" koanf:"lifespan"`
	Cookie                Cookie `yaml:"cookie" json:"cookie" koanf:"cookie"`
}

func (s *Session) Validate() error {
	_, err := time.ParseDuration(s.Lifespan)
	if err != nil {
		return errors.New("failed to parse lifespan")
	}
	return nil
}

type AuditLog struct {
	ConsoleOutput AuditLogConsole `yaml:"console_output" json:"console_output" koanf:"console_output"`
	Storage       AuditLogStorage `yaml:"storage" json:"storage" koanf:"storage"`
}

type AuditLogStorage struct {
	Enabled bool `yaml:"enabled" json:"enabled" koanf:"enabled"`
}

type AuditLogConsole struct {
	Enabled      bool         `yaml:"enabled" json:"enabled" koanf:"enabled"`
	OutputStream OutputStream `yaml:"output" json:"output" koanf:"output"`
}

type Emails struct {
	RequireVerification bool `yaml:"require_verification" json:"require_verification" koanf:"require_verification"`
	MaxNumOfAddresses   int  `yaml:"max_num_of_addresses" json:"max_num_of_addresses" koanf:"max_num_of_addresses"`
}

type OutputStream string

var (
	OutputStreamStdOut OutputStream = "stdout"
	OutputStreamStdErr OutputStream = "stderr"
)

type RateLimiter struct {
	Enabled        bool                 `yaml:"enabled" json:"enabled" koanf:"enabled"`
	Store          RateLimiterStoreType `yaml:"store" json:"store" koanf:"store"`
	Redis          *RedisConfig         `yaml:"redis_config" json:"redis_config" koanf:"redis_config"`
	PasscodeLimits RateLimits           `yaml:"passcode_limits" json:"passcode_limits" koanf:"passcode_limits"`
	PasswordLimits RateLimits           `yaml:"password_limits" json:"password_limits" koanf:"password_limits"`
}

type RateLimits struct {
	Tokens   uint64        `yaml:"tokens" json:"tokens" koanf:"tokens"`
	Interval time.Duration `yaml:"interval" json:"interval" koanf:"interval"`
}

type RateLimiterStoreType string

const (
	RATE_LIMITER_STORE_IN_MEMORY RateLimiterStoreType = "in_memory"
	RATE_LIMITER_STORE_REDIS                          = "redis"
)

func (r *RateLimiter) Validate() error {
	if r.Enabled {
		switch r.Store {
		case RATE_LIMITER_STORE_REDIS:
			if r.Redis == nil {
				return errors.New("when enabling the redis store you have to specify the redis config")
			}
			if r.Redis.Address == "" {
				return errors.New("when enabling the redis store you have to specify the address where hanko can reach the redis instance")
			}
		case RATE_LIMITER_STORE_IN_MEMORY:
			break
		default:
			return errors.New(string(r.Store) + " is not a valid rate limiter store.")
		}
	}
	return nil
}

type RedisConfig struct {
	//Address of redis in the form of host[:port][/database]
	Address  string `yaml:"address" json:"address" koanf:"address"`
	Password string `yaml:"password" json:"password" koanf:"password"`
}
