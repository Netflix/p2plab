package metadata

// TrialDefinition is a grouping of a cluster, and scenario to run together
type TrialDefinition struct {
	Cluster  []ClusterGroup     `json:"cluster"`
	Scenario ScenarioDefinition `json:"scenario"`
}
