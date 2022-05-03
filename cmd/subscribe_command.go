/*
Copyright © 2022 codeperfio <filipecosta.90@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/codeperfio/pubsub-bench/cmd/subscribe"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

var redisPubSub string = "redis-pubsub"
var redisShardedPubSub string = "redis-sharded-pubsub"
var subscribeChoices []string = []string{redisPubSub, redisShardedPubSub}

// subscribeCmd represents the subscribe command
var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscriber workload generator for each of the implementation",
	Long: `This tool is meant to provide a rough estimate on how fast each Pub/Sub can process messages.
It uses very simplified infrastructure to set things up and default configurations.
Keep in mind that final performance depends on multiple factors.`,
	Run: subcribeLogic,
}

func init() {
	rootCmd.AddCommand(subscribeCmd)
	rootCmd.PersistentFlags().String("system", "redis-pubsub", fmt.Sprintf("System to use. (choices %s)", strings.Join(subscribeChoices, ",")))
	rootCmd.PersistentFlags().String("json-out-file", "", "Name of json output file, if not set, will not print to json.")
	rootCmd.PersistentFlags().String("subscriber-prefix", "channel-", "prefix for subscribing to channel, used in conjunction with key-minimum and key-maximum.")
	rootCmd.PersistentFlags().Int("channel-minimum", 1, "channel ID minimum value ( each channel has a dedicated thread ).")
	rootCmd.PersistentFlags().Int("channel-maximum", 100, "channel ID maximum value ( each channel has a dedicated thread ).")
	rootCmd.PersistentFlags().Int("subscribers-per-channel", 1, "number of subscribers per channel.")
	rootCmd.PersistentFlags().Int("messages", 0, "Number of total messages per subscriber per channel.")
	rootCmd.PersistentFlags().Int("client-update-tick", 1, "client update tick.")
	rootCmd.PersistentFlags().Int("test-time", 0, "Number of seconds to run the test, after receiving the first message.")
	rootCmd.PersistentFlags().Int("debug-level", 0, "debug level. 0 - no debug; 1 - info; 2 - verbose.")

	// specific to redis
	rootCmd.PersistentFlags().Bool("oss-cluster-api-distribute-subscribers", false, "read cluster slots and distribute subscribers among them.")
	rootCmd.PersistentFlags().String("client-output-buffer-limit-pubsub", "", "Specify client output buffer limits for clients subscribed to at least one pubsub channel or pattern. If the value specified is different that the one present on the DB, this setting will apply.")
	rootCmd.PersistentFlags().String("host", "127.0.0.1", "redis host.")
	rootCmd.PersistentFlags().String("port", "6379", "redis port.")
	rootCmd.PersistentFlags().String("subscribers-placement-per-channel", "dense", "(dense,sparse) dense - Place all subscribers to channel in a specific shard. sparse- spread the subscribers across as many shards possible, in a round-robin manner.")

}

type testResult struct {
	StartTime             int64     `json:"StartTime"`
	Duration              float64   `json:"Duration"`
	MessageRate           float64   `json:"MessageRate"`
	TotalMessages         uint64    `json:"TotalMessages"`
	TotalSubscriptions    int       `json:"TotalSubscriptions"`
	ChannelMin            int       `json:"ChannelMin"`
	ChannelMax            int       `json:"ChannelMax"`
	SubscribersPerChannel int       `json:"SubscribersPerChannel"`
	MessagesPerChannel    int64     `json:"MessagesPerChannel"`
	MessageRateTs         []float64 `json:"MessageRateTs"`
	OSSDistributedSlots   bool      `json:"OSSDistributedSlots"`
	Addresses             []string  `json:"Addresses"`
}

func subcribeLogic(cmd *cobra.Command, args []string) {
	system, _ := cmd.Flags().GetString("system")
	json_out_file, _ := cmd.Flags().GetString("json-out-file")
	subscribe_prefix, _ := cmd.Flags().GetString("subscriber-prefix")
	client_output_buffer_limit_pubsub, _ := cmd.Flags().GetString("client-output-buffer-limit-pubsub")
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetString("port")
	subscribers_placement, _ := cmd.Flags().GetString("subscribers-placement-per-channel")
	debugLevel, _ := cmd.Flags().GetInt("debug-level")
	distributeSubscribers, _ := cmd.Flags().GetBool("oss-cluster-api-distribute-subscribers")
	channel_minimum, _ := cmd.Flags().GetInt("channel-minimum")
	channel_maximum, _ := cmd.Flags().GetInt("channel-maximum")
	subscribers_per_channel, _ := cmd.Flags().GetInt("subscribers-per-channel")
	messages_per_channel_subscriber, _ := cmd.Flags().GetInt("messages")
	client_update_tick, _ := cmd.Flags().GetInt("client-update-tick")
	test_time, _ := cmd.Flags().GetInt("test-time")

	if test_time != 0 && messages_per_channel_subscriber != 0 {
		log.Fatal(fmt.Errorf("--messages and --test-time are mutially exclusive ( please specify one or the other )"))
	}

	total_channels := channel_maximum - channel_minimum + 1
	total_subscriptions := total_channels * subscribers_per_channel
	total_messages := int64(total_subscriptions * messages_per_channel_subscriber)
	fmt.Println(fmt.Sprintf("Total subcriptions: %d. Total messages: %d", total_subscriptions, total_messages))

	subscribe.TotalMessages = 0

	stopChan := make(chan struct{})
	// a WaitGroup for the goroutines to tell us they've stopped
	wg := sync.WaitGroup{}

	switch system {
	case redisPubSub:
		{
			subscribe.RedisPubSubLogic(debugLevel, stopChan, &wg, distributeSubscribers, host, port, client_output_buffer_limit_pubsub, channel_maximum, channel_minimum, subscribers_per_channel, subscribers_placement, subscribe_prefix)
		}
	case redisShardedPubSub:
		{
			subscribe.RedisShardedPubSubLogic(debugLevel, stopChan, &wg, distributeSubscribers, host, port, client_output_buffer_limit_pubsub, channel_maximum, channel_minimum, subscribers_per_channel, subscribers_placement, subscribe_prefix)
		}
	}

	// listen for C-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	w := new(tabwriter.Writer)

	tick := time.NewTicker(time.Duration(client_update_tick) * time.Second)
	closed, start_time, duration, totalMessages, messageRateTs := updateCLI(tick, c, total_messages, w, test_time)
	messageRate := float64(totalMessages) / float64(duration.Seconds())

	fmt.Fprint(w, fmt.Sprintf("#################################################\nTotal Duration %f Seconds\nMessage Rate %f\n#################################################\n", duration.Seconds(), messageRate))
	fmt.Fprint(w, "\r\n")
	w.Flush()

	if strings.Compare(json_out_file, "") != 0 {

		res := testResult{
			StartTime:             start_time.Unix(),
			Duration:              duration.Seconds(),
			MessageRate:           messageRate,
			TotalMessages:         totalMessages,
			TotalSubscriptions:    total_subscriptions,
			ChannelMin:            channel_minimum,
			ChannelMax:            channel_maximum,
			SubscribersPerChannel: subscribers_per_channel,
			MessagesPerChannel:    int64(messages_per_channel_subscriber),
			MessageRateTs:         messageRateTs,
		}
		file, err := json.MarshalIndent(res, "", " ")
		if err != nil {
			log.Fatal(err)
		}

		err = ioutil.WriteFile(json_out_file, file, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	if closed {
		return
	}

	// tell the goroutine to stop
	close(stopChan)
	// and wait for them both to reply back
	wg.Wait()
}

func updateCLI(tick *time.Ticker, c chan os.Signal, message_limit int64, w *tabwriter.Writer, test_time int) (bool, time.Time, time.Duration, uint64, []float64) {

	start := time.Now()
	prevTime := time.Now()
	prevMessageCount := uint64(0)
	messageRateTs := []float64{}

	w.Init(os.Stdout, 25, 0, 1, ' ', tabwriter.AlignRight)
	fmt.Fprint(w, fmt.Sprintf("Test Time\tTotal Messages\t Message Rate \t"))
	fmt.Fprint(w, "\n")
	w.Flush()
	for {
		select {
		case <-tick.C:
			{
				now := time.Now()
				took := now.Sub(prevTime)
				messageRate := float64(subscribe.TotalMessages-prevMessageCount) / float64(took.Seconds())
				if prevMessageCount == 0 && subscribe.TotalMessages != 0 {
					start = time.Now()
				}
				if subscribe.TotalMessages != 0 {
					messageRateTs = append(messageRateTs, messageRate)
				}
				prevMessageCount = subscribe.TotalMessages
				prevTime = now

				fmt.Fprint(w, fmt.Sprintf("%.0f\t%d\t%.2f\t", time.Since(start).Seconds(), subscribe.TotalMessages, messageRate))
				fmt.Fprint(w, "\r\n")
				w.Flush()
				if message_limit > 0 && subscribe.TotalMessages >= uint64(message_limit) {
					return true, start, time.Since(start), subscribe.TotalMessages, messageRateTs
				}
				if test_time > 0 && time.Since(start) >= time.Duration(test_time*1000*1000*1000) && subscribe.TotalMessages != 0 {
					return true, start, time.Since(start), subscribe.TotalMessages, messageRateTs
				}

				break
			}

		case <-c:
			fmt.Println("received Ctrl-c - shutting down")
			return true, start, time.Since(start), subscribe.TotalMessages, messageRateTs
		}
	}
	return false, start, time.Since(start), subscribe.TotalMessages, messageRateTs
}
