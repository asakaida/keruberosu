package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestDatabaseConfig_ConnectionString(t *testing.T) {
	tests := []struct {
		name string
		cfg  DatabaseConfig
		want string
	}{
		{
			name: "standard configuration",
			cfg: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "testuser",
				Password: "testpass",
				Database: "testdb",
				SSLMode:  "disable",
			},
			want: "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable",
		},
		{
			name: "production configuration",
			cfg: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "produser",
				Password: "securepass123",
				Database: "proddb",
				SSLMode:  "require",
			},
			want: "host=db.example.com port=5433 user=produser password=securepass123 dbname=proddb sslmode=require",
		},
		{
			name: "IPv6 host",
			cfg: DatabaseConfig{
				Host:     "::1",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "db",
				SSLMode:  "disable",
			},
			want: "host=::1 port=5432 user=user password=pass dbname=db sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.ConnectionString(); got != tt.want {
				t.Errorf("DatabaseConfig.ConnectionString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{
			name:    "default dev environment",
			env:     "",
			wantErr: false,
		},
		{
			name:    "explicit dev environment",
			env:     "dev",
			wantErr: false,
		},
		{
			name:    "test environment",
			env:     "test",
			wantErr: false,
		},
		{
			name:    "prod environment",
			env:     "prod",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()

			err := InitConfig(tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify default values are set
			if !tt.wantErr {
				if viper.GetString("SERVER_HOST") != "0.0.0.0" {
					t.Errorf("InitConfig() SERVER_HOST = %v, want 0.0.0.0", viper.GetString("SERVER_HOST"))
				}
				if viper.GetInt("SERVER_PORT") != 50051 {
					t.Errorf("InitConfig() SERVER_PORT = %v, want 50051", viper.GetInt("SERVER_PORT"))
				}
				if viper.GetString("DB_HOST") != "localhost" {
					t.Errorf("InitConfig() DB_HOST = %v, want localhost", viper.GetString("DB_HOST"))
				}
				if viper.GetString("DB_USER") != "keruberosu" {
					t.Errorf("InitConfig() DB_USER = %v, want keruberosu", viper.GetString("DB_USER"))
				}
				if viper.GetString("DB_SSLMODE") != "disable" {
					t.Errorf("InitConfig() DB_SSLMODE = %v, want disable", viper.GetString("DB_SSLMODE"))
				}
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		wantErr     bool
		wantErrMsg  string
		validateCfg func(*testing.T, *Config)
	}{
		{
			name: "successful load with password",
			setupEnv: func() {
				viper.Reset()
				viper.Set("DB_PASSWORD", "testpassword")
				viper.SetDefault("SERVER_HOST", "0.0.0.0")
				viper.SetDefault("SERVER_PORT", 50051)
				viper.SetDefault("DB_HOST", "localhost")
				viper.SetDefault("DB_PORT", 15432)
				viper.SetDefault("DB_USER", "keruberosu")
				viper.SetDefault("DB_NAME", "keruberosu_dev")
				viper.SetDefault("DB_SSLMODE", "disable")
			},
			cleanupEnv: func() {
				viper.Reset()
			},
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if cfg.Server.Host != "0.0.0.0" {
					t.Errorf("Load() Server.Host = %v, want 0.0.0.0", cfg.Server.Host)
				}
				if cfg.Server.Port != 50051 {
					t.Errorf("Load() Server.Port = %v, want 50051", cfg.Server.Port)
				}
				if cfg.Database.Host != "localhost" {
					t.Errorf("Load() Database.Host = %v, want localhost", cfg.Database.Host)
				}
				if cfg.Database.Port != 15432 {
					t.Errorf("Load() Database.Port = %v, want 15432", cfg.Database.Port)
				}
				if cfg.Database.User != "keruberosu" {
					t.Errorf("Load() Database.User = %v, want keruberosu", cfg.Database.User)
				}
				if cfg.Database.Password != "testpassword" {
					t.Errorf("Load() Database.Password = %v, want testpassword", cfg.Database.Password)
				}
				if cfg.Database.Database != "keruberosu_dev" {
					t.Errorf("Load() Database.Database = %v, want keruberosu_dev", cfg.Database.Database)
				}
				if cfg.Database.SSLMode != "disable" {
					t.Errorf("Load() Database.SSLMode = %v, want disable", cfg.Database.SSLMode)
				}
			},
		},
		{
			name: "missing password",
			setupEnv: func() {
				viper.Reset()
				viper.SetDefault("SERVER_HOST", "0.0.0.0")
				viper.SetDefault("SERVER_PORT", 50051)
			},
			cleanupEnv: func() {
				viper.Reset()
			},
			wantErr:    true,
			wantErrMsg: "DB_PASSWORD is required (set via environment variable or .env file)",
		},
		{
			name: "custom server config",
			setupEnv: func() {
				viper.Reset()
				viper.Set("DB_PASSWORD", "pass123")
				viper.Set("SERVER_HOST", "custom.host")
				viper.Set("SERVER_PORT", 8080)
				viper.SetDefault("DB_HOST", "localhost")
				viper.SetDefault("DB_PORT", 15432)
				viper.SetDefault("DB_USER", "keruberosu")
				viper.SetDefault("DB_NAME", "keruberosu_dev")
				viper.SetDefault("DB_SSLMODE", "disable")
			},
			cleanupEnv: func() {
				viper.Reset()
			},
			wantErr: false,
			validateCfg: func(t *testing.T, cfg *Config) {
				if cfg.Server.Host != "custom.host" {
					t.Errorf("Load() Server.Host = %v, want custom.host", cfg.Server.Host)
				}
				if cfg.Server.Port != 8080 {
					t.Errorf("Load() Server.Port = %v, want 8080", cfg.Server.Port)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.wantErrMsg {
					t.Errorf("Load() error = %v, want %v", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if tt.validateCfg != nil {
				tt.validateCfg(t, cfg)
			}
		})
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// This test assumes we're running from within the project
	root, err := findProjectRoot()
	if err != nil {
		t.Errorf("findProjectRoot() error = %v, want nil", err)
		return
	}

	// Verify go.mod exists in the returned root
	goModPath := root + "/go.mod"
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Errorf("findProjectRoot() returned %v, but go.mod does not exist at %v", root, goModPath)
	}
}
