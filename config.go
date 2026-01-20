package smartlog

// TimberjackConfig holds the configuration for the timberjack logger.
type TimberjackConfig struct {
	Filename         string `mapstructure:"filename"`
	MaxSize          int    `mapstructure:"max_size"`
	MaxBackups       int    `mapstructure:"max_backups"`
	MaxAge           int    `mapstructure:"max_age"`
	Compression      string `mapstructure:"compression"`
	RotationInterval int    `mapstructure:"rotation_interval"` // in hours
	Level            string `mapstructure:"level"`
}

// GormConfig holds the configuration for the GORM logger.
type GormConfig struct {
	Level string `mapstructure:"level"`
}

// Config holds the configuration for the logger.
type Config struct {
	ServiceName string           `mapstructure:"service_name"`
	Env         string           `mapstructure:"env"`
	Log         TimberjackConfig `mapstructure:"log"`
	Gorm        GormConfig       `mapstructure:"gorm"`
	RedactKeys  []string         `mapstructure:"redact_keys"`
	SkipPaths   []string         `mapstructure:"skip_paths"`
}
