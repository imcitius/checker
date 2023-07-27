package config

import (
	"github.com/InVisionApp/conjungo"
	"github.com/google/uuid"
	"reflect"
	"strings"
	"time"
)

func (c *TConfig) refineProjects() {
	newProjects := make(map[string]TProject)
	opts := conjungo.NewOptions()
	opts.SetTypeMergeFunc(
		reflect.TypeOf(TCheckParameters{}),
		func(t, s reflect.Value, o *conjungo.Options) (reflect.Value, error) {
			tFoo := t.Interface().(TCheckParameters)
			sFoo := s.Interface().(TCheckParameters)

			if sFoo.Duration != "" {
				tFoo.Duration = sFoo.Duration
			}
			if sFoo.Mode != "" {
				tFoo.Mode = sFoo.Mode
			}
			if sFoo.Timeout != "" {
				tFoo.Timeout = sFoo.Timeout
			}

			if sFoo.SSLExpirationPeriod != "" {
				tFoo.SSLExpirationPeriod = sFoo.SSLExpirationPeriod
			}
			if sFoo.MinHealth != 0 {
				tFoo.MinHealth = sFoo.MinHealth
			}
			if sFoo.AllowFails != 0 {
				tFoo.AllowFails = sFoo.AllowFails
			}

			if sFoo.AlerterName != "" {
				tFoo.AlerterName = sFoo.AlerterName
			}
			if sFoo.AlertChannel != "" {
				tFoo.AlertChannel = sFoo.AlertChannel
			}
			if sFoo.CritAlertChannel != "" {
				tFoo.CritAlertChannel = sFoo.CritAlertChannel
			}
			if sFoo.CommandChannel != "" {
				tFoo.CommandChannel = sFoo.CommandChannel
			}
			if len(sFoo.Mentions) != 0 {
				tFoo.Mentions = sFoo.Mentions
			}

			return reflect.ValueOf(tFoo), nil
		})

	for i, _p := range config.Projects {
		_project := TProject{
			Name:         "",
			Healthchecks: map[string]THealthcheck{},
			Parameters:   config.Defaults.DefaultCheckParameters,
		}
		_project.Name = i

		__pparams := config.Defaults.DefaultCheckParameters
		err := conjungo.Merge(&__pparams, _p.Parameters, opts)
		if err != nil {
			logger.Error(err)
		}
		_project.Parameters = __pparams
		//logrus.Fatalf("%+v", _project.Parameters)

		for j, _h := range _p.Healthchecks {
			_healthcheck := THealthcheck{
				Name:       "",
				Checks:     map[string]TCheckConfig{},
				Parameters: _project.Parameters,
			}
			_healthcheck.Name = j

			__hparams := _project.Parameters
			err := conjungo.Merge(&__hparams, _h.Parameters, opts)
			if err != nil {
				logger.Error(err)
			}

			_healthcheck.Parameters = __hparams
			_healthcheck.Checks = make(map[string]TCheckConfig)

			for k, _c := range _h.Checks {
				_check := _c
				_check.UUid = genUUID(_healthcheck.Name, _c.Name, hostOrUrl(_c))
				_check.Name = k

				__cparams := __hparams
				err := conjungo.Merge(&__cparams, _c.Parameters, opts)
				if err != nil {
					logger.Error(err)
				}

				_check.Parameters = __cparams
				_healthcheck.Checks[k] = _check
			}

			_project.Healthchecks[j] = _healthcheck
		}

		newProjects[i] = _project
	}
	config.Projects = newProjects
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

func hostOrUrl(c TCheckConfig) string {
	if c.Url != "" {
		return c.Url
	}
	return c.Host
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
