package p2plab


items:: object & {
    golang: {
        type: "oci"
        source: "docker.io/library/golang:latest"
    }
    mysql: {
        type: "oci"
        source: "docker.io/library/mysql:latest"
    }
}

clust1:: Cluster & {
    groups: [
            Nodes & {
                size: 10
                instanceType: "t3.micro"
                region: "us-west-1"
            }, 
            Nodes & {
                size: 2
                instanceType: "t3.medium"
                region: "us-east-1"
                labels: [ "neighbors" ]
            } 
    ]
}

scen1:: Scenario & {
        objects:  [ items ]
        seed: {
            "neighbors": "golang"
        }
        benchmark: {
            "(not neighbors)": "golang"
        }
}

scen2:: Scenario & {
        objects:  [ items ]
        seed: {
            "neighbors": "golang"
        }
        benchmark: {
            "(neighbors)": "golang"
        }
}

experiment: Experiment & {
    cluster: clust1
    scenario: scen1
    trials: [[clust1,scen1],[clust1,scen2]]
}