#!/bin/sh

# Generated by: tyk-ci/wf-gen
# Generated on: Fri Dec 17 15:28:00 UTC 2021

# Generation commands:
# ./pr.zsh -p -title sync for m4 templates -branch pkr176 -base pkr176 -repos tyk-pump
# m4 -E -DxREPO=tyk-pump


if command -V systemctl >/dev/null 2>&1; then
    if [ ! -f /lib/systemd/system/tyk-pump.service ]; then
        cp /opt/tyk-pump/install/inits/systemd/system/tyk-pump.service /lib/systemd/system/tyk-pump.service
    fi
else
    if [ ! -f /etc/init.d/tyk-pump ]; then
        cp /opt/tyk-pump/install/inits/sysv/init.d/tyk-pump /etc/init.d/tyk-pump
    fi
fi
