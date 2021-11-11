package catalog

import (
	checks "my/checker/checks"
	"my/checker/common"
	"my/checker/config"
	projects "my/checker/projects"
)

func CheckCatalog(timeout string) {

	//config.Log.Infof("Catalog: %+v", config.ProjectsCatalog)

	for _, p := range config.ProjectsCatalog {
		for _, h := range p.Healthchecks {
			for _, c := range h.Checks {
				if timeout == h.Parameters.RunEvery || timeout == p.Parameters.RunEvery {
					//config.Log.Debugf("check: %+v", c)
					duration, tempErr := checks.Execute(&projects.Project{p}, &c)
					checks.EvaluateCheckResult(&projects.Project{p}, &h, &c, tempErr, common.GetRandomId(), duration)
				}
			}
		}
	}
}
