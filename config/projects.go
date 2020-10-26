package config

func CeaseProjectAlerts(p *Project) error {
	Log.Printf("Old mode: %s", p.Parameters.Mode)
	p.Parameters.Mode = "quiet"
	Log.Printf("New mode: %s", p.Parameters.Mode)
	return nil
}

func EnableProjectAlerts(p *Project) error {
	Log.Printf("Old mode: %s", p.Parameters.Mode)
	p.Parameters.Mode = "loud"
	Log.Printf("New GetCommandChannel: %s", p.Parameters.Mode)
	return nil
}
