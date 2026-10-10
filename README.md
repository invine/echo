# Echo

Simple echo app for connectivity testing.

## GET /?message={message}

Returns message text sent as URL parameter.

## POST /

Body shoud contain JSON object with message parameter which will be returned in the response.

## GET /ping

Returns pong.

## Client address logging

Request logs retain `remoteAddr`, the immediate TCP peer. With
`TRUSTED_PROXY_CIDRS` set to a comma-separated list of IPv4/IPv6 CIDRs, requests
from those peers can additionally log `clientIp` and optional numeric
`clientPort`. For example, `TRUSTED_PROXY_CIDRS=10.0.4.0/22` trusts every caller
in that private subnet. The default is to trust no forwarded identity.

The ingress must overwrite `X-Echo-Client-IP` and `X-Echo-Client-Port` with the
source address and source port from its LB's PROXY protocol metadata, removing
the headers when that metadata is absent. Do not expose an enabled deployment
through an ingress that merely passes these headers through. Internal callers
in the configured CIDRs are trusted and can supply these values themselves;
this is a diagnostic identity, not an authentication mechanism.

Missing, malformed, duplicate, or untrusted IP metadata produces no client
fields; requests still run normally and retain `remoteAddr`. An invalid or
missing port omits only `clientPort`. `Forwarded`, `X-Forwarded-For`,
`X-Real-IP`, and `X-Forwarded-Port` are not identity sources. In particular,
`X-Forwarded-Port` usually describes a destination port such as 443.

The reported address is the public source reaching the LB; NAT or VPNs may
change it and its port. Successful echo/ping requests keep their existing
debug log level, so use `LOG_LEVEL=debug` to see them. Existing request headers
are still logged as raw diagnostic data, not as authoritative client fields.
