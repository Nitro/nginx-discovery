Nginx Discovery
===============

Service that connects to Sidecar on an ongoing basis and manages an Nginx
configuration to point to a single service by name.

Example configuration:

```bash
$ DISCOVERY_SIDECAR_ADDRESS=dev-singularity.uw2.nitro.us:7777 \
	DISCOVERY_FOLLOW_SERVICE=nginx-lazyraster \
	DISCOVERY_NGINX_CONF=/tmp/nginx.conf \
	go run main.go
```
