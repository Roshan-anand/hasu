## prerequisites

- `go` v1.26.x
- `bun` v1.3.x
- `turbo` v2.10.x
- `cloudflared`
- `docker` v29.x
- `docker-compose`

## setting up local env

- add `127.0.0.1  *.godploy.localhost` in new line to your `/etc/hosts` file

## running the services

- run `make env` to create `.env` files
- Fill up the `apps/web/.env` and `apps/server/.env`
- Setup a domain for local tunnel using `cloudflared`
  - paste it in `apps/server/.env` file in `SERVER_PUBLIC_URL` var
  - see the setup guide in to bottom.
- run `make install` to install all dependency for the project.
- run `make setup` to setup traefik and build needed docker images
- run `make start` to start all services
- you can access services at
  - Traefik dashboard : `https://traefik.godploy.localhost` (to access the dashboard username : `godploy`, password : `godploy`)
  - Godploy web : `https://localhost:3000`
  - Godploy server : `https://localhost:8000` || `https://<custom_dev_domain>`

## watch the services

- run `make web-logs` to watch the web service logs
- run `make server-logs` to watch the server service logs
- run `make traefik-logs` to watch the traefik service logs

## stopping the development environment

- run `make stop` to stop all service
- run `make services-rm` to stop and remove the traefik

<hr/>

## permanent subdomain for dev

it get's really tricky to manage external service like Github with dynamic urls from cloudflare tunnel. so if u have a domain then use a permanent subdomain for local development.

- Login and create a named tunnel

  ```bash
    cloudflared tunnel login
    cloudflared tunnel create dev
  ```

- Create a DNS record for the subdomain

  ```bash
    cloudflared tunnel route dns dev dev.<your-domain>.com
  ```

- Configure the tunnel
  - Create ~/.cloudflared/config.yml

  ```yaml
  tunnel: dev
  credentials-file: /home/your-user/.cloudflared/<tunnel-id>.json

  ingress:
    - hostname: dev.<your-domain>.com
      service: http://localhost:8080

    - service: http_status:404
  ```

- Start the tunnel
  ```bash
    cloudflared tunnel run dev
  ```
