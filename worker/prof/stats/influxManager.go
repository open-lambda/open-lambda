package main

import (
	"fmt"
	"log"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
)

const (
	// TODO: Do any of these need to be configurable?
	precision   = "ms"
	database    = "mydata"
	flushThresh = 100
)

type InfluxManager struct {
	client influx.Client

	bp         influx.BatchPoints
	pointCount int
}

func NewInfluxManager(host string, port string) (mgr *InfluxManager) {
	mgr = new(InfluxManager)
	client, err := influx.NewHTTPClient(influx.HTTPConfig{
		Addr: fmt.Sprintf("http://%s:%s", host, port),
	})
	if err != nil {
		log.Fatalf("failed to create new http client with err: %v\n", err)
	}

	q := influx.NewQuery(fmt.Sprintf("CREATE DATABASE mydata"), "", "")
	// q := influx.NewQuery(fmt.Sprintf("CREATE DATABASE %s", database), "", "")
	if _, err := client.Query(q); err != nil {
		log.Fatalf("failed to create db: %v\n", err)
	}

	mgr.client = client
	mgr.bp = newBP()
	mgr.pointCount = 0

	return mgr
}

func (mgr *InfluxManager) AddPointNow(name string, fields map[string]interface{}) {
	if mgr.pointCount > flushThresh {
		err := mgr.client.Write(mgr.bp)
		if err != nil {
			log.Fatalf("failed to write points with err %v\n", err)
		}
		log.Printf("good flush\n")
		mgr.bp = newBP()
		mgr.pointCount = 0
	}

	pt, err := influx.NewPoint(name, map[string]string{}, fields, time.Now())
	if err != nil {
		log.Fatalf("failed to create new point with err %v\n", err)
	}
	mgr.bp.AddPoint(pt)
	mgr.pointCount++
}

func newBP() influx.BatchPoints {
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  database,
		Precision: precision,
	})
	if err != nil {
		log.Fatalf("failed to make new bp err: %v\n")
	}
	return bp
}
