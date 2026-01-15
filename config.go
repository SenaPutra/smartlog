package smartlog

// TimberjackConfig holds the configuration for the timberjack logger.
type TimberjackConfig struct {
	Filename         string `mapstructure:"filename"`
	MaxSize          int    `mapstructure:"max_size"`
	MaxBackups       int    `mapstructure:"max_backups"`
	MaxAge           int    `mapstructure:"max_age"`
	Compression      string `mapstructure:"compression"`
	RotationInterval int    `mapstructure:"rotation_interval"` // in hours
}

// Config holds the configuration for the logger.
type Config struct {
	ServiceName string           `mapstructure:"service_name"`
	Env         string           `mapstructure:"env"`
	Log         TimberjackConfig `mapstructure:"log"`
	RedactKeys  []string         `mapstructure:"redact_keys"`
}
