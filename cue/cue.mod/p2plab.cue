// defines a set of nodes of size 1 or higher
// a "node" is simply an EC2 instance provisioned of the given type
Nodes :: {
    // must be greater than or equal to 1
    // default value of this field is 1
    size: >=1 | *1
    instanceType: string
    region: string
    // labels is an optional field
    labels?: [...string]
}

// a cluster is a collection of 1 or more groups of nodes
Cluster :: {
    groups: [...Nodes]
}

object :: [Name=_]: { 
    type: string
    source: string
}

Scenario :: {
    objects: [...object]
    seed: { ... }
    // enable any fields for benchmark
    benchmark:  { ... }
}

Trial :: {
    cluster: Cluster
    scenario: Scenario
}

Experiment :: {
    cluster: Cluster
    scenario: Scenario
    // trials: [ ... ]
    trials: [ ...[Cluster,Scenario]]
}