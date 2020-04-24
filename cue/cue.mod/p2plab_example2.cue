package p2plab
experiment: Experiment & {
	trials: [
		Trial & {
			cluster: groups: [
				{
					size:         1
					instanceType: "t3.micro"
					region:       "us-west-1"
				},
				{
					size:         2
					instanceType: "t3.micro"
					region:       "us-west-1"
					labels: [ "neighbors"]
				},
			]
			scenario: {
				objects: o
				seed: {
					"neighbors": "image"
				}
				benchmark: {
					"(not neighbors)": "image"
				}
			}
		} for o in objects
	]
}
objects :: [
	[{
		image: {
			type:   "oci"
			source: "docker.io/library/golang:latest"
		}
	}],
	[{
		image: {
			type:   "oci"
			source: "docker.io/library/mysql:latest"
		}
	}],
]