#!/bin/sh
for host in /sys/class/scsi_host/host*/scan ; do
        echo "- - -" > $host
      done
