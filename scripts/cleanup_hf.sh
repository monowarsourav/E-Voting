#!/bin/bash
echo "Stopping and cleaning HF network..."
cd "$(dirname "$0")/../network"
if command -v docker-compose >/dev/null 2>&1; then
    docker-compose -f docker-compose-hf.yml down -v
else
    docker compose -f docker-compose-hf.yml down -v
fi
docker rm -f $(docker ps -aq --filter label=service=hyperledger-fabric) 2>/dev/null || true
docker rmi -f $(docker images -q --filter reference='dev-peer*') 2>/dev/null || true
echo "Done!"
