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

	nodes = []string{}
	node_subscriptions_count = []int{}
	topology, err := client.Do(ctx, client.B().ClusterSlots().Build()).ToArray()
	for _, message := range topology {
		group, _ := message.ToArray()
		firstNode, _ := group[2].ToArray()
		host, _ := firstNode[0].ToString()
		port, _ := firstNode[1].ToInt64()
		node := fmt.Sprintf("%s:%d", host, port)
		nodes = append(nodes, node)
		node_subscriptions_count = append(node_subscriptions_count, 0)
	}
	return
}
