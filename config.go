package smartlog

// Config holds the configuration for the logger.
type Config struct {
	ServiceName string   `mapstructure:"service_name"`
	Env         string   `mapstructure:"env"`
	LogPath     string   `mapstructure:"log_path"`
	RedactKeys  []string `mapstructure:"redact_keys"`
}
