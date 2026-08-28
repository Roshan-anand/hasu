
get_hasu_version() {
    echo "latest"
}

generate_jwt_secret() {
    # Generate a cryptographically secure random 64-character hex string
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 32
    else
        # Fallback: derive hex directly from /dev/urandom
        head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'
    fi
}

update_hasu() {
    # Detect version tag
    VERSION_TAG=$(detect_version)
    DOCKER_IMAGE="hasu:${VERSION_TAG}"

    echo "Updating Hasu to version: ${VERSION_TAG}"

    # Pull the image
    docker pull $DOCKER_IMAGE

    # Update the service
    docker service update --image $DOCKER_IMAGE hasu

    echo "Hasu has been updated to version: ${VERSION_TAG}"
}

install_hasu() {
    # take input of the domain name from the user
    read -p "Enter the domain name for Hasu (e.g., dashboard.example.com): " DOMAIN_NAME
    if [ -z "$DOMAIN_NAME" ]; then
        echo "Domain name cannot be empty. Please run the script again and provide a valid domain name."
        exit 1
    fi


    # Detect version tag
    VERSION_TAG=$(get_hasu_version)
    DOCKER_IMAGE="hasu:${VERSION_TAG}"

    echo "Installing hasu..."

    # check if is Mac OS
    if [ "$(uname)" = "Darwin" ]; then
        echo "This script must be run on Linux" >&2
        exit 1
    fi

    # check if is running inside a container
    if [ -f /.dockerenv ]; then
        echo "This script must be run on Linux" >&2
        exit 1
    fi

    # check if something is running on port 80
    if ss -tulnp | grep ':80 ' >/dev/null; then
        echo "Error: something is already running on port 80" >&2
        exit 1
    fi

    # check if something is running on port 443
    if ss -tulnp | grep ':443 ' >/dev/null; then
        echo "Error: something is already running on port 443" >&2
        exit 1
    fi

    # check if something is running on port 8080
    if ss -tulnp | grep ':8080 ' >/dev/null; then
        echo "Error: something is already running on port 8080" >&2
        echo "Hasu requires port 8080 to be available. Please stop any service using this port." >&2
        exit 1
    fi

    # check if docker is installed
    if command -v docker >/dev/null 2>&1; then
        echo "Docker is installed."
    else
        echo "Docker is not installed. Installing Docker..."
        curl -sSL https://get.docker.com | sh -s -- --version 29.6.2
        exit 1
    fi

    # TODO : Check compatibility for Proxmox LXC container

    docker swarm leave --force 2>/dev/null

    # get_ip() {
    #     local ip=""

    #     # Try IPv4 first
    #     # First attempt: ifconfig.io
    #     ip=$(curl -4s --connect-timeout 5 https://ifconfig.io 2>/dev/null)

    #     # Second attempt: icanhazip.com
    #     if [ -z "$ip" ]; then
    #         ip=$(curl -4s --connect-timeout 5 https://icanhazip.com 2>/dev/null)
    #     fi

    #     # Third attempt: ipecho.net
    #     if [ -z "$ip" ]; then
    #         ip=$(curl -4s --connect-timeout 5 https://ipecho.net/plain 2>/dev/null)
    #     fi

    #     # If no IPv4, try IPv6
    #     if [ -z "$ip" ]; then
    #         # Try IPv6 with ifconfig.io
    #         ip=$(curl -6s --connect-timeout 5 https://ifconfig.io 2>/dev/null)

    #         # Try IPv6 with icanhazip.com
    #         if [ -z "$ip" ]; then
    #             ip=$(curl -6s --connect-timeout 5 https://icanhazip.com 2>/dev/null)
    #         fi

    #         # Try IPv6 with ipecho.net
    #         if [ -z "$ip" ]; then
    #             ip=$(curl -6s --connect-timeout 5 https://ipecho.net/plain 2>/dev/null)
    #         fi
    #     fi

    #     if [ -z "$ip" ]; then
    #         echo "Error: Could not determine server IP address automatically (neither IPv4 nor IPv6)." >&2
    #         echo "Please set the ADVERTISE_ADDR environment variable manually." >&2
    #         echo "Example: export ADVERTISE_ADDR=<your-server-ip>" >&2
    #         exit 1
    #     fi

    #     echo "$ip"
    # }

    get_private_ip() {
        ip addr show | grep -E "inet (192\.168\.|10\.|172\.1[6-9]\.|172\.2[0-9]\.|172\.3[0-1]\.)" | head -n1 | awk '{print $2}' | cut -d/ -f1
    }

    advertise_addr="${ADVERTISE_ADDR:-$(get_private_ip)}"

    if [ -z "$advertise_addr" ]; then
        echo "ERROR: We couldn't find a private IP address."
        echo "Please set the ADVERTISE_ADDR environment variable manually."
        echo "Example: export ADVERTISE_ADDR=192.168.1.100"
        exit 1
    fi
    echo "Using advertise address: $advertise_addr"

    # initialize docker swarm
    swarm_init_args="${DOCKER_SWARM_INIT_ARGS:-}"

    if [ -n "$swarm_init_args" ]; then
        echo "Using custom swarm init arguments: $swarm_init_args"
        docker swarm init --advertise-addr $advertise_addr $swarm_init_args
    else
        docker swarm init --advertise-addr $advertise_addr
    fi

    if [ $? -ne 0 ]; then
        echo "Error: Failed to initialize Docker Swarm" >&2
        exit 1
    fi
    echo "Swarm initialized"

    docker network rm -f hasu_proxy 2>/dev/null
    docker network create --driver overlay --attachable hasu_proxy
    echo "Network created"

    mkdir -p /etc/hasu
    chmod 777 /etc/hasu

    # volumes for hasu
    docker volume create hasu-data 2>/dev/null || true
    docker volume create hasu-logs 2>/dev/null || true
    # docker volume create hasu-codebase 2>/dev/null || true
    docker volume create hasu-letsencrypt 2>/dev/null || true

    # generate a unique JWT secret for this installation
    JWT_SECRET=$(generate_jwt_secret)

    # create docker swarm service for hasu
    docker service create \
      --name hasu \
      --replicas 1 \
      --network hasu_proxy \
      --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
      --mount type=volume,src=hasu-data,dst=/app/hasu-data/sqlite \
      --mount type=volume,src=hasu-logs,dst=/app/hasu-data/badger \
      --mount type=volume,src=/etc/hasu/code,dst=/etc/hasu/code \
      --env JWT_SECRET="$JWT_SECRET" \
      --env APP_ENV=production \
      --env SERVER_PUBLIC_URL=https://$DOMAIN_NAME \
      --label traefik.enable=true \
      --label traefik.constraint-label=head-proxy \
      --label traefik.swarm.network=hasu_proxy \
      --label "traefik.http.routers.hasu.rule=Host(\`$DOMAIN_NAME\`)" \
      --label traefik.http.routers.hasu.entrypoints=websecure \
      --label traefik.http.routers.hasu.tls=true \
      --label traefik.http.routers.hasu.tls.certresolver=le \
      --label traefik.http.services.hasu.loadbalancer.server.port=8080 \
      --update-order start-first \
      --restart-condition on-failure \
      --restart-delay 5s \
      --restart-max-attempts 3 \
      $DOCKER_IMAGE

    sleep 4

    docker service create \
        --name hasu-traefik \
        --constraint 'node.role==manager' \
        --network hasu_proxy \
        --mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock,readonly \
        --mount type=bind,source=/etc/hasu/traefik/traefik.yml,target=/etc/traefik/traefik.yml,readonly \
        --mount type=bind,source=/etc/hasu/traefik/dynamic.yml,target=/etc/traefik/dynamic.yml,readonly \
        --mount type=volume,source=hasu-letsencrypt,target=/letsencrypt \
        --publish mode=host,published=443,target=443 \
        --publish mode=host,published=80,target=80 \
        traefik:v3.6.7

    GREEN="\033[0;32m"
    YELLOW="\033[1;33m"
    BLUE="\033[0;34m"
    NC="\033[0m"

    echo ""
    printf "${GREEN}Congratulations, Hasu is installed!${NC}\n"
    printf "${BLUE}Wait 15 seconds for the server to start${NC}\n"
    printf "${YELLOW}Please go to http://${formatted_addr}:8080${NC}\n\n"
}

# Entry point for the script
if [ "$1" = "update" ]; then
    update_hasu
else
    install_hasu
fi
