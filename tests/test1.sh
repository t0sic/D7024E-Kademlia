#!/bin/bash

# Stop all running containers
docker stop $(docker ps -aq)

# Remove all containers
docker rm $(docker ps -aq)

docker image prune -a
cd ..
docker compose -f docker-compose.test.yml up --build --remove-orphans --scale node=50 --abort-on-container-exit --exit-code-from tester