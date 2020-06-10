package metrics

import "my/checker/config"

func AddCheckError(p *config.Project, h *config.Healtchecks, c *config.Check) error {
	CheckMetrics.WithLabelValues(p.Name, h.Name, c.UUid, "Error").Inc()
	CheckAlertsHistory.WithLabelValues(p.Name, h.Name, c.UUid, "Error").Observe(1)
	return nil
}

func AddCheckNoError(p *config.Project, h *config.Healtchecks, c *config.Check) error {
	CheckAlertsHistory.WithLabelValues(p.Name, h.Name, c.UUid, "Error").Observe(0)
	return nil
}

func AddCheckRunCount(p *config.Project, h *config.Healtchecks, c *config.Check) error {
	CheckMetrics.WithLabelValues(p.Name, h.Name, c.UUid, "RunCount").Inc()
	CheckAlertsHistory.WithLabelValues(p.Name, h.Name, c.UUid, "Run").Observe(1)
	return nil
}
