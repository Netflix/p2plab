// defines a set of nodes of size 1 or higher
// a "node" is simply an EC2 instance provisioned of the given type
// and there may be more than 1 node in a group, however there must always be 1
Group :: {
    // must be greater than or equal to 1
    // default value of this field is 1
    size: >=1 | *1
    instanceType: string
    region: string
    // labels is an optional field
    labels?: [...string]
}

// a cluster is a collection of 1 or more groups of nodes
// that will be participating in a given benchmark
Cluster :: {
    groups: [...Group]
}

// an object is a particular data format to be used in benchmarking
// typically these are container images
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
    trials: [...Trial]
    // trials: [ ...[Cluster,Scenario]]
}