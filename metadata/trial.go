package metadata

// TrialDefinition is a grouping of clusters, and scenarios to run together
type TrialDefinition struct {
	Trials []Trial `json:"trials"`
}

// Trial is one particular cluster + scenario combination to run
type Trial struct {
	Cluster  []ClusterGroup     `json:"cluster"`
	Scenario ScenarioDefinition `json:"scenario"`
}
