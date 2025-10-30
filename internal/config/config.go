package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Cloudinary      CloudinaryConfig      `mapstructure:"cloudinary"`
	Upload          UploadConfig          `mapstructure:"upload"`
	Transformations TransformationsConfig `mapstructure:"transformations"`
	Eager           []EagerTransformation `mapstructure:"eager"`
	State           StateConfig           `mapstructure:"state"`
	Output          OutputConfig          `mapstructure:"output"`
}

// CloudinaryConfig holds Cloudinary-specific settings
type CloudinaryConfig struct {
	CloudName string `mapstructure:"cloud_name"`
	APIKey    string `mapstructure:"api_key"`
	APISecret string `mapstructure:"api_secret"`
	Folder    string `mapstructure:"folder"`
}

// UploadConfig holds upload-related settings
type UploadConfig struct {
	ConcurrentWorkers int           `mapstructure:"concurrent_workers"`
	RetryAttempts     int           `mapstructure:"retry_attempts"`
	RetryDelay        time.Duration `mapstructure:"retry_delay"`
	Timeout           time.Duration `mapstructure:"timeout"`
}

// TransformationsConfig holds image transformation settings
type TransformationsConfig struct {
	Format        string `mapstructure:"format"`
	Quality       string `mapstructure:"quality"`
	MaxWidth      int    `mapstructure:"max_width"`
	MaxHeight     int    `mapstructure:"max_height"`
	StripMetadata bool   `mapstructure:"strip_metadata"`
}

// EagerTransformation represents a pre-generated transformation
type EagerTransformation struct {
	Transformation string `mapstructure:"transformation"`
	Name           string `mapstructure:"name"`
}

// StateConfig holds state management settings
type StateConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	File    string `mapstructure:"file"`
}

// OutputConfig holds output-related settings
type OutputConfig struct {
	ReportDir    string `mapstructure:"report_dir"`
	Verbose      bool   `mapstructure:"verbose"`
	ShowProgress bool   `mapstructure:"show_progress"`
}

// Load reads the configuration from the specified file
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// set config file path
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.image-uploader")
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate Cloudinary config
	if c.Cloudinary.CloudName == "" {
		return fmt.Errorf("cloudinary.cloud_name is required")
	}
	if c.Cloudinary.APIKey == "" {
		return fmt.Errorf("cloudinary.api_key is required")
	}
	if c.Cloudinary.APISecret == "" {
		return fmt.Errorf("cloudinary.api_secret is required")
	}

	// Validate upload config
	if c.Upload.ConcurrentWorkers <= 0 {
		return fmt.Errorf("upload.concurrent_workers must be greater than 0")
	}
	if c.Upload.ConcurrentWorkers > 100 {
		return fmt.Errorf("upload.concurrent_workers should not exceed 100")
	}
	if c.Upload.RetryAttempts < 0 {
		return fmt.Errorf("upload.retry_attempts must be non-negative")
	}

	// Validate transformations
	if c.Transformations.MaxWidth <= 0 {
		return fmt.Errorf("transformations.max_width must be greater than 0")
	}
	if c.Transformations.MaxHeight <= 0 {
		return fmt.Errorf("transformations.max_height must be greater than 0")
	}

	// Validate state config
	if c.State.Enabled && c.State.File == "" {
		return fmt.Errorf("state.file is required when state tracking is enabled")
	}

	// Validate output config
	if c.Output.ReportDir == "" {
		return fmt.Errorf("output.report_dir is required")
	}

	return nil
}

// GetCloudinaryURL returns the Cloudinary base URL
func (c *CloudinaryConfig) GetCloudinaryURL() string {
	return fmt.Sprintf("cloudinary://%s:%s@%s", c.APIKey, c.APISecret, c.CloudName)
}
