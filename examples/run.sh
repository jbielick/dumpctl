#!/bin/bash

set -euo pipefail

find examples -mindepth 1 -type d -name "${1:-*}" | while read -r dir; do
  echo "$dir"
  mysql -uroot -proot -P3307 --protocol=tcp --database myapp_production < "./$dir/data.sql"
  go run . -c "$dir/$(basename $dir).hcl" -uroot -proot -P3307 > "$dir/result.sql"
done
