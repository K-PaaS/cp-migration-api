#!/bin/sh
sudo podman image rm harbor.115.68.198.100.nip.io/cp-portal-repository/kps-migration-api
sudo podman build -t harbor.115.68.198.100.nip.io/cp-portal-repository/kps-migration-api:latest .
sudo podman push harbor.115.68.198.100.nip.io/cp-portal-repository/kps-migration-api:latest
