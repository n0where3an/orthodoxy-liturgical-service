# pgsql.md

## 1. Создание и запуск PostgreSQL 18

```bash
sudo mkdir -p /opt/orthodoxy-liturgical-service

podman run -d \
  --name postgres18 \
  --restart=always \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD='waxoo6ezae4I' \
  -e POSTGRES_DB=postgres \
  -p 127.0.0.1:5432:5432 \
  -v /opt/orthodoxy-liturgical-service:/var/lib/postgresql \
  docker.io/library/postgres:18-alpine
```

Образ на Alpine (musl). Каталог данных, созданный образом на Debian (glibc),
к нему не подключать: базы созданы с правилом сортировки `en_US.utf8` из
libc, а в musl оно сортирует иначе, и индексы по тексту окажутся
неверными. Переход с Debian-образа — через дамп и восстановление
(разделы 5 и 3) в новый, пустой каталог данных.

Проверка:

```bash
podman ps
podman logs -f postgres18
```

Подключение:

```bash
psql -h 127.0.0.1 -U postgres -d postgres
```

## 2. Настройка автоматического запуска

Для rootless Podman. `podman-restart.service` поднимает после перезагрузки
только контейнеры с политикой `always`
(`podman start --all --filter restart-policy=always`); контейнер с
`--restart=unless-stopped` он не запускает.

```bash
# Политика перезапуска уже созданного контейнера
podman update --restart=always postgres18
podman inspect postgres18 --format '{{.HostConfig.RestartPolicy.Name}}'   # always

# Автоматический перезапуск контейнеров после перезагрузки
systemctl --user enable --now podman-restart.service

# Запуск пользовательских сервисов без входа в систему
loginctl enable-linger ${USER}
```

## 3. Восстановление базы данных из liturgy.dump

Дамп создаётся на сервере с PostgreSQL 15:

```bash
podman exec pg pg_dump \
  -F c \
  -U liturgy \
  liturgy > ./liturgy.dump
```

Скопировать `liturgy.dump` на машину с PostgreSQL 18 в текущий каталог.

### Создание пользователя и базы данных

```bash
# Создать пользователя liturgy
podman exec postgres18 psql -U postgres -d postgres -c \
  "CREATE USER liturgy WITH ENCRYPTED PASSWORD 'liturgy';"

# Создать базу данных с владельцем liturgy
podman exec postgres18 createdb \
  -U postgres \
  -O liturgy \
  liturgy
```

### Копирование и восстановление дампа

```bash
podman cp liturgy.dump postgres18:/tmp/liturgy.dump

podman exec postgres18 pg_restore \
  -U liturgy \
  -d liturgy \
  --no-owner \
  --no-acl \
  /tmp/liturgy.dump
```

### Проверка восстановления

```bash
# Проверка схем
podman exec postgres18 psql -U postgres -d liturgy -c "\dn"

# Проверка таблиц (схема public)
podman exec postgres18 psql -U postgres -d liturgy -c "\dt public.*"
```

## 4. Управление контейнером

```bash
# Просмотр контейнеров
podman ps

# Просмотр логов
podman logs -f postgres18

# Остановка
podman stop postgres18

# Запуск
podman start postgres18

# Подключение к PostgreSQL
psql -h 127.0.0.1 -U postgres -d postgres

# Подключение к базе проекта
psql -h 127.0.0.1 -U liturgy -d liturgy
```

## 5. Резервное копирование

Создание резервной копии базы проекта:

```bash
podman exec postgres18 pg_dump \
  -U liturgy \
  -d liturgy \
  -F c > liturgy.dump
```

Восстановление из резервной копии (с предварительным пересозданием базы;
`dropdb` не пройдёт, пока к базе подключены `liturgy serve` или `psql`):

```bash
# Удаление старой базы
podman exec postgres18 dropdb \
  -U postgres \
  --if-exists \
  liturgy

# Создание новой базы
podman exec postgres18 createdb \
  -U postgres \
  -O liturgy \
  liturgy

# Копирование дампа
podman cp liturgy.dump postgres18:/tmp/liturgy.dump

# Восстановление
podman exec postgres18 pg_restore \
  -U liturgy \
  -d liturgy \
  --no-owner \
  --no-acl \
  /tmp/liturgy.dump
  