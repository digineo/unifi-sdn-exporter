#!/bin/sh

case "$1" in
    remove)
        systemctl disable unifi-sdn-exporter || true
        systemctl stop unifi-sdn-exporter || true
    ;;
esac
