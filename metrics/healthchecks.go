package metrics

import "my/checker/config"

func initHealtheckMetric(h *config.Healtchecks) {
	if _, ok := Metrics.Healthchecks[h.Name]; !ok {
		Metrics.Healthchecks[h.Name] = new(HealtcheckMetrics)
		Metrics.Healthchecks[h.Name].Name = h.Name
	}
}

func HealtcheckAddError(h *config.Healtchecks) error {
	initHealtheckMetric(h)

	Metrics.Healthchecks[h.Name].ErrorsCount++
	return nil
}

func HealtcheckDecError(h *config.Healtchecks) error {
	initHealtheckMetric(h)

	if Metrics.Healthchecks[h.Name].ErrorsCount > 0 {
		Metrics.Healthchecks[h.Name].ErrorsCount--
	}
	return nil
}

func HealtcheckGetErrors(h *config.Healtchecks) int {
	initHealtheckMetric(h)

	return Metrics.Healthchecks[h.Name].ErrorsCount
}

func HealtcheckAddFail(h *config.Healtchecks) error {
	initHealtheckMetric(h)

	Metrics.Healthchecks[h.Name].FailsCount++
	return nil
}

func HealtcheckDecFail(h *config.Healtchecks) error {
	initHealtheckMetric(h)

	if Metrics.Healthchecks[h.Name].FailsCount > 0 {
		Metrics.Healthchecks[h.Name].FailsCount--
	}
	return nil
}

func HealtcheckGetFails(h *config.Healtchecks) int {
	initHealtheckMetric(h)

	return Metrics.Healthchecks[h.Name].FailsCount
}
