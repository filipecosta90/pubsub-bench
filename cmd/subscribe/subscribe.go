package subscribe

import (
	"context"
	"fmt"
	"github.com/rueian/rueidis"
	"strings"
)

var TotalMessages uint64

func getClusterNodesFromArgs(port string, host string) (nodes []string, node_subscriptions_count []int, err error) {
	ports := strings.Split(port, ",")
	for idx, nhost := range strings.Split(host, ",") {
		node := fmt.Sprintf("%s:%s", nhost, ports[idx])
		nodes = append(nodes, node)
		node_subscriptions_count = append(node_subscriptions_count, 0)
	}
	return
}

func getClusterNodesFromTopology(host string, port string) (nodes []string, node_subscriptions_count []int, err error) {
	ports := strings.Split(port, ",")
	for idx, nhost := range strings.Split(host, ",") {
		node := fmt.Sprintf("%s:%s", nhost, ports[idx])
		nodes = append(nodes, node)
		node_subscriptions_count = append(node_subscriptions_count, 0)
	}

	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: nodes,
	})
	if err != nil {
		return
	}
	ctx := context.Background()
	topology, err := client.Do(ctx, client.B().ClusterSlots().Build()).AsStrSlice()
	fmt.Println(topology)
	return
}
