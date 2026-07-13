#!/usr/bin/env bash
# Builds all 3 Go binaries and all 3 frontend apps, laying them out
# under dist/ the same way they'd be copied to /opt/family-app-suite/*
# on the VPS (see deploy/systemd/*.service for the expected paths).
#
# Each frontend app (frontend/apps/<name>) is its own independent Vite
# build -- there is no single shared bundle anymore, so each one is
# copied only into its matching service's public/ dir.
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="${root_dir}/dist"

rm -rf "${out_dir}"
mkdir -p "${out_dir}/finance-svc/public" "${out_dir}/geochat-svc/public" "${out_dir}/fitness-svc/public"

echo "building frontend apps..."
(cd "${root_dir}/frontend" && npm install && npm run build)

echo "building Go services..."
(cd "${root_dir}/backend" && go build -o "${out_dir}/finance-svc/finance-svc" ./finance-svc)
(cd "${root_dir}/backend" && go build -o "${out_dir}/geochat-svc/geochat-svc" ./geochat-svc)
(cd "${root_dir}/backend" && go build -o "${out_dir}/fitness-svc/fitness-svc" ./fitness-svc)
(cd "${root_dir}/backend" && go build -o "${out_dir}/seed" ./cmd/seed)

echo "copying each app's own build into its matching service's public/ dir..."
cp -r "${root_dir}/frontend/apps/finance/dist/." "${out_dir}/finance-svc/public/"
cp -r "${root_dir}/frontend/apps/geochat/dist/." "${out_dir}/geochat-svc/public/"
cp -r "${root_dir}/frontend/apps/fitness/dist/." "${out_dir}/fitness-svc/public/"

echo "done. rsync ${out_dir}/<service>/ to /opt/family-app-suite/<service>/ on the VPS."
