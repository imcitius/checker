package check

import (
	"my/checker/config"
	"my/checker/metrics"
	projects "my/checker/projects"
)

func AddCheckError(p *projects.Project, h *config.Healthcheck, c *config.Check) error {
	metrics.CheckMetrics.WithLabelValues(p.Name, h.Name, c.UUid, "Error").Inc()
	return nil
}

func AddCheckRunCount(p *projects.Project, h *config.Healthcheck, c *config.Check) error {
	metrics.CheckMetrics.WithLabelValues(p.Name, h.Name, c.UUid, "Run").Inc()
	return nil
}
