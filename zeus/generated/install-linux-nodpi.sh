#!/bin/bash
# generated by ZEUS v0.9.8
# Timestamp: [Wed Jul 7 01:48:52 2021]

VERSION="0.5.14"


mkdir -p bin
go build -tags nodpi -ldflags "-s -w" -o bin/net github.com/dreadl0ck/netcap/cmd
echo "setting capabilities for attaching to a network interface and moving binary to /usr/local/bin (requires root)..."
sudo setcap cap_net_raw,cap_net_admin=eip $(which net)
sudo mv bin/net /usr/local/bin
