#pgsql.txt
sudo mkdir -p /opt/orthodoxy-liturgical-service

podman run -d \
  --name postgres18 \
  --restart=unless-stopped \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD='waxoo6ezae4I' \
  -e POSTGRES_DB=postgres \
  -p 127.0.0.1:5432:5432 \
  -v /opt/orthodoxy-liturgical-service:/var/lib/postgresql \
  docker.io/library/postgres:18

podman ps
podman logs -f postgres18
psql -h 127.0.0.1 -U postgres -d postgres

# Запуск после перезагрузки (контейнер уже создан командой выше):
podman start postgres18
podman ps                      # postgres18 Up, 127.0.0.1:5432->5432/tcp

# Чтобы контейнер поднимался сам после перезагрузки
# (у rootless podman --restart=unless-stopped без этого не срабатывает):
systemctl --user enable --now podman-restart.service
loginctl enable-linger $USER   # запуск без входа в систему

# Остановка:
podman stop postgres18
