#!/bin/bash

BRIDGE_IP="10.0.0.1/8"

if [[ $EUID -ne 0 ]]; then
	echo "You must be root to run this script."
	exit 1
fi

ip link add name br1 type bridge
ip addr add $BRIDGE_IP dev br1
ip link set dev br1 up

# br1 is the container bridge
# eno1 is the internet device
iptables -t nat -A POSTROUTING -o eno1 -j MASQUERADE
iptables -A FORWARD -i eno1 -o br1 -j ACCEPT
iptables -A FORWARD -o eno1 -i br1 -j ACCEPT
