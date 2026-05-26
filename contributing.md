## prerequisites

- `docker` v29.x
- `docker-compose`

## setting up local env

- add `127.0.0.1  *.godploy.localhost` in new line to your `/etc/hosts` file

## running the services

- run `make env` to create `.env` files
- run `make cloud-tunnel` to start cloudflared tunnel for local https support
  - copy the `https://<custom_generated>.trycloudflare.com` url
  - paste it in `.env` file in the root folder in `SERVER_PUBLIC_URL` var
- run `make setup` to setup traefik and build needed docker images
- run `make server-start` to start the server service
- run `make web-start` to start the web service
- you can access services at
  - Traefik dashboard : `https://traefik.godploy.localhost` (to access the dashboard username : `godploy`, password : `godploy`)
  - Godploy web : `https://localhost:3000`
  - Godploy server : `https://localhost:8000` || `https://<custom_generated>.trycloudflare.com`

## watch the services

- run `make web-logs` to watch the web service logs
- run `make server-logs` to watch the server service logs
- run `make traefik-logs` to watch the traefik service logs

## stopping the development environment

- run `make dev-stop` to stop dev services
- run `make services-rm` to stop and remove the traefik

## errors

- while running `make check` or `bun check` in frontend, if you get permission error, then run `make permission` and now its fixed.
