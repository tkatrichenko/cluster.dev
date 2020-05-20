#!/usr/bin/env sh

# .ssh dir should be mounts with same rights as user on host. Can't be accessible with other rights.
addgroup -g $GID cluster.dev
adduser -u $UID -D -G cluster.dev cluster.dev
su - cluster.dev -c "cd /app/ && python cluster_dev.py $*"
