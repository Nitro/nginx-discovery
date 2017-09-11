Nginx Discovery
===============

Service that connects to Sidecar on an ongoing basis and manages an Nginx
configuration to point to a single service by name.

Recognised environment variables:

Variable                              | Mandatory  | Default
--------------------------------------| ---------- | -------------------
DISCOVERY_REFRESH_INTERVAL            | no         | 5s
DISCOVERY_FOLLOW_SERVICE              | no         | lazyraster
DISCOVERY_FOLLOW_PORT                 | yes        |
DISCOVERY_TEMPLATE_FILENAME           | no         | templates/nginx.conf.tmpl
DISCOVERY_UPDATE_COMMAND              | no         |
DISCOVERY_VALIDATE_COMMAND            | no         |
DISCOVERY_SIDECAR_ADDRESS             | yes        |
DISCOVERY_NGINX_CONF                  | no         | /nginx/nginx.conf

Example configuration:

```bash
$ DISCOVERY_SIDECAR_ADDRESS=dev-singularity.uw2.nitro.us:7777 \
	DISCOVERY_FOLLOW_SERVICE=lazyraster \
	DISCOVERY_FOLLOW_PORT=10109 \
	DISCOVERY_NGINX_CONF=/tmp/nginx.conf \
	go run main.go
```
