package main

import (
	"fmt"

	vtepdb "github.com/cn-pmlabs/govtep/lib/odbapi/controllervtep"
	ovnnb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnnorthbound"
	ovnsb "github.com/cn-pmlabs/govtep/lib/odbapi/ovnsouthbound"
)

func main() {
	// Init ovsdb lib
	vtepdb.InitControllervtep("tcp:0.0.0.0:6644")
	ovnnb.InitOvnnorthbound("tcp:0.0.0.0:6641")
	ovnsb.InitOvnsouthbound("tcp:0.0.0.0:6642")

	lbIndex := ovnnb.LoadBalancerUUIDIndex{
		UUID: "web_lb",
	}
	tableLB, err := ovnnb.LoadBalancerGetByIndex(lbIndex)
	if err != nil {
		fmt.Printf("get lb %s failed\n", lbIndex.UUID)
	}

	for vip, backends := range tableLB.Vips {
		fmt.Printf("web_lb vip %s backends %s\n", vip.(string), backends.(string))
		tableLBHealthyCheck := ovnnb.TableLoadBalancerHealthCheck{
			Vip: vip.(string),
		}
		options := make(map[interface{}]interface{})
		options["interval"] = "5"
		options["timeout"] = "3"
		options["success_count"] = "1"
		options["failure_count"] = "1"
		tableLBHealthyCheck.Options = options

		ovnnb.LoadBalancerUpdateAddHealthCheck(lbIndex, tableLBHealthyCheck)
	}

	ipPortMappings := make(map[interface{}]interface{})
	ipPortMappings["172.16.10.10"] = "ls1-vm1:172.16.20.10"
	ipPortMappings["172.16.10.11"] = "ls1-vm2:172.16.10.10"

	ovnnb.LoadBalancerSetField(lbIndex, ovnnb.LoadBalancerFieldIPPortMappings, ipPortMappings)
}
