#!/usr/bin/env bash
# Builds all 3 Go binaries and the shared frontend, laying them out
# under dist/ the same way they'd be copied to /opt/family-app-suite/*
# on the VPS (see deploy/systemd/*.service for the expected paths).
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="${root_dir}/dist"

rm -rf "${out_dir}"
mkdir -p "${out_dir}/finance-svc/public" "${out_dir}/geochat-svc/public" "${out_dir}/fitness-svc/public"

echo "building frontend..."
(cd "${root_dir}/frontend" && npm install && npm run build)

echo "building Go services..."
(cd "${root_dir}/backend" && go build -o "${out_dir}/finance-svc/finance-svc" ./finance-svc)
(cd "${root_dir}/backend" && go build -o "${out_dir}/geochat-svc/geochat-svc" ./geochat-svc)
(cd "${root_dir}/backend" && go build -o "${out_dir}/fitness-svc/fitness-svc" ./fitness-svc)
(cd "${root_dir}/backend" && go build -o "${out_dir}/seed" ./cmd/seed)

echo "copying shared frontend build into each service's public/ dir..."
cp -r "${root_dir}/frontend/dist/." "${out_dir}/finance-svc/public/"
cp -r "${root_dir}/frontend/dist/." "${out_dir}/geochat-svc/public/"
cp -r "${root_dir}/frontend/dist/." "${out_dir}/fitness-svc/public/"

echo "done. rsync ${out_dir}/<service>/ to /opt/family-app-suite/<service>/ on the VPS."
