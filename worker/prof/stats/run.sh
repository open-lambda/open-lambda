# run influxdb
docker run -d -p 8083:8083 -p 8086:8086 --expose 8090 --expose 8099 --name influxdb tutum/influxdb

sleep 5

# run stats daemon
docker run -d --link influxdb:influx --name stats-worker stats-worker /go/bin/app 
