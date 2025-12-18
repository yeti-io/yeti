#!/bin/sh

JMX_PORT=${JMX_PORT:-5556}
JMX_CONFIG=${JMX_CONFIG:-/etc/jmx-exporter/config.yml}

if [ ! -f "$JMX_CONFIG" ]; then
    echo "ERROR: Config file not found at $JMX_CONFIG"
    exit 1
fi

echo "Starting JMX Exporter..."
echo "  Port: $JMX_PORT"
echo "  Config: $JMX_CONFIG"

exec java \
    -Dcom.sun.jndi.ldap.connect.pool.protocol=plain \
    -Dcom.sun.jndi.rmi.object.trustURLCodebase=false \
    -jar /opt/jmx-exporter/jmx_prometheus_httpserver.jar \
    "$JMX_PORT" "$JMX_CONFIG"
