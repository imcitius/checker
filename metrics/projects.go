package metrics

import "my/checker/config"

func initProjectMetric(p *config.Project) {
	if _, ok := Metrics.Projects[p.Name]; !ok {
		Metrics.Projects[p.Name] = new(ProjectsMetrics)
		Metrics.Projects[p.Name].Name = p.Name
	}
}

func AddError(p *config.Project) error {
	initProjectMetric(p)

	Metrics.Projects[p.Name].ErrorsCount++
	return nil
}

func DecError(p *config.Project) error {
	initProjectMetric(p)

	if Metrics.Projects[p.Name].ErrorsCount > 0 {
		Metrics.Projects[p.Name].ErrorsCount--
	}
	return nil
}

func GetErrors(p *config.Project) int {
	initProjectMetric(p)

	return Metrics.Projects[p.Name].ErrorsCount
}

func AddFail(p *config.Project) error {
	initProjectMetric(p)

	Metrics.Projects[p.Name].FailsCount++
	return nil
}

func DecFail(p *config.Project) error {
	initProjectMetric(p)

	if Metrics.Projects[p.Name].FailsCount > 0 {
		Metrics.Projects[p.Name].FailsCount--
	}
	return nil
}

func GetFails(p *config.Project) int {
	initProjectMetric(p)

	return Metrics.Projects[p.Name].FailsCount
}
