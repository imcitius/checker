package checks

import (
	"github.com/google/uuid"
	"github.com/teris-io/shortid"
	"math/rand"
	"my/checker/checks/http"
	"my/checker/checks/icmp"
	"my/checker/config"
	"strings"
	"time"
)

func refineProjects() {
	for i, _p := range configurer.Projects {
		(configurer.Projects)[i] = config.TProject{
			Name:         i,
			Healthchecks: _p.Healthchecks,
			Parameters:   _p.Parameters,
		}

		for j, _h := range _p.Healthchecks {
			(_p.Healthchecks)[j] = config.THealthcheck{
				Name:   j,
				Checks: _h.Checks,
			}

			for k, _c := range _h.Checks {
				_c.UUid = genUUID((_p.Healthchecks)[j].Name, _c.Name, hostOrUrl(_c))

				if _c.Parameters.Duration == "" {
					if _p.Parameters.Duration != "" {
						_c.Parameters.Duration = _p.Parameters.Duration
					} else {
						_c.Parameters.Duration = minDuration(configurer.Defaults.Duration, configurer.Defaults.DefaultCheckParameters.Duration)
					}
				}

				(_h.Checks)[k] = _c
			}
		}
	}
}

func SetSID() string {
	sid, _ := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	s, _ := sid.Generate()
	return s
}

func (c *TCommonCheck) GetSID() string {
	return c.Sid
}

func (c *TCommonCheck) GetProject() string {
	return c.Project
}

func (c *TCommonCheck) GetHealthcheck() string {
	return c.Healthcheck
}

func (c *TCommonCheck) GetHost() string {
	if c.CheckConfig.Url != "" {
		return c.CheckConfig.Url
	} else {
		return c.CheckConfig.Host
	}
}

func (c *TCommonCheck) GetType() string {
	return c.Type
}

func (c *TCommonCheck) GetName() string {
	return c.Name
}

func (c *TCommonCheck) Execute() (time.Duration, error) {
	//logger.Infof("TCommonCheck started: %s", c.Name)

	d, err := c.RealCheck.RealExecute()
	return d, err
}

// TODO add caching
func GetChecksByDuration(duration string) (TChecksCollection, error) {
	//logger.Infof("Checks: %s", duration)
	checksCollection := TChecksCollection{
		Checks: make(map[string][]TCheckWithDuration),
	}

	for _, p := range configurer.Projects {

		healthchecks := p.Healthchecks
		//logger.Fatalf("Project %s Healthchecks: %+v", pName, healthchecks)

		for _, h := range healthchecks {
			//logger.Infof("Healthcheck: %+v", h)
			for _, check := range h.Checks {
				//logger.Infof("Check: %+v", check)
				//logger.Infof("Check: %s, dur: %s", check.Name, check.Parameters.Duration)

				if check.Parameters.Duration == duration {
					checkToAdd := newCommonCheck(check, h, p)
					checksCollection.Checks[duration] = append(checksCollection.Checks[duration], TCheckWithDuration{
						Check:    checkToAdd,
						Duration: duration,
					})
				}
			}
		}
	}

	//logger.Infof("Checks: %s", checksCollection)

	return checksCollection, nil
}

func newCommonCheck(c config.TCheckConfig, h config.THealthcheck, p config.TProject) ICommonCheck {

	newCheck := TCommonCheck{
		CheckConfig: c,
		Name:        c.Name,
		Project:     p.Name,
		Healthcheck: h.Name,
		Type:        c.Type,
		Parameters:  c.Parameters,
		RealCheck:   newSpecificCheck(c),
		Sid:         SetSID(),
	}

	return &newCheck
}

func newSpecificCheck(c config.TCheckConfig) ISpecificCheck {
	switch c.Type {
	case "http":
		return http.New(c)
	case "icmp":
		return icmp.New(c)
	}

	logger.Error("Error constructing specific check: unknown check type")
	return nil
}

func minDuration(a, b string) string {
	aDur, _ := time.ParseDuration(a)
	bDur, _ := time.ParseDuration(b)

	if aDur < bDur {
		return a
	} else {
		return b
	}
}

func genUUID(name ...string) string {
	var err error

	ns, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	if err != nil {
		return ""
	}

	u2 := uuid.NewSHA1(ns, []byte(strings.Join(name, ".")))
	return u2.String()
}

func hostOrUrl(c config.TCheckConfig) string {
	if c.Url != "" {
		return c.Url
	}
	return c.Host
}
