#!/bin/bash

set -euo pipefail

echo 'begin acceptance test suite'
find ./tests -mindepth 1 -type d | while read -r path; do
  title="$(basename "$path")"
  echo "$title"
  mysql="mysql -h mysql -uroot -proot"
  echo '  -> prepare'
  $mysql -e "drop database if exists "'`'"$title"'`;'"create database "'`'"$title"'`;'
  echo '  -> seed'
  $mysql "$title" < "$path/setup.sql"
  echo '  -> dumping'
  if ! contents="$(go run . -hmysql -uroot -proot -c "$path/config.hcl")"; then
    echo "❌ $title"
    continue
  fi
  echo '  -> importing'
  if ! $mysql <<< "$contents"; then
    echo "❌ $title"
    continue
  fi
  echo "✅ $title"
done
