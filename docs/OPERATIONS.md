# Эксплуатация и сохранение данных

Приложение работает одним Go-процессом. PostgreSQL — основное хранилище; файлы доказательств находятся в отдельном приватном volume `evidence`. Учётные данные первого входа — в приватном `credentials`. Исходные JSON/CSV не заменяют существующую БД при перезапуске.

## Обновить приложение без сброса данных

Из корня репозитория, PowerShell:

```powershell
git pull --ff-only origin main
$env:BUILD_VERSION = git rev-parse --short HEAD
docker compose build careerquest
docker compose up -d --wait postgres
docker compose up -d --no-deps careerquest
Invoke-RestMethod http://localhost:8080/api/health
```

Bash: `export BUILD_VERSION=$(git rev-parse --short HEAD)`, остальные Docker-команды совпадают; для проверки используйте `curl http://localhost:8080/api/health`.

Перед обновлением сделайте резервную копию по инструкции ниже. Прикладные миграции применяются сервером при запуске под транзакционной блокировкой. Новая схема требует совместимой версии приложения. Не редактируйте уже применённые файлы миграций; добавляйте следующие номера.

`git pull` меняет исходный код, но не пересобирает Docker image. В production Go отдаёт frontend из образа; bind mount исходников отсутствует. Поэтому после изменения React нужен `docker compose build careerquest` и пересоздание сервиса. Обновление страницы не заменяет эти шаги. `BUILD_VERSION` позволяет сравнить сборку с Git.

## Перенести старый JSON-прогресс в PostgreSQL

Это одноразовая процедура для обновления с раннего MVP. Нельзя сначала запустить новое приложение или provision accounts: первый запуск инициализирует пустую БД из seed. Мигратор намеренно откажется перезаписывать непустую БД.

1. Остановите старый сервис, сохранив контейнер и его volume. Скопируйте фактический runtime, а не только seed.

```powershell
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$backupDir = Join-Path (Get-Location) "data/backups/legacy-$stamp"
New-Item -ItemType Directory -Force -Path $backupDir | Out-Null
$oldContainer = docker compose ps -q careerquest
docker compose stop careerquest
docker cp "${oldContainer}:/app/data/runtime/state.json" (Join-Path $backupDir 'state.json')
Get-FileHash (Join-Path $backupDir 'state.json') -Algorithm SHA256
```

Если `state.json` отсутствует, в этом Docker volume ещё не было сохранённых изменений; используйте исходный набор для новой установки. При локальном старом запуске копируйте `data/runtime/state.json` отдельно: локальный JSON и Docker volume могли содержать разные изменения. Не объединяйте их автоматически.

2. Подготовьте `.env` и seed через `npm run setup -- --data-dir "<папка-набора>"`. Setup сохраняет существующие значения `.env`, поэтому проверьте, что `DATABASE_URL` указывает на **целевую пустую БД**, а не рабочую БД другой команды.

3. Поднимите только PostgreSQL, соберите приложение и проверьте перенос без записи:

```powershell
docker compose up -d --wait postgres
docker compose build careerquest
$legacyMount = "${backupDir}:/legacy:ro"
docker compose run --rm --no-deps -v $legacyMount careerquest /app/migrate-legacy --state /legacy/state.json --dry-run
```

4. Если dry-run показал ожидаемую ревизию, количество сотрудников/событий/истории и демо-завершений:

```powershell
docker compose run --rm --no-deps -v $legacyMount careerquest /app/migrate-legacy --state /legacy/state.json --apply
docker compose run --rm --no-deps careerquest /app/provision-accounts --output /app/data/credentials/accounts.json
docker compose up -d careerquest
```

Мигратор проверяет весь snapshot, сохраняет исходные ID, описания событий, значения полей, порядок, ревизию и флаг `demo`. После записи он читает БД и сравнивает поля перед COMMIT. Контрольная сумма исходного файла записывается в той же транзакции. Повтор с тем же файлом сообщает `already_imported=true` и не перезаписывает дальнейшую работу. Другой snapshot в непустую БД отклоняется.

Для локального Go вместо контейнерной команды доступны:

```powershell
npm run migrate:legacy -- --state "<полный-путь-к-state.json>" --dry-run
npm run migrate:legacy -- --state "<полный-путь-к-state.json>" --apply
```

Эти команды берут `DATABASE_URL` из `.env`. JSON остаётся неизменным. Старый progress volume сохраняйте до проверки всех профилей и дальнейшего периода эксплуатации. Не удаляйте volumes ради обновления.

Если PostgreSQL уже содержит данные, остановитесь на сравнении и резервном копировании: мигратор не является инструментом объединения двух независимых историй. Создайте отдельную новую целевую БД для проверки переноса либо согласуйте явное разрешение конфликтов записей.

## Резервная копия PostgreSQL и доказательств

Для согласованного снимка временно остановите приложение; PostgreSQL может работать. Выполняйте из корня проекта в PowerShell:

```powershell
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$backupDir = Join-Path (Get-Location) "data/backups/$stamp"
New-Item -ItemType Directory -Force -Path $backupDir | Out-Null
docker compose stop careerquest
docker compose run --rm --no-deps --user 0 -v "${backupDir}:/backup" careerquest sh -c 'umask 077; tar -C /app/data -czf /backup/evidence.tgz evidence'
$pgContainer = docker compose ps -q postgres
docker compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc -f /tmp/careerquest.dump'
docker cp "${pgContainer}:/tmp/careerquest.dump" (Join-Path $backupDir 'careerquest.dump')
docker compose start careerquest
```

Архив создаётся одноразовым контейнером после остановки сервиса. Только этот контейнер запускается с пользователем 0 для записи в локальный каталог резервных копий; основной сервис остаётся непривилегированным. Не копируйте бинарный `pg_dump` через текстовый PowerShell pipeline: `docker cp` сохраняет байты.

Каталог `data/backups` исключён из Git. Экспорт содержит внутренние данные и хеши паролей; храните его в приватном месте. Отдельно сохраните `.env` и приватный экспорт первоначальных паролей с ограниченным доступом. Не прикладывайте их к README, issue или скриншотам.

## Проверить восстановление

Восстанавливайте сначала в **отдельную пустую БД** и отдельный evidence volume. Не используйте `--clean` против рабочей БД.

```powershell
$pgContainer = docker compose ps -q postgres
docker cp "<полный-путь-к-careerquest.dump>" "${pgContainer}:/tmp/restore.dump"
docker compose exec -T postgres sh -c 'createdb -U "$POSTGRES_USER" careerquest_restore_check'
docker compose exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d careerquest_restore_check --exit-on-error /tmp/restore.dump'
```

Для проверки приложения укажите URL этой БД отдельному QA-процессу, другой порт (например 8081) и отдельный `EVIDENCE_DIR`. Распакуйте архив доказательств в этот каталог, проверьте вход, профиль, истории и скачивание вложения. Работающий основной сервис не должен использовать восстановленную БД до завершения проверки.

## Частые проблемы

| Симптом                                  | Проверка и действие                                                                                                                                                                                            |
| ---------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| После pull прежний интерфейс             | `docker compose images`, версия `/api/health`, пересборка образа и `up -d --no-deps careerquest`; затем обновление страницы.                                                                                   |
| Порт 8080 занят                          | `Get-NetTCPConnection -LocalPort 8080` в PowerShell; не запускайте локальный Go и Docker-приложение одновременно.                                                                                              |
| БД недоступна                            | `docker compose ps`, `docker compose logs --tail 50 postgres`; сверить host/port в `DATABASE_URL` без публикации строки с паролем.                                                                             |
| Пароль PostgreSQL изменён в .env         | Изменение `.env` не меняет пароль существующего пользователя в volume; верните согласованные настройки либо смените пароль штатно в PostgreSQL.                                                                |
| Конфликт `stale_revision`                | Второй сервер изменил общий снимок. Оставьте один экземпляр и перезапустите приложение; не удаляйте БД.                                                                                                        |
| Мигратор отказывается писать             | Целевая БД непуста или snapshot невалиден. Сохраните оба источника и разберите конкретную ошибку; не сбрасывайте данные.                                                                                       |
| Пустые описания старой PostgreSQL-сборки | Миграция добавляет поле, но не выдумывает утраченное содержимое. Восстановите описания из сохранённого каталога явной миграцией по event_id с проверкой; новый импорт и первичный seed сохраняют их полностью. |
| Нет первоначального пароля               | Хеши необратимы. Найдите приватный экспорт provision accounts; повторная подготовка не восстанавливает старый пароль.                                                                                          |
| AI работает в резервном режиме           | Проверить `LLM_BASE_URL`, `LLM_MODEL`, серверный ключ и доступность провайдера. Расчёты навыков и EXP остаются серверными.                                                                                     |
