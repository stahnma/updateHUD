
# Goal

Build an agent to let me know when I have updates required on my home nodes.


# Build

These are things you can do. This works on MacOS, Linux on ARM, Linux on x86_64.

    `make build`

    `make arm`

    `make linux`


# Infrastructure

To see what's happening, install mosquitto broker and point the agents at it. I just used mqttexplorer to inpsect the messages.


# Lacking

There's no server (visual aggregator) for this yet.

# Alternatives

Run a cron job or systemd timer to auto update. The issue then is that sometimes reboots are required or docker crashes or whatever -- so I didn't want to do that.


