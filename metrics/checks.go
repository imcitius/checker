package metrics

import "my/checker/config"

func initCheckMetric(c *config.Check) {
	if _, ok := Metrics.Checks[c.UUid]; !ok {
		Metrics.Checks[c.UUid] = new(CheckMetrics)
		Metrics.Checks[c.UUid].UUID = c.UUid
	}
}

func CheckAddError(c *config.Check) error {
	initCheckMetric(c)

	Metrics.Checks[c.UUid].ErrorsCount++
	return nil
}

func CheckDecError(c *config.Check) error {
	initCheckMetric(c)

	if Metrics.Checks[c.UUid].ErrorsCount > 0 {
		Metrics.Checks[c.UUid].ErrorsCount--
	}
	return nil
}

func CheckGetErrors(c *config.Check) int {
	initCheckMetric(c)

	return Metrics.Checks[c.UUid].ErrorsCount
}

func CheckAddFail(c *config.Check) error {
	initCheckMetric(c)

	Metrics.Checks[c.UUid].FailsCount++
	return nil
}

func CheckDecFail(c *config.Check) error {
	initCheckMetric(c)

	if Metrics.Checks[c.UUid].FailsCount > 0 {
		Metrics.Checks[c.UUid].FailsCount--
	}
	return nil
}

func CheckGetFails(c *config.Check) int {
	initCheckMetric(c)

	return Metrics.Checks[c.UUid].FailsCount
}
