# Jamulus recording toggle service

A tiny web service for toggling the recording state on your Jamulus server, by simply sending a SIGUSR2 signal to the process.
Only works on Linux servers and must be run in a way such that the service can find the Jamulus process and send the kill signal to the Jamulus process obviously.

Of course this is only a workaround for now, ultimately this should be built into Jamulus itself, for example by integrating with the Chat feature instead.

Note that votes for changing the recording state are simply based on the request source IP, so a reverse proxy is not going to work for toggle threshold > 1.

Usage:

```bash
Usage of /usr/local/bin/jamulus-ui:
  -index-page string
        The index page file to display at the root.
  -listen-address string
        The address the HTTP server should listen on.
  -process-name string
        Process name to scan for (default "Jamulus")
  -toggle-threshold int
        The number of votes necessary to toggle recording. (default 2)
```

## Installing

- Download and adjust the parameters of the [systemd unit](jamulus-toggle.service)
- Install the unit:
```bash
cp jamulus-toggle.service /etc/systemd/system/
systemctl daemon-reload
```
- Download and install the static index page
```bash
mkdir -p /opt/jamulus-ui
chown -R jamulus:jamulus /opt/jamulus-ui
cp index.html /opt/jamulus-ui
```
- Get the latest binary from the releases and install it
```bash
cp jamulus-toggle /usr/local/bin/
# Ensure it can be run by the jamulus user
chmod a+x /usr/local/bin/jamulus-toggle
```
- Start the service
```bash
systemctl start jamulus-toggle
# Optionally, enable at boot time
systemctl enable jamulus-toggle
```