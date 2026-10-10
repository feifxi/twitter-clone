#!/bin/sh
# Runs at container start (from /docker-entrypoint.d). Certbot renews the certificate
# from another container; this reloads nginx when the files change, so the new
# certificate is served without a restart or a Docker socket.
(
  last=$(cat /etc/nginx/tls/*.pem 2>/dev/null | md5sum)
  while sleep 60; do
    now=$(cat /etc/nginx/tls/*.pem 2>/dev/null | md5sum)
    # Only remember the new files once nginx accepted them: a reload in the moment
    # between certbot writing the certificate and the key is simply retried.
    if [ "$now" != "$last" ] && nginx -t -q && nginx -s reload; then
      last=$now
      echo "certificate changed: nginx reloaded"
    fi
  done
) &
