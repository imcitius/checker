package catalog

import (
	"fmt"
	"github.com/BurntSushi/ty/fun"
	"my/checker/config"
	"my/checker/status"
	"strings"
	"time"

	consul "github.com/hashicorp/consul/api"
)

func getServiceIds(services []*consul.CatalogService) []string {
	var serviceIds []string
	for _, service := range services {
		serviceIds = append(serviceIds, service.ID)
	}
	return serviceIds
}

func getServicePorts(services []*consul.CatalogService) []int {
	var servicePorts []int
	for _, service := range services {
		servicePorts = append(servicePorts, service.ServicePort)
	}
	return servicePorts
}

func getServiceAddresses(services []*consul.CatalogService) []string {
	var serviceAddresses []string
	for _, service := range services {
		serviceAddresses = append(serviceAddresses, service.ServiceAddress)
	}
	return serviceAddresses
}

func getChangedServiceKeys(current map[string]config.ConsulService, previous map[string]config.ConsulService) ([]string, []string) {
	currKeySet := fun.Set(fun.Keys(current).([]string)).(map[string]bool)
	prevKeySet := fun.Set(fun.Keys(previous).([]string)).(map[string]bool)

	addedKeys := fun.Difference(currKeySet, prevKeySet).(map[string]bool)
	removedKeys := fun.Difference(prevKeySet, currKeySet).(map[string]bool)

	return fun.Keys(addedKeys).([]string), fun.Keys(removedKeys).([]string)
}

func hasChanged(current map[string]config.ConsulService, previous map[string]config.ConsulService) bool {
	if len(current) != len(previous) {
		return true
	}
	addedServiceKeys, removedServiceKeys := getChangedServiceKeys(current, previous)
	return len(removedServiceKeys) > 0 || len(addedServiceKeys) > 0 || hasServiceChanged(current, previous)
}

func hasServiceChanged(current map[string]config.ConsulService, previous map[string]config.ConsulService) bool {
	for key, value := range current {
		if prevValue, ok := previous[key]; ok {
			addedNodesKeys, removedNodesKeys := getChangedStringKeys(value.Nodes, prevValue.Nodes)
			if len(addedNodesKeys) > 0 || len(removedNodesKeys) > 0 {
				return true
			}
			addedTagsKeys, removedTagsKeys := getChangedStringKeys(value.Tags, prevValue.Tags)
			if len(addedTagsKeys) > 0 || len(removedTagsKeys) > 0 {
				return true
			}
			addedAddressesKeys, removedAddressesKeys := getChangedStringKeys(value.Addresses, prevValue.Addresses)
			if len(addedAddressesKeys) > 0 || len(removedAddressesKeys) > 0 {
				return true
			}
			addedPortsKeys, removedPortsKeys := getChangedIntKeys(value.Ports, prevValue.Ports)
			if len(addedPortsKeys) > 0 || len(removedPortsKeys) > 0 {
				return true
			}
		}
	}
	return false
}

func getChangedIntKeys(currState []int, prevState []int) ([]int, []int) {
	currKeySet := fun.Set(currState).(map[int]bool)
	prevKeySet := fun.Set(prevState).(map[int]bool)

	addedKeys := fun.Difference(currKeySet, prevKeySet).(map[int]bool)
	removedKeys := fun.Difference(prevKeySet, currKeySet).(map[int]bool)

	return fun.Keys(addedKeys).([]int), fun.Keys(removedKeys).([]int)
}

func getChangedStringKeys(currState []string, prevState []string) ([]string, []string) {
	currKeySet := fun.Set(currState).(map[string]bool)
	prevKeySet := fun.Set(prevState).(map[string]bool)

	addedKeys := fun.Difference(currKeySet, prevKeySet).(map[string]bool)
	removedKeys := fun.Difference(prevKeySet, currKeySet).(map[string]bool)

	return fun.Keys(addedKeys).([]string), fun.Keys(removedKeys).([]string)
}

func GetConsulServices() (map[string]config.ConsulService, error) {
	const (
		DefaultWatchWaitTime = 15 * time.Second
		//passingOnly          = true
	)

	var (
		addr    = config.Config.ConsulCatalog.Address
		conf    = consul.DefaultConfig()
		options = &consul.QueryOptions{WaitTime: DefaultWatchWaitTime, AllowStale: false}
		current = make(map[string]config.ConsulService)
	)

	config.Log.Debug("Getting consul services")

	conf.Address = addr

	c, err := consul.NewClient(conf)
	if err != nil {
		config.Log.Fatalf("Consul client error: %v", err)
	}

	catalog := c.Catalog()

	data, meta, err := catalog.Services(options)
	if err != nil {
		config.Log.Errorf("Failed to list services: %v", err)
		//notifyError(err)
		return map[string]config.ConsulService{}, fmt.Errorf("failed to list services: %v", err)
	}

	//if options.WaitIndex == meta.LastIndex {
	//	continue
	//}
	options.WaitIndex = meta.LastIndex

	if data != nil {
		for key, value := range data {
			nodes, _, err := catalog.Service(key, "", &consul.QueryOptions{AllowStale: false})
			if err != nil {
				config.Log.Errorf("Failed to get detail of service %s: %v", key, err)
				//notifyError(err)
				return map[string]config.ConsulService{}, fmt.Errorf("failed to get detail of service %s: %v", key, err)

			}

			nodesID := getServiceIds(nodes)
			ports := getServicePorts(nodes)
			addresses := getServiceAddresses(nodes)

			if service, ok := current[key]; ok {
				service.Tags = value
				service.Nodes = nodesID
				service.Ports = ports
			} else {
				service := config.ConsulService{
					Name:      key,
					Tags:      value,
					Nodes:     nodesID,
					Addresses: addresses,
					Ports:     ports,
				}
				current[key] = service
			}
		}
	}
	return current, nil
}

func WatchServices() {

	//stopCh <-chan struct{}, watchCh chan<- map[string][]string
	var flashback map[string]config.ConsulService
	const watchPeriod = 30 * time.Second

	go func() {

		for {

			current, err := GetConsulServices()
			if err != nil {
				config.Log.Errorf("Failed to get consul services: %s", err)
				//notifyError(err)
				return
			}

			// A critical note is that the return of a blocking request is no guarantee of a change.
			// It is possible that there was an idempotent write that does not affect the result of the query.
			// Thus it is required to do extra check for changes...
			if hasChanged(current, flashback) {
				//config.Log.Infof("%+v", len(current))
				//config.Log.Infof("%+v", len(flashback))
				//	watchCh <- data
				flashback = current
				ParseCatalog(current)
			}
			//config.Log.Infof("services: %+v\n\n\n", current)

			// we do not want to DoS consul's api
			time.Sleep(watchPeriod)
		}

	}()

}

func ParseCatalog(catalog map[string]config.ConsulService) {

	var (
		projects = make(map[string]config.Project)
	)

	for catname, element := range catalog {

		var (
			project      = config.Project{}
			healthcheck  = config.Healthcheck{}
			check        = config.Check{}
			healthchecks []config.Healthcheck
			tags         = make(map[string]string)
			found        = false
		)

		for _, tag := range element.Tags {
			if t := strings.Split(tag, "="); strings.HasPrefix(t[0], "checker") {
				// not empty tags only
				if len(t) > 1 {
					tags[t[0]] = t[1]
				}
				found = true
			}

		}
		if found {
			for name, value := range tags {
				//config.Log.Infof("tag name: %s, value %s", name, value)
				switch true {
				case strings.HasPrefix(name, "checker.check.type"):
					check.Type = value
				case strings.HasPrefix(name, "checker.check.host"):
					check.Host = value
					check.UUid = config.GenUUID(name, value)
				case strings.HasPrefix(name, "checker.check.timeout"):
					check.Timeout = value
				case strings.HasPrefix(name, "checker.check.mode"):
					err := status.SetCheckMode(&check, value)
					if err != nil {
						config.Log.Errorf("Error change check's status: %s", err)
					}
				case strings.HasPrefix(name, "checker.project.name"):
					project.Name = value
				case strings.HasPrefix(name, "checker.healthcheck.name"):
					healthcheck.Name = value
				case strings.HasPrefix(name, "checker.alert.channel"):
					project.Parameters.AlertChannel = value
				case strings.HasPrefix(name, "checker.alert.critchannel"):
					project.Parameters.CritAlertChannel = value
				}
			}
			if healthcheck.Name == "" {
				healthcheck.Name = "hc: " + catname
			}
			if project.Name == "" {
				project.Name = "c: " + catname
			}

			healthcheck.Checks = append(healthcheck.Checks, check)
			healthchecks = append(healthchecks, healthcheck)
			//config.Log.Infof("healthchecks: %+v", healthchecks)

			if _, ok := projects[project.Name]; !ok {
				projects[project.Name] = config.Project{
					Name:         project.Name,
					Parameters:   config.Config.Defaults.Parameters,
					Healthchecks: []config.Healthcheck{},
				}
			}

			project = projects[project.Name]
			project.Healthchecks = append(project.Healthchecks, healthchecks...)
			status.InitProject(&project)
			projects[project.Name] = project
			//config.Log.Infof("project: %+v", project)

		}

	}

	//config.Log.Panicf("projects: %+v\n", projects)
	config.ProjectsCatalog = projects

}
