package metrics

import "my/checker/config"

func AddCheckError(p *config.Project, h *config.Healthcheck, c *config.Check) error {
	CheckMetrics.WithLabelValues(p.Name, h.Name, c.UUid, "Error").Inc()
	return nil
}

func AddCheckRunCount(p *config.Project, h *config.Healthcheck, c *config.Check) error {
	CheckMetrics.WithLabelValues(p.Name, h.Name, c.UUid, "Run").Inc()
	return nil
}
