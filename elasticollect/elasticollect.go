package elasticollect

import (
	"bandwidth/redis"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	//"bandwidth/redis"
	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	client     *elastic.Client
	timelayout = "2006-01-02 15:04:05"
	lastTime   time.Time
	hits       = make(chan json.RawMessage, 10000)
	count      uint64
)

func initLog() {
	filename := viper.GetString("log_file")
	logfile, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(logfile)
	log.SetFormatter(&log.JSONFormatter{})
}

func createElasticClient() (*elastic.Client, error) {
	url := viper.GetString("es_api")
	client, err := elastic.NewClient(elastic.SetURL(url))
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Run search from elasticsearch
func Run() {
	initLog()
	setBeginDateIndex()

	client, err := createElasticClient()
	if err != nil {
		log.Fatal(err)
		return
	}

	go pushDataTicker(client)

	for {
		select {
		case msg := <-hits:
			fmt.Println(string(msg))
		}
	}
}

func setBeginDateIndex() {
	startTime := viper.GetString("esorigintime")
	index, err := time.ParseInLocation(timelayout, startTime, time.Local)
	if err != nil {
		return
	}
	lastTime = index
}

func pushDataTicker(client *elastic.Client) {
	//lastTime = lastMonthDay()
	scrollParallel(client, lastTime, time.Now())

	ticker := time.NewTicker(time.Minute * 1)
	for currentTime := range ticker.C {
		scrollParallel(client, lastTime, currentTime)
	}
}

func scrollParallel(client *elastic.Client, last time.Time, current time.Time) {
	lastTime = current

	left := last.Format(timelayout)
	right := current.Format(timelayout)

	log.WithFields(log.Fields{"start": left, "stop": right, "count": atomic.LoadUint64(&count)}).Info("es search timestamp range")

	index := fmt.Sprintf("nginx-%s", last.Format("2006-01-02"))
	go func(index string) {
		// Initialize scroller, Just don't call Do yet.
		query := elastic.NewBoolQuery().Filter(
			elastic.NewRangeQuery("Timestamp").Gte(left).Lt(right),
		)

		search := client.Search().Index(index).Query(query).Size(0)
		aggHTTPHost := elastic.NewTermsAggregation().Field("http_host").Size(10000).OrderByCountDesc()
		sumRequestLength := elastic.NewSumAggregation().Field("request_length")
		sumUpstreamResponseLength := elastic.NewSumAggregation().Field("upstream_response_length")
		aggHTTPHost = aggHTTPHost.SubAggregation("requestLength", sumRequestLength).SubAggregation("upstreamResponseLength", sumUpstreamResponseLength)
		search = search.Aggregation("agg_httpHost", aggHTTPHost)

		res, err := search.Do(context.Background())
		if err != nil {
			log.Error("search elasticsearch failed:" + err.Error())
			return
		}
		if res.Hits.TotalHits > 0 {
			agg, found := res.Aggregations.Terms("agg_httpHost")
			if !found {
				log.Error("not found a terms aggregation called agg_httpHost")
				return
			}

			for _, recordBucket := range agg.Buckets {
				host := recordBucket.Key.(string)

				var in, out float64
				requestLength, _ := recordBucket.Sum("requestLength")
				if requestLength != nil {
					in = *requestLength.Value
				}

				upstreamResponseLength, _ := recordBucket.Sum("upstreamResponseLength")
				if upstreamResponseLength != nil {
					out = *upstreamResponseLength.Value
				}

				redis.Hset(host, "bandwidth_in_current", fmt.Sprintf("%d", int(in)/60))
				redis.Hset(host, "bandwidth_out_current", fmt.Sprintf("%d", int(out)/60))
				redis.Hset(host, "bandwidth_total_current", fmt.Sprintf("%d", int(in+out)/60))
				redis.Expire(host, 60)
			}
		}
	}(index)
}
