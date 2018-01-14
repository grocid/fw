# FW

Forwards connections using whitelisting via Google Authenticator. It works with TLS connections, simply by just passing on the packets. Of course, this requires that packets are redirected to the same machine.

## Example

Assume that we have an instance of e.g. Hashicorp's Vault running behind a firewall and we want to limit access to it. First, we invoke

```
$ fw 0.0.0.0:9999 127.0.0.1:8200
```

This will create a token file `token` containg a Base32-encoded secret. Putting the secret into Google Authenticator will allow us to generate TOTP codes.

If we try to access Vault via

```
$ curl https://yourdomain.com:9999/v1/sys/seal-status
```

this will immediately drop the connection. To whitelist our IP, we need to authenticate. This is done as

```
$ curl https://yourdomain.com:8000/auth -d '{"token": "<token from Google Authenticator>"}'

{"authenticated": true}
```

Good -- now, we are whitelisted for 7 days!

```
$ curl https://yourdomain.com:9999/v1/sys/seal-status

{"request_id":"419d889c-7f7a-d818-45a0-01c73f520d6e","lease_id":"",...
```

## Threats

It *is* possible to bruteforce the token. There are only 1,000,000 possible tokens. Blacklisting of bruteforce is not a thing in this code, but could be implemented. The simplest measure is to set a small validity window.