FROM telegraf:1.14-alpine

ENV INFLUXDB_URL "http://influxdb:8086"

COPY collect.*     /usr/bin/
COPY telegraf.conf /etc/telegraf/