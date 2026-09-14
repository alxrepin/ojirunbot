#!/bin/sh
set -eu

project=${PROJECT_DIR:-$(cd "$(dirname "$0")/.." && pwd)}
backup_dir=${BACKUP_DIR:-/srv/main/backups/ojirunbot}
keep=${BACKUP_KEEP:-5}
photo_volume=${PHOTO_VOLUME:-$(basename "$project")_photo_storage}
stamp=$(date -u +%Y%m%d-%H%M%S)

mkdir -p "$backup_dir"
cd "$project"

docker compose exec -T postgres pg_dump -U nutrition -d nutrition -Fc > "$backup_dir/db-$stamp.dump.part"
mv "$backup_dir/db-$stamp.dump.part" "$backup_dir/db-$stamp.dump"

docker run --rm --user "$(id -u):$(id -g)" \
    -v "$photo_volume":/photos:ro -v "$backup_dir":/backup alpine:3.22 \
    tar -czf "/backup/photos-$stamp.tar.gz.part" -C /photos .
mv "$backup_dir/photos-$stamp.tar.gz.part" "$backup_dir/photos-$stamp.tar.gz"

for prefix in db- photos-; do
    ls -1t "$backup_dir"/"$prefix"* | tail -n +$((keep + 1)) | xargs -r rm -f
done

echo "$(date -u +%FT%TZ) ok db=$(du -h "$backup_dir/db-$stamp.dump" | cut -f1) photos=$(du -h "$backup_dir/photos-$stamp.tar.gz" | cut -f1)"
