package config

func (c *Config) GetRunMode() string {
	if c.runMode == "" {
		return ModeAIO
	}
	return c.runMode
}
