package metrics

//
//type ProjectsMetrics struct {
//	Name           string
//	SeqErrorsCount int
//	ErrorsCount    int
//	FailsCount     int
//	RunCount       int
//	Alive          int
//}
//type AlertMetrics struct {
//	Name        string
//	AlertCount  int
//	NonCritical int
//	Critical    int
//	CommandAns  int
//	CommandReqs int
//}
//type HealtcheckMetrics struct {
//	Name        string
//	RunCount    int
//	ErrorsCount int
//	FailsCount  int
//}
//type CheckMetrics struct {
//	UUID        string
//	RunCount    int
//	ErrorsCount int
//	FailsCount  int
//	LastResult  bool
//}
//
//type MetricsCollection struct {
//	Projects     map[string]*ProjectsMetrics
//	Alerts       map[string]*AlertMetrics
//	Healthchecks map[string]*HealtcheckMetrics
//	Checks       map[string]*CheckMetrics
//}
//
//var Metrics *MetricsCollection
//
//func init() {
//	Metrics = new(MetricsCollection)
//	Metrics.Projects = make(map[string]*ProjectsMetrics)
//	Metrics.Alerts = make(map[string]*AlertMetrics)
//	Metrics.Checks = make(map[string]*CheckMetrics)
//	Metrics.Healthchecks = make(map[string]*HealtcheckMetrics)
//}
