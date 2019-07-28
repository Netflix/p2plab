package p2plab

type LabdAPI interface {
	Cluster() ClusterAPI

	Node() NodeAPI

	Scenario() ScenarioAPI

	Benchmark() BenchmarkAPI
}

type LabAgentAPI interface {
}
