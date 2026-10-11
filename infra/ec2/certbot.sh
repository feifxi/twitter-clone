#!/bin/sh
# Entry point of the certbot container: issues the certificate on the first run and
# renews it once it's within 30 days of expiry (checked twice a day). nginx answers
# the HTTP-01 challenge from the shared webroot and picks up the files in /tls.
set -u
live="/etc/letsencrypt/live/$SITE_DOMAIN"

while :; do
  # CERTBOT_FLAGS is a list of extra flags: split on purpose.
  # shellcheck disable=SC2086
  if certbot certonly --webroot -w /var/www/acme -d "$SITE_DOMAIN" --cert-name "$SITE_DOMAIN" \
    --email "$ACME_EMAIL" --agree-tos --no-eff-email --non-interactive --keep-until-expiring \
    --server "${ACME_SERVER:-https://acme-v02.api.letsencrypt.org/directory}" ${CERTBOT_FLAGS:-}; then
    # Copied (not linked): /tls is nginx's mount. Each file is replaced atomically.
    cp -L "$live/privkey.pem" /tls/privkey.pem.new && mv /tls/privkey.pem.new /tls/privkey.pem
    cp -L "$live/fullchain.pem" /tls/fullchain.pem.new && mv /tls/fullchain.pem.new /tls/fullchain.pem
    sleep 12h
  else
    # Usually DNS doesn't point here yet. Let's Encrypt allows 5 failed validations
    # per hostname per hour, so retry slower than that.
    echo "certbot: no certificate yet, retrying in 15 minutes" >&2
    sleep 15m
  fi
done
