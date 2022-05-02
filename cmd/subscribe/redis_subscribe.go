package subscribe

import (
	"context"
	"fmt"
	"github.com/rueian/rueidis"
	"log"
	"strings"
	"sync"
	"sync/atomic"
)

func SubscriberRoutine(addr string, subscriberName string, channel string, printMessages bool, stop chan struct{}, wg *sync.WaitGroup) {
	// tell the caller we've stopped
	defer wg.Done()

	conn, _ := BootstrapPubSub(addr, subscriberName, channel)
	defer conn.Close()
	sub := conn.B().Subscribe().Channel(channel).Build()
	msgCh := make(chan rueidis.PubSubMessage)
	go func() {
		for {
			conn.Receive(context.Background(), sub, func(msg rueidis.PubSubMessage) {
				// handle the msg
				msgCh <- msg
			})
		}
	}()

	for {
		select {
		case msg := <-msgCh:
			if printMessages {
				fmt.Println(fmt.Sprintf("received message in channel %s. Message: %s", msg.Channel, msg.Message))
			}
			atomic.AddUint64(&TotalMessages, 1)
			break
		case <-stop:
			return
		}
	}
}

func BootstrapPubSub(addr string, subscriberName string, channel string) (rueidis.Client, error) {
	// Create a normal redis connection

	c, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	err = c.Do(ctx, c.B().ClientSetname().ConnectionName(subscriberName).Build()).Error()
	if err != nil {
		log.Fatal(err)
	}

	return c, err
}

func RedisPubSubLogic(stopChan chan struct{}, wg *sync.WaitGroup, distributeSubscribers bool, host string, port string, client_output_buffer_limit_pubsub string, channel_maximum int, channel_minimum int, subscribers_per_channel int, subscribers_placement string, subscribe_prefix string, printMessages bool) {
	var nodes []string
	var node_subscriptions_count []int
	var err error

	if distributeSubscribers {
		nodes, node_subscriptions_count, err = getClusterNodesFromTopology(host, port)
	} else {
		nodes, node_subscriptions_count, err = getClusterNodesFromArgs(port, host)
	}
	if err != nil {
		log.Fatal(err)
	}

	if strings.Compare(subscribers_placement, "dense") == 0 {
		for channel_id := channel_minimum; channel_id <= channel_maximum; channel_id++ {
			for channel_subscriber_number := 1; channel_subscriber_number <= subscribers_per_channel; channel_subscriber_number++ {
				nodes_pos := channel_id % len(nodes)
				node_subscriptions_count[nodes_pos]++
				addr := nodes[nodes_pos]
				channel := fmt.Sprintf("%s%d", subscribe_prefix, channel_id)
				subscriberName := fmt.Sprintf("subscriber#%d-%s%d", channel_subscriber_number, subscribe_prefix, channel_id)
				wg.Add(1)
				go SubscriberRoutine(addr, subscriberName, channel, printMessages, stopChan, wg)
			}
		}
	}
}
