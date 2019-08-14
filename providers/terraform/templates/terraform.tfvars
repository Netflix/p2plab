cluster_id = "{{$.ID}}"

labagents = {
    {{range .RegionalClusterGroups}}
    {{.Region}} = {
        {{range $i, $group := .Groups}}
        {{$.ID}}-{{$i}} = {
            size          = {{$group.Size}}
            instance_type = "{{$group.InstanceType}}"
        }
        {{end}}
    }
    {{end}}
}
