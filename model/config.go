package model

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
)

const (
	DATABASE_DRIVER_MYSQL            = "mysql"
	SQL_SETTINGS_DEFAULT_DATA_SOURCE = "root:@tcp(localhost:3306)/qa_discussion?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"

	CONN_SECURITY_NONE = ""
	CONN_SECURITY_TLS  = "TLS"

	SERVICE_SETTINGS_DEFAULT_SITE_URL           = "http://localhost:8080"
	SERVICE_SETTINGS_DEFAULT_LISTEN_AND_ADDRESS = ":8080"
	SERVICE_SETTINGS_DEFAULT_ALLOW_CORS_FROM    = ""

	SERVICE_SETTINGS_DEFAULT_READ_TIMEOUT       = 300
	SERVICE_SETTINGS_DEFAULT_WRITE_TIMEOUT      = 300
	SERVICE_SETTINGS_DEFAULT_MAX_LOGIN_ATTEMPTS = 10
	SERVICE_SETTINGS_DEFAULT_TLS_CERT_FILE      = ""
	SERVICE_SETTINGS_DEFAULT_TLS_KEY_FILE       = ""

	PASSWORD_MAXIMUM_LENGTH = 64
	PASSWORD_MINIMUM_LENGTH = 8

	TEAM_SETTINGS_DEFAULT_MAX_USERS_PER_TEAM = 50

	CACHE_SETTINGS_DEFAULT_ENDPOINT  = "http://localhost:6379"
	SEARCH_SETTINGS_DEFAULT_ENDPOINT = "http://localhost:9200"
)

type ServiceSettings struct {
	SiteURL                             *string
	ListenAddress                       *string
	ConnectionSecurity                  *string
	TLSCertFile                         *string
	TLSKeyFile                          *string
	ReadTimeout                         *int
	WriteTimeout                        *int
	TLSMinVer                           *string
	EnableDeveloper                     *bool
	EnableAdminUser                     *bool
	UseLetsEncrypt                      *bool
	LetsEncryptCertificateCacheFile     *string
	MaximumLoginAttempts                *int
	Forward80To443                      *bool
	WebserverMode                       *string `restricted:"true"`
	SessionLengthWebInDays              *int
	SessionLengthOAuthInDays            *int `restricted:"true"`
	TrustedProxyIPHeader                []string
	AllowedUntrustedInternalConnections *string `restricted:"true"`

	AllowCorsFrom        *string
	CorsExposedHeaders   *string
	CorsAllowCredentials *bool
	CorsDebug            *bool
}

func (s *ServiceSettings) SetDefaults() {
	if s.SiteURL == nil {
		if s.EnableDeveloper != nil && *s.EnableDeveloper {
			s.SiteURL = NewString(SERVICE_SETTINGS_DEFAULT_SITE_URL)
		} else {
			s.SiteURL = NewString("")
		}
	}

	if s.ListenAddress == nil {
		s.ListenAddress = NewString(SERVICE_SETTINGS_DEFAULT_LISTEN_AND_ADDRESS)
	}

	if s.EnableDeveloper == nil {
		s.EnableDeveloper = NewBool(false)
	}

	if s.EnableAdminUser == nil {
		s.EnableAdminUser = NewBool(false)
	}

	if s.TLSKeyFile == nil {
		s.TLSKeyFile = NewString(SERVICE_SETTINGS_DEFAULT_TLS_KEY_FILE)
	}

	if s.TLSCertFile == nil {
		s.TLSCertFile = NewString(SERVICE_SETTINGS_DEFAULT_TLS_CERT_FILE)
	}

	if s.TLSMinVer == nil {
		s.TLSMinVer = NewString("1.3")
	}

	if s.UseLetsEncrypt == nil {
		s.UseLetsEncrypt = NewBool(false)
	}

	if s.LetsEncryptCertificateCacheFile == nil {
		s.LetsEncryptCertificateCacheFile = NewString("./config/letsencrypt.cache")
	}

	if s.ConnectionSecurity == nil {
		s.ConnectionSecurity = NewString("")
	}

	if s.ReadTimeout == nil {
		s.ReadTimeout = NewInt(SERVICE_SETTINGS_DEFAULT_READ_TIMEOUT)
	}

	if s.WriteTimeout == nil {
		s.WriteTimeout = NewInt(SERVICE_SETTINGS_DEFAULT_WRITE_TIMEOUT)
	}

	if s.MaximumLoginAttempts == nil {
		s.MaximumLoginAttempts = NewInt(SERVICE_SETTINGS_DEFAULT_MAX_LOGIN_ATTEMPTS)
	}

	if s.Forward80To443 == nil {
		s.Forward80To443 = NewBool(false)
	}

	if s.WebserverMode == nil {
		s.WebserverMode = NewString("gzip")
	} else if *s.WebserverMode == "regular" {
		*s.WebserverMode = "gzip"
	}

	if s.SessionLengthWebInDays == nil {
		s.SessionLengthWebInDays = NewInt(180)
	}

	if s.SessionLengthOAuthInDays == nil {
		s.SessionLengthOAuthInDays = NewInt(30)
	}

	if s.TrustedProxyIPHeader == nil {
		s.TrustedProxyIPHeader = []string{HEADER_FORWARDED, HEADER_REAL_IP}
	}

	if s.AllowedUntrustedInternalConnections == nil {
		s.AllowedUntrustedInternalConnections = NewString("")
	}

	if s.AllowCorsFrom == nil {
		s.AllowCorsFrom = NewString(SERVICE_SETTINGS_DEFAULT_ALLOW_CORS_FROM)
	}

	if s.CorsExposedHeaders == nil {
		s.CorsExposedHeaders = NewString("")
	}

	if s.CorsAllowCredentials == nil {
		s.CorsAllowCredentials = NewBool(false)
	}

	if s.CorsDebug == nil {
		s.CorsDebug = NewBool(false)
	}
}

func (ss *ServiceSettings) isValid() *AppError {
	if !(*ss.ConnectionSecurity == CONN_SECURITY_NONE || *ss.ConnectionSecurity == CONN_SECURITY_TLS) {
		return NewAppError("Config.IsValid", "model.config.is_valid.webserver_security.app_error", nil, "", http.StatusBadRequest)
	}

	if *ss.ConnectionSecurity == CONN_SECURITY_TLS && !*ss.UseLetsEncrypt {
		appErr := NewAppError("Config.IsValid", "model.config.is_valid.tls_cert_file.app_error", nil, "", http.StatusBadRequest)

		if *ss.TLSCertFile == "" {
			return appErr
		} else if _, err := os.Stat(*ss.TLSCertFile); os.IsNotExist(err) {
			return appErr
		}

		appErr = NewAppError("Config.IsValid", "model.config.is_valid.tls_key_file.app_error", nil, "", http.StatusBadRequest)

		if *ss.TLSKeyFile == "" {
			return appErr
		} else if _, err := os.Stat(*ss.TLSKeyFile); os.IsNotExist(err) {
			return appErr
		}
	}

	if *ss.ReadTimeout <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.read_timeout.app_error", nil, "", http.StatusBadRequest)
	}

	if *ss.WriteTimeout <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.write_timeout.app_error", nil, "", http.StatusBadRequest)
	}

	if *ss.MaximumLoginAttempts <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.login_attempts.app_error", nil, "", http.StatusBadRequest)
	}

	host, port, _ := net.SplitHostPort(*ss.ListenAddress)
	var isValidHost bool
	if host == "" {
		isValidHost = true
	} else {
		isValidHost = (net.ParseIP(host) != nil) || IsDomainName(host)
	}
	portInt, err := strconv.Atoi(port)
	if err != nil || !isValidHost || portInt < 0 || portInt > math.MaxUint16 {
		return NewAppError("Config.IsValid", "model.config.is_valid.listen_address.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

type TeamSettings struct {
	MaxUsersPerTeam       *int
	MaxGroupsPerTeam      *int64
	MaxCollectionsPerTeam *int64
}

func (s *TeamSettings) SetDefaults() {
	if s.MaxUsersPerTeam == nil {
		s.MaxUsersPerTeam = NewInt(TEAM_SETTINGS_DEFAULT_MAX_USERS_PER_TEAM)
	}

	if s.MaxGroupsPerTeam == nil {
		s.MaxGroupsPerTeam = NewInt64(100)
	}

	if s.MaxCollectionsPerTeam == nil {
		s.MaxCollectionsPerTeam = NewInt64(100)
	}
}

func (s *TeamSettings) isValid() *AppError {
	if *s.MaxUsersPerTeam <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.max_users.app_error", nil, "", http.StatusBadRequest)
	}

	if *s.MaxGroupsPerTeam <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.max_groups.app_error", nil, "", http.StatusBadRequest)
	}

	if *s.MaxCollectionsPerTeam <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.max_collections.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

type SqlSettings struct {
	DriverName                  *string
	DataSource                  *string
	DataSourceReplicas          []string
	MaxIdleConns                *int
	ConnMaxLifetimeMilliseconds *int
	MaxOpenConns                *int
	Trace                       *bool
	QueryTimeout                *int
}

func (s *SqlSettings) SetDefaults() {
	if s.DriverName == nil {
		s.DriverName = NewString(DATABASE_DRIVER_MYSQL)
	}

	if s.DataSource == nil {
		s.DataSource = NewString(SQL_SETTINGS_DEFAULT_DATA_SOURCE)
	}

	if s.DataSourceReplicas == nil {
		s.DataSourceReplicas = []string{}
	}

	if s.MaxIdleConns == nil {
		s.MaxIdleConns = NewInt(20)
	}

	if s.MaxOpenConns == nil {
		s.MaxOpenConns = NewInt(300)
	}

	if s.ConnMaxLifetimeMilliseconds == nil {
		s.ConnMaxLifetimeMilliseconds = NewInt(3600000)
	}

	if s.Trace == nil {
		s.Trace = NewBool(false)
	}

	if s.QueryTimeout == nil {
		s.QueryTimeout = NewInt(30)
	}
}

func (ss *SqlSettings) isValid() *AppError {
	if !(*ss.DriverName == DATABASE_DRIVER_MYSQL) {
		return NewAppError("Config.IsValid", "model.config.is_valid.sql_driver.app_error", nil, "", http.StatusBadRequest)
	}

	if *ss.MaxIdleConns <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.sql_idle.app_error", nil, "", http.StatusBadRequest)
	}

	if *ss.ConnMaxLifetimeMilliseconds < 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.sql_conn_max_lifetime_milliseconds.app_error", nil, "", http.StatusBadRequest)
	}

	if *ss.QueryTimeout <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.sql_query_timeout.app_error", nil, "", http.StatusBadRequest)
	}

	if len(*ss.DataSource) == 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.sql_data_src.app_error", nil, "", http.StatusBadRequest)
	}

	if *ss.MaxOpenConns <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.sql_max_conn.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

type CacheSettings struct {
	CacheEndpoint  *string
	CacheDefaultDb *int
}

func (s *CacheSettings) SetDefaults() {
	if s.CacheEndpoint == nil {
		s.CacheEndpoint = NewString(CACHE_SETTINGS_DEFAULT_ENDPOINT)
	}

	if s.CacheDefaultDb == nil {
		s.CacheDefaultDb = NewInt(0)
	}
}

func (s *CacheSettings) isValid() *AppError {
	return nil
}

type SearchSettings struct {
	SearchEndpoint *string
}

func (s *SearchSettings) SetDefaults() {
	if s.SearchEndpoint == nil {
		s.SearchEndpoint = NewString(SEARCH_SETTINGS_DEFAULT_ENDPOINT)
	}
}

func (s *SearchSettings) isValid() *AppError {
	return nil
}

type FileSettings struct {
	MaxFileSize             *int64
	AmazonS3AccessKeyId     *string `restricted:"true"`
	AmazonS3SecretAccessKey *string `restricted:"true"`
	AmazonS3Bucket          *string `restricted:"true"`
	AmazonS3Region          *string `restricted:"true"`
	AmazonS3Endpoint        *string `restricted:"true"`
	AmazonS3SSL             *bool   `restricted:"true"`
	AmazonS3SSE             *bool   `restricted:"true"`
	AmazonS3Trace           *bool   `restricted:"true"`
	AmazonCloudFrontURL     *string
}

func (s *FileSettings) SetDefaults() {
	if s.MaxFileSize == nil {
		s.MaxFileSize = NewInt64(52428800) // 50 MB
	}

	if s.AmazonS3AccessKeyId == nil {
		s.AmazonS3AccessKeyId = NewString("")
	}

	if s.AmazonS3SecretAccessKey == nil {
		s.AmazonS3SecretAccessKey = NewString("")
	}

	if s.AmazonS3Bucket == nil {
		s.AmazonS3Bucket = NewString("")
	}

	if s.AmazonS3Region == nil {
		s.AmazonS3Region = NewString("")
	}

	if s.AmazonS3Endpoint == nil || len(*s.AmazonS3Endpoint) == 0 {
		s.AmazonS3Endpoint = NewString("s3.amazonaws.com")
	}

	if s.AmazonS3SSL == nil {
		s.AmazonS3SSL = NewBool(true) // Secure by default.
	}

	if s.AmazonS3SSE == nil {
		s.AmazonS3SSE = NewBool(false) // Not Encrypted by default.
	}

	if s.AmazonS3Trace == nil {
		s.AmazonS3Trace = NewBool(false)
	}

	if s.AmazonCloudFrontURL == nil {
		s.AmazonCloudFrontURL = NewString("")
	}
}

func (s *FileSettings) isValid() *AppError {
	if *s.MaxFileSize <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.max_file_size.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

type PasswordSettings struct {
	MinimumLength *int
	Lowercase     *bool
	Number        *bool
	Uppsercase    *bool
	Symbol        *bool
}

func (s *PasswordSettings) SetDefaults() {
	if s.MinimumLength == nil {
		s.MinimumLength = NewInt(10)
	}

	if s.Lowercase == nil {
		s.Lowercase = NewBool(true)
	}

	if s.Number == nil {
		s.Number = NewBool(true)
	}

	if s.Uppsercase == nil {
		s.Uppsercase = NewBool(true)
	}

	if s.Symbol == nil {
		s.Symbol = NewBool(true)
	}
}

type RateLimitSettings struct {
	Enable           *bool `restricted:"true"`
	PerSec           *int  `restricted:"true"`
	MaxBurst         *int  `restricted:"true"`
	MemoryStoreSize  *int  `restricted:"true"`
	VaryByRemoteAddr *bool `restricted:"true"`
	VaryByUser       *bool `restricted:"true"`
}

func (s *RateLimitSettings) SetDefaults() {
	if s.Enable == nil {
		s.Enable = NewBool(true)
	}

	if s.PerSec == nil {
		s.PerSec = NewInt(10)
	}

	if s.MaxBurst == nil {
		s.MaxBurst = NewInt(100)
	}

	if s.MemoryStoreSize == nil {
		s.MemoryStoreSize = NewInt(10000)
	}

	if s.VaryByRemoteAddr == nil {
		s.VaryByRemoteAddr = NewBool(true)
	}

	if s.VaryByUser == nil {
		s.VaryByUser = NewBool(true)
	}
}

func (rls *RateLimitSettings) isValid() *AppError {
	if *rls.MemoryStoreSize <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.rate_mem.app_error", nil, "", http.StatusBadRequest)
	}

	if *rls.PerSec <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.rate_sec.app_error", nil, "", http.StatusBadRequest)
	}

	if *rls.MaxBurst <= 0 {
		return NewAppError("Config.IsValid", "model.config.is_valid.max_burst.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

type EmailSettings struct {
	AmazonSESAccessKeyId     *string
	AmazonSESSecretAccessKey *string
	AmazonSESRegion          *string
	SupportEmail             *string
}

func (s *EmailSettings) SetDefaults() {
	if s.AmazonSESAccessKeyId == nil {
		s.AmazonSESAccessKeyId = NewString("")
	}

	if s.AmazonSESSecretAccessKey == nil {
		s.AmazonSESSecretAccessKey = NewString("")
	}

	if s.AmazonSESRegion == nil {
		s.AmazonSESRegion = NewString("")
	}

	if s.SupportEmail == nil {
		s.SupportEmail = NewString("")
	}
}

func (s *EmailSettings) isValid() *AppError {
	return nil
}

type EmailBatchJobSettings struct {
	Enable      *bool
	ThreeHourly *bool
	Daily       *bool
	Weekly      *bool
}

func (s *EmailBatchJobSettings) SetDefaults() {
	if s.Enable == nil {
		s.Enable = NewBool(false)
	}

	if s.ThreeHourly == nil {
		s.ThreeHourly = NewBool(false)
	}

	if s.Daily == nil {
		s.Daily = NewBool(false)
	}

	if s.Weekly == nil {
		s.Weekly = NewBool(false)
	}
}

func (s *EmailBatchJobSettings) isValid() *AppError {
	if *s.Enable && !*s.ThreeHourly && !*s.Daily && !*s.Weekly {
		return NewAppError("Config.IsValid", "model.config.is_valid.email_batch_job_enable.app_error", nil, "", http.StatusBadRequest)
	}

	return nil
}

type Config struct {
	ServiceSettings       ServiceSettings
	SqlSettings           SqlSettings
	CacheSettings         CacheSettings
	SearchSettings        SearchSettings
	TeamSettings          TeamSettings
	FileSettings          FileSettings
	PasswordSettings      PasswordSettings
	RateLimitSettings     RateLimitSettings
	EmailBatchJobSettings EmailBatchJobSettings
	EmailSettings         EmailSettings
}

func (o *Config) ToJson() string {
	b, _ := json.Marshal(o)
	return string(b)
}

func (o *Config) Clone() *Config {
	var ret Config
	if err := json.Unmarshal([]byte(o.ToJson()), &ret); err != nil {
		panic(err)
	}
	return &ret
}

func (o *Config) SetDefaults() {
	o.SqlSettings.SetDefaults()
	o.CacheSettings.SetDefaults()
	o.SearchSettings.SetDefaults()
	o.TeamSettings.SetDefaults()
	o.ServiceSettings.SetDefaults()
	o.FileSettings.SetDefaults()
	o.PasswordSettings.SetDefaults()
	o.RateLimitSettings.SetDefaults()
	o.EmailBatchJobSettings.SetDefaults()
	o.EmailSettings.SetDefaults()
}

func (o *Config) IsValid() *AppError {
	if err := o.SqlSettings.isValid(); err != nil {
		return err
	}

	if err := o.CacheSettings.isValid(); err != nil {
		return err
	}

	if err := o.SearchSettings.isValid(); err != nil {
		return err
	}

	if err := o.TeamSettings.isValid(); err != nil {
		return err
	}

	if err := o.ServiceSettings.isValid(); err != nil {
		return err
	}

	if err := o.FileSettings.isValid(); err != nil {
		return err
	}

	if *o.PasswordSettings.MinimumLength < PASSWORD_MINIMUM_LENGTH || *o.PasswordSettings.MinimumLength > PASSWORD_MAXIMUM_LENGTH {
		return NewAppError("Config.IsValid", "model.config.is_valid.password_length.app_error", map[string]interface{}{"MinLength": PASSWORD_MINIMUM_LENGTH, "MaxLength": PASSWORD_MAXIMUM_LENGTH}, "", http.StatusBadRequest)
	}

	if err := o.RateLimitSettings.isValid(); err != nil {
		return err
	}

	if err := o.EmailBatchJobSettings.isValid(); err != nil {
		return err
	}

	if err := o.EmailSettings.isValid(); err != nil {
		return err
	}

	return nil
}
