package catalog

import (
	checks "my/checker/checks"
	"my/checker/config"
	projects "my/checker/projects"
)

func CheckCatalog(ticker string) {

	//config.Log.Infof("Catalog: %+v", config.ProjectsCatalog)

	for _, p := range config.ProjectsCatalog {
		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				if ticker == h.Parameters.Period || ticker == p.Parameters.Period {
					checkRandomId := config.GetRandomId()
					config.Log.Warnf("(%s) Checking project/healthcheck/check: '%s/%s/%s(%s)'", checkRandomId, "projectCatalog", h.Name, c.Name, c.Type)
					duration, tempErr := checks.Execute(&projects.Project{Project: p}, &c)
					checks.EvaluateCheckResult(&projects.Project{Project: p}, &h, &c, tempErr, config.GetRandomId(), duration, "CheckCatalog")
				}
			}
		}
	}
}
