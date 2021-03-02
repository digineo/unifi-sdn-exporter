#!/bin/sh

systemctl daemon-reload
systemctl enable unifi-sdn-exporter
systemctl restart unifi-sdn-exporter
