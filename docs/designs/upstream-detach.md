# Отвязка форка от upstream: aghelm рядом с agent-manager

*   **Автор:** Maksim Ivanov
*   **Статус:** Proposed
*   **Дата:** 2026-11-22

--------------------------------------------------------------------------------

## Overview

`Doremi203/aghelm` — личный форк `YoanWai/agent-manager`, уже выведенный
из сети форков GitHub. Репозиторий переименован из `Doremi203/agent-manager`
в `Doremi203/aghelm`; GitHub перенаправляет старый адрес. Код при этом
остаётся копией upstream: тот же путь модуля Go, то же имя бинарника, тот же каталог конфигурации, сокет
tmux, имя MCP-сервера, переменные окружения, адреса ленты и апдейтера.

Документ описывает, как сделать из форка отдельную программу `aghelm`. Она
работает на той же машине и под тем же пользователем, что и установленный
upstream, ничего с ним не делит, ничего не запрашивает у его хостов и ничего
туда не отправляет. Так обе программы можно запускать параллельно и сравнивать.

## The Problem

Если собрать форк и запустить его рядом с upstream, две копии мешают друг другу
на нескольких уровнях:

1.  **Общее состояние.** Обе копии читают и пишут одну `state.db` (каталог
    `agent-manager` в пользовательском конфиге) и подключаются к одному сокету
    tmux (`-L agentmgr`, объявлен в `internal/tmux/tmux.go` и отдельным
    литералом в `internal/remote/ssh.go`). Поллер одной копии может забрать
    сессии другой, пометить их мёртвыми или перезапустить.
2.  **Общее имя в окружении агентов.** Агенту в сессии велят вызывать
    `agent-manager` по имени: в подсказках при запуске, в словаре команд, в
    справке CLI. В PATH он найдёт бинарник upstream. Переменные
    `AGENT_MANAGER_*` (`CONFIG_DIR`, `SESSION_ID` и другие) наследуются
    дочерними процессами и указывают на чужую базу и чужую сессию.
3.  **Общие записи в конфигах CLI.** MCP-сервер регистрируется в глобальных
    конфигах нескольких CLI под одним именем. Файлы хуков и плагинов
    (`agent-manager-pi.ts` и другие) пишутся по одинаковым путям: кто записал
    последним, тот и владеет. Bundle ID нотификатора macOS
    (`dev.agent-manager.notifier`), звуки уведомлений, временный каталог
    вставок и кэш `sigstore-tuf` тоже общие.
4.  **Установка затирает upstream.** `go install .` берёт имя бинарника из
    последнего элемента пути модуля и ставит `agent-manager` в `GOBIN` поверх
    копии upstream.
5.  **Исходящие запросы к upstream.** Апдейтер, лента сообщений, проверка
    провенанса, каналы пакетных менеджеров (`yoanwai/tap`, scoop, AUR) и
    `report_issue` ходят на `wainmann/agent-manager` или `YoanWai`.
    Самообновление заменило бы бинарник форка бинарником upstream.
6.  **Инфраструктура upstream в репозитории.** Workflows, упаковка, шаблоны
    issue и PR, GOVERNANCE, MAINTAINERS, CODEOWNERS, FUNDING и переводы README
    описывают проект upstream. Правила из AGENTS.md (границы UI, имена и размер
    файлов, staticcheck) проверяет только `ci.yml`. Если удалить workflows и
    ничем их не заменить, эти правила перестанут проверяться.

## Goals

*   **Нет исходящих запросов к upstream.** При настройках по умолчанию aghelm
    не обращается ни к одному хосту upstream: ни к ленте, ни к проверке
    обновлений, ни к провенансу, ни к каналам пакетных менеджеров. Процесс
    aghelm не открывает соединений с внешними хостами, кроме явных действий
    пользователя: установка CLI (`deps.go`, `npm.go`) и SSH-подключение.
    Loopback к локальному серверу каталога моделей и сетевые запросы дочерних
    CLI сюда не относятся. Проверяется наблюдением за соединениями процесса,
    пока пользователь открывает панель сообщений, Settings и справку и
    создаёт сессию.
*   **Работа рядом с upstream.** aghelm и agent-manager работают одновременно
    под одним `HOME` и не мешают друг другу. Раздельно всё: сокет tmux (в том
    числе по SSH), каталог конфигурации и `state.db`, префикс переменных
    окружения, имя MCP-сервера, файлы хуков и плагинов, bundle ID и звуки
    уведомлений, временные каталоги, имя удалённого бинарника, путь установки.
*   **Код и документация принадлежат форку.** Путь модуля
    `github.com/Doremi203/aghelm`. В репозитории нет `agent-manager`,
    `YoanWai` и `wainmann`, кроме мест из списка исключений `check-identity`
    (Technical Plan): LICENSE, NOTICE, фраза об атрибуции в README,
    записанные экраны CLI и этот документ.
*   **Правила AGENTS.md проверяются локально.** `make check` запускает gofmt,
    vet, staticcheck, три архитектурные проверки, `check-identity` и тесты с
    `-race` под `env -u TMUX TMUX_TMPDIR=/tmp/amtest SHELL=/bin/sh`, как в
    `ci.yml`.
*   **Диалог `minimum_version` для CLI продолжает работать** после удаления
    апдейтера. Сравнение версий нужно ему, а не самообновлению.

## Non-Goals

*   **Собственный конвейер релизов и апдейтер.** Появятся позже, отдельным
    дизайном. Схема тегов версий форка (сейчас `git describe` наследует теги
    upstream `v0.x`) тоже решается тогда.
*   **Публикация в Homebrew и других менеджерах пакетов.** `.goreleaser.yaml`
    удаляется вместе с install-скриптами; конфигурацию упаковки можно
    восстановить из истории git при проектировании релизов.
*   **Перенос состояния из upstream.** aghelm начинает с пустой `state.db`.
    Команды импорта, копирования базы или выборочного переноса настроек нет.
*   **Синхронизация с upstream в будущем.** Remote `upstream` удаляется.
    Отсутствие механизма слияния — принятая цена.
*   **Чистка записей upstream в глобальных конфигах CLI.** Записи `agent-manager`
    принадлежат установленной копии upstream, aghelm их не трогает.
*   **Перевод документации.** Переводы README на японский и китайский
    удаляются, а не обновляются.
*   **Матрица совместимости с опубликованными релизами.** `matrix.py`
    сравнивает только с бинарником, указанным явно.

## Open Questions

*   **Схема тегов версий форка.** Не блокирует эту работу, решается до первого
    собственного релиза.

## Critical User Journeys (CUJs)

### CUJ 1: Две копии рядом

*   **Given**: установлен upstream `agent-manager` с живыми сессиями. В PATH
    есть `aghelm`, собранный из форка.
*   **When**: пользователь запускает `aghelm` и открывает в нём сессию Claude
    Code.
*   **Then**: aghelm создаёт свою пустую `state.db` в каталоге `aghelm` и
    сервер tmux на своём сокете. Сессии upstream в нём не видны и не меняются.
    Агент в сессии aghelm вызывает `aghelm`, видит `AGHELM_SESSION_ID` и
    MCP-сервер `aghelm`. MCP-сервер `agent-manager`, если CLI запустит его
    в сессии aghelm, работает в режиме без вызывающего и отказывает в
    действиях над сессиями. Так же и при запуске aghelm из панели upstream.

### CUJ 2: Обе копии в одном глобальном CLI

*   **Given**: установлены upstream и aghelm, в обоих открыты сессии CLI с
    глобальной MCP-регистрацией (например, Gemini).
*   **When**: каждая копия регистрирует свой MCP-сервер и хуки.
*   **Then**: в глобальном конфиге две записи под разными именями. Файлы хуков
    лежат по разным путям и не перезаписывают друг друга. Каждый сервер
    действует только на сессии своей копии, если upstream запущен не из
    панели aghelm (см. Risks & Mitigations).

### CUJ 3: Нет сетевых запросов к upstream

*   **Given**: свежий запуск aghelm с настройками по умолчанию, трафик
    наблюдается.
*   **When**: пользователь открывает панель сообщений, Settings и справку и
    создаёт сессию.
*   **Then**: нет ни одного запроса к `wainmann`, `YoanWai`, `api.github.com/repos/`
    или к каналам пакетных менеджеров upstream. Процесс aghelm не
    соединяется с внешними хостами, кроме явной установки CLI (`deps.go`,
    `npm.go`) и SSH.

### CUJ 4: Установка из исходников

*   **Given**: клон форка.
*   **When**: разработчик выполняет `make build`, `make install` или
    `go install github.com/Doremi203/aghelm@latest`.
*   **Then**: получается бинарник `aghelm`. Бинарник `agent-manager` в `GOBIN`
    остаётся нетронутым. Install-скриптов нет, установка только из исходников.

### CUJ 5: Проверка перед коммитом без CI

*   **Given**: в репозитории нет `.github/workflows`.
*   **When**: разработчик выполняет `make check`.
*   **Then**: проходят gofmt, vet, staticcheck, `check-ui-boundaries`,
    `check-file-names`, `check-file-sizes`, `check-identity` и тесты с `-race` под
    `env -u TMUX TMUX_TMPDIR=/tmp/amtest SHELL=/bin/sh`. Нарушение любого из
    этих правил возвращает ненулевой код выхода.

## Technical Plan (Architecture)

Изменение состоит из четырёх частей. Первая: **переименование** всего, что
видят пользователь, CLI и ОС. Вторая: **удаление** кода, который ходит к
upstream или существует только ради него. Третья: **замена CI** локальной
проверкой. Четвёртая: **документация**. Новой подсистемы нет. Новых пакетов
два: `internal/version` (копия хелперов версий из `internal/update`) и
`tools/architecture/check-identity`.

### Решения, принятые при подготовке документа

| Вопрос | Решение | Почему |
| --- | --- | --- |
| Как переименовывать литералы | Прямая замена на месте + проверка `check-identity` | Дифф читается как текст, форма кода не меняется. Проверка держит инвариант вместо пакета-реестра. |
| Имя бинарника при установке | Репозиторий и модуль называются `aghelm`, пакет `main` остаётся в корне | `go install` берёт имя из последнего элемента пути модуля, так что любой способ установки (`make`, `go install .`, `go install github.com/Doremi203/aghelm@latest`) даёт `aghelm`. |
| Лента сообщений | `internal/feed` удаляется целиком | Анонсов у форка нет. Пустая лента означала бы фоновый запрос без пользы или недостижимый код. |
| `install.sh`, `install.ps1` | Удаляются | Устанавливать нечего, пока нет релизов. При проектировании релизов их можно восстановить из истории git. |
| `.goreleaser.yaml` | Удаляется | Тот же довод, что для install-скриптов: файл никто не запускает и проверить нельзя, пока нет релизов. |
| Приветствие первого запуска | Остаётся, без проверки `IsRelease`: экран настройки открывается, когда в `state.db` нет `last_seen_version` | Новый пользователь aghelm, как и новый пользователь upstream, видит настройку при первом запуске. С проверкой `IsRelease` сборка из исходников не показала бы её никогда, а вручную экран находят не все. |

### Часть 1. Переименование

#### Путь модуля и имя бинарника

Репозиторий на GitHub переименован в `Doremi203/aghelm` (уже сделано, remote
`origin` обновлён). `go.mod` получает `module github.com/Doremi203/aghelm`.
Импорты меняются механически (`gofmt -r` не нужен, достаточно замены
префикса). Литерал `module` в `tools/architecture/check-ui-boundaries/main.go`
заменяется на вызов `go list -m`, чтобы следующее переименование его не
задело.

Пакет `main` остаётся в корне. Последний элемент пути модуля теперь `aghelm`,
поэтому `go install` ставит бинарник `aghelm`. В `Makefile` меняется только
`-o agent-manager` в цели `build`.

#### Поверхности изоляции

Каждая строка таблицы — место, где две копии могли бы встретиться. Новые
значения выбраны так, чтобы ни одно не совпадало со значением upstream.

| Поверхность | Где объявлено | Было | Стало |
| --- | --- | --- | --- |
| Бинарник | `go.mod` (последний элемент пути модуля), `Makefile` | `agent-manager` | `aghelm` |
| Каталог конфигурации и `state.db` | `internal/config/config.go` | `<UserConfigDir>/agent-manager` | `<UserConfigDir>/aghelm` |
| Сокет tmux | `internal/tmux/tmux.go:25`, `internal/remote/ssh.go:211` | `agentmgr` | `aghelm` |
| Удалённый бинарник по SSH | `internal/remote/connection.go:30` | `agent-manager` | `aghelm` |
| Префикс ошибок, который снимает удалённый клиент | `main.go`, `internal/remote/client.go:281` | `agent-manager: ` | `aghelm: ` |
| Имя MCP-сервера | `internal/mcpreg/mcpreg.go:27`, `internal/mcpserver/mcpserver.go:288` | `agent-manager` | `aghelm` |
| Переменные окружения | `internal/hooks`, `internal/config`, `internal/tmux/launch.go`, `internal/store/envprofiles.go` и другие | `AGENT_MANAGER_*` | `AGHELM_*` |
| Чужие переменные окружения | `main.go` | aghelm, запущенный из панели upstream, наследует `AGENT_MANAGER_*` и передаёт их серверу tmux и сессиям | при старте, до любого `exec` и запуска сервера tmux, aghelm удаляет из своего окружения все переменные с префиксом `AGENT_MANAGER_` |
| Файлы хуков и плагинов CLI | `internal/sessionreport/sessionreport.go` | `agent-manager-{pi,omp,command-code}.ts`, `agent-manager-{copilot,opencode}/`, манифест `agent-manager`, `agent-manager.track-conversation` | `aghelm-*`, манифест `aghelm`, `aghelm.track-conversation` |
| Нотификатор macOS | `internal/notify/mac_darwin.go` | `Agent Manager.app`, `dev.agent-manager.notifier`, `agent-manager-*.aiff`, `agent-manager.icns` | `aghelm.app`, `io.github.doremi203.aghelm.notifier`, `aghelm-*.aiff`, `aghelm.icns` |
| Уведомления Linux, WSL, OSC 777 | `internal/notify/notify.go`, `command.go`, `wsl.go` | `x-agent-manager.*`, заголовок `agent-manager` | `x-aghelm.*`, `aghelm` |
| Временные каталоги | `internal/clipboard/clipboard.go:63`, `internal/config/config.go` | `agent-manager-pastes`, `agent-manager-test-` | `aghelm-pastes`, `aghelm-test-` |
| Маркер курсора IME | `internal/ui/frame/ime.go`, `internal/ui/focus/render.go` | `agent-manager-ime-cursor` | `aghelm-ime-cursor` |
| Текст для агентов и пользователя | `internal/launch`, `internal/sessioncmd`, `internal/cli`, `internal/mcpserver`, `internal/store`, `internal/ui/*` | `agent-manager …`, `Agent Manager` | `aghelm …`, `aghelm` |

Bundle ID построен на домене `io.github.doremi203`, который принадлежит
владельцу форка, а не на `dev.aghelm`: домен `aghelm.dev` владельцу форка не
принадлежит.

Последствие для агентов: MCP-инструменты в Claude Code будут называться
`mcp__aghelm__*`, а не `mcp__agent-manager__*`. В репозитории нет текстов,
которые называют инструменты с префиксом `mcp__agent-manager__`;
`check-identity` не даст им появиться.

MCP-сервер upstream, зарегистрированный глобально, запускается и в сессиях
aghelm. Он ничего не делает, потому что в сессии нет
`AGENT_MANAGER_SESSION_ID`, и включается режим без вызывающего
(`internal/mcpserver/mcpserver_nocaller_test.go`). Изоляция MCP опирается на
это поведение upstream. Поэтому aghelm удаляет `AGENT_MANAGER_*` из своего
окружения при старте: иначе aghelm, запущенный из панели upstream, передал бы
своим сессиям ID чужой сессии (сервер tmux наследует окружение процесса,
`commandEnv` в `internal/tmux/platform_unix.go`). Обратный случай, upstream,
запущенный из панели aghelm, aghelm исправить не может: окружение передаёт
процесс upstream. Это принятое ограничение (Risks & Mitigations).

#### Проверка `tools/architecture/check-identity`

Новая программа, по образцу соседних `check-file-names` и `check-file-sizes`.
Один текстовый проход по всем файлам из `git ls-files`, кроме бинарных,
включая `.go` с комментариями и тестами. Запрещённые подстроки:
`agent-manager`, `agent_manager`, `AgentManager`, `Agent Manager`,
`agentmgr`, `AGENT_MANAGER_`, `wainmann`, `YoanWai`. Репозиторий и модуль
называются `aghelm`, поэтому законных вхождений `agent-manager` вне списка
исключений нет, в том числе в путях импорта. Разбирать Go-код не нужно.
Проверка вводится одним коммитом на последнем шаге, когда переименовано всё.

Исключения хранятся в программе как пары «путь + точная подстрока» (или
«каталог целиком»), а не как номера строк. Список может только сокращаться,
как `rootExtras` в `check-file-names`. Полный список:

*   `LICENSE`, `NOTICE` — целиком.
*   `README.md` — строка `aghelm — модифицированная версия
    [agent-manager](https://github.com/YoanWai/agent-manager).`
*   `tools/architecture/check-identity/` — целиком: в нём хранятся
    запрещённые строки и фикстуры теста.
*   `main.go` — подстрока `"AGENT_MANAGER_"` (удаление чужих переменных, см.
    Поверхности изоляции).
*   Записанные экраны CLI — вход для правил статуса; правка поменяет то, что
    проверяет тест: фрагменты в `internal/status/status_copilot_test.go`,
    `status_test.go`, `transcript_test.go` и каталог
    `internal/status/testdata/`.
*   `docs/designs/upstream-detach.md` — целиком: документ описывает, что было
    до переименования.

Инвариант проверяется в двух местах. `check-identity` падает на любой новой
строке с запрещённой подстрокой, в коде, комментарии или документе;
конкатенацию (`"agent-" + "manager"`) он не видит. Изоляцию во время работы подтверждает ручная проверка «Работа рядом».

### Часть 2. Удаление

```text
                 сейчас                                    после
 ┌──────────────────────────────┐            ┌──────────────────────────────┐
 │ main ─► update.IsRelease     │            │ main ─► version              │
 │ config/version ─► update     │            │ config/version ─► version    │
 │ feed ─► update.VersionWithin │   ──►      │ (feed удалён)                │
 │ ui/notices/release ─► update │            │ (release разобран, см. ниже) │
 │ cli/update ─► update         │            │ (cli update удалён)          │
 │ cli/report, mcp ─► report    │            │ (report удалён)              │
 └──────────────────────────────┘            └──────────────────────────────┘
```

#### `internal/version` (новый, копия)

Получает копию того, что нужно оставшемуся коду из `internal/update/update.go`:
`IsRelease`, `VersionWithin` и их приватные хелперы разбора (`parseVersion`,
`greater`). Тесты на эти функции копируются вместе с ними. `internal/update`
держит свои копии до собственного удаления, так что промежуточный коммит
собирается. Пакет не делает сетевых запросов и не читает файлов. Потребители
после удаления `update`: `main.go` (`resolveVersion`) и
`internal/config/version.go`. Последний обслуживает диалог `minimum_version`:
он предупреждает, что установленный CLI (Claude Code и другие) старее
`minimum_version` из определения инструмента, и предлагает обновить CLI.

#### `internal/update` и всё, что на нём стоит

Удаляются целиком:

*   `internal/update/` (апдейтер, каналы brew/scoop/AUR/mise/Nix, провенанс,
    `testdata`).
*   `internal/ui/notices/release/`, `internal/ui/notices/release_pane.go`,
    `release_tabs.go` и их тесты: вкладка «что нового» и Highlights / Thank you
    из релизов GitHub. Пакет `release` держит не только релизы. Через
    `release.State` также проходят строка версии в заголовке и Settings,
    наборы закрытых и просмотренных notices (`Dismissed`, `Seen`),
    приветствие первого запуска (`Greet` → `SetupGreeting`) и `RestartPath`.
    Всё, кроме `RestartPath`, переезжает: `Version` становится полем `Model`,
    `Dismissed`/`Seen` переходят в пакет `internal/ui/notices`, где уже есть
    `Panel`, а решение о приветствии — в `notices_adapter.go`. `RestartPath`
    удаляется вместе с самообновлением.
*   `restart_unix.go`, `restart_windows.go` и ветка перезапуска в
    `main.go` (`finished.RestartPath()`): процесс перезапускался
    только после того, как самообновление подменяло бинарник на диске.
*   **Приветствие первого запуска без проверки на релиз.** Сейчас
    `openStartupGreeting` показывает экран настройки только сборкам, для
    которых `IsRelease(version)` истинно. aghelm собирается только из
    исходников (`git describe` даёт `0.47.1-10-g…`), так что с этой проверкой
    он никогда не показал бы настройку. Проверка снимается. Экран настройки открывается, когда в `state.db` нет
    `last_seen_version`, и при показе записывает туда текущую версию.
    Приветствие «что нового» уходит вместе с вкладкой релизов, а ключи
    `whats_new_version` и `whats_new_from_version` больше никто не пишет.
    Свежая `state.db` их не содержит, так что миграция не нужна.
*   `internal/cli/update.go` и подкоманда `aghelm update`.
*   Строка версии в Settings перестаёт быть действием: `versionRow` в
    `internal/ui/settings/view.go` показывает только версию, поле
    `FieldUpdate`, состояние `Applying`/`Latest` и обработчик в
    `settings_adapter.go` уходят.
*   В `main.go` убираются `update.SetBuildSource` и переменная
    `buildSource` (`-X main.buildSource=Homebrew` выключал самообновление,
    других читателей у неё нет).
*   Переменная `AGENT_MANAGER_VERIFY_ATTESTATION` и `sigstore`-зависимости,
    `go mod tidy`.

#### `internal/feed`

Удаляются `internal/feed/`, `internal/ui/notices/feed_pane.go` и тест к нему,
`docs/messages.json`, связка в `notices_adapter.go`. Правило о `max_version`
из AGENTS.md (раздел Pitfalls) удаляется вместе с файлом. После этого в панели
сообщений остаются inbox и встроенные в бинарник notices.

#### `report_issue`

Удаляются `internal/report/`, `internal/cli/report.go` (подкоманды `issue` и
`feature`), регистрация инструмента в `internal/mcpserver/mcpserver.go`,
абзац о нём в тексте инструкций MCP, тесты
(`mcpserver_test.go`, `cli/report_test.go`, `cli_test.go`), раздел в
`docs/usage.md`. Строка справки в `Help` (`internal/cli/cli.go`) про `issue and feature`
правится.

#### Прочее, что указывает на upstream

*   `internal/ui/notices/notice.go` (`RepoURL`) переводится на
    `https://github.com/Doremi203/aghelm`. В `links.go` `DocsURL`
    (`agent-manager.dev/docs/`) ведёт на `RepoURL + "/tree/main/docs"`:
    отдельного сайта у форка нет. `BugReportURL` и `FeatureRequestURL`
    теряют параметр `template=…`, потому что шаблоны issue удаляются в
    части 3, и открывают пустой `issues/new` форка. Текст в
    `SupportRequestURL` называет aghelm.
*   `tools/badges/` удаляется вместе с workflow, который его запускает.

### Часть 3. CI → `make check`

Удаляются `.github/workflows/` (все шесть), `.github/dependabot.yml`,
`.github/ISSUE_TEMPLATE/`, `.github/pull_request_template.md`,
`.github/CODEOWNERS`, `.github/FUNDING.yml`, `.coderabbit.yaml`,
`codecov.yml`, `install.sh`, `install.ps1`, `.goreleaser.yaml`. Конфигурацию
goreleaser никто не запускает, пока нет релизного workflow, и проверить её
нельзя, как и install-скрипты (альтернатива I).

`Makefile` получает цели `check` и `vulncheck` (строка `check-identity`
добавляется на последнем шаге, вместе с самой проверкой):

```make
check:
	test -z "$$(gofmt -l .)"
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./...
	go run ./tools/architecture/check-ui-boundaries
	go run ./tools/architecture/check-file-names
	go run ./tools/architecture/check-file-sizes
	go run ./tools/architecture/check-identity
	mkdir -p /tmp/amtest
	env -u TMUX TMUX_TMPDIR=/tmp/amtest SHELL=/bin/sh go test -race ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...
```

Версия staticcheck берётся из `ci.yml:155` до его удаления.

govulncheck ходит в сеть за базой уязвимостей, поэтому вынесен в отдельную
цель `vulncheck` и в `check` не входит: его запускают вручную. Основной
сигнал об уязвимых зависимостях — Dependabot alerts, включённые в настройках
репозитория `Doremi203/aghelm`: они не требуют ни файла, ни workflow и к
upstream не обращаются. Цель `check`
вызывает команды sh (`test`, `mkdir -p`, `env -u`), а цель `build` использует
только функции make. Поэтому на Windows без sh `check` не работает (см. Risks
& Mitigations).

### Часть 4. Документация

*   Удаляются: `GOVERNANCE.md`, `MAINTAINERS.md`, `README.ja.md`,
    `README.zh-CN.md`, `.github/CODE_OF_CONDUCT.md`, знак upstream
    `docs/brand/*.svg`.
*   Переписываются кратко: `README.md` (что такое aghelm, откуда он, как
    собрать), `.github/SECURITY.md` (куда сообщать), `.github/CONTRIBUTING.md`
    (сохраняются заголовки «Checks» и «Capturing a TUI frame», на которые
    ссылается AGENTS.md), `docs/install.md` (только из исходников).
*   Правятся: `AGENTS.md` (Build and test, Releases, инварианты про pin-версии
    actions и `internal/update`, Pitfalls про `messages.json`, Layout),
    `docs/architecture.md` (раздел «Rules CI enforces» → `make check`, список
    пакетов), `docs/usage.md`, `docs/configuration.md`,
    `docs/notifications.md`, `docs/ssh-connections.md` (имена, переменные,
    сокет), `PRODUCT.md`, `DESIGN.md`, `REVIEW.md`.
*   `skills/agent-manager/` и `skills/agent-manager-reference/`
    переименовываются в `skills/aghelm/` и `skills/aghelm-reference/`,
    команды в них меняются.
*   `NOTICE` дополняется строкой о том, что aghelm — модифицированная версия
    agent-manager (Apache 2.0 §4b). `LICENSE` не меняется.
*   Python-скрипты `tools/e2e/` и `tools/compatibility/` (включая
    `two_manager.py`): имя бинарника, сокет, переменные окружения.
    `matrix.py` перестаёт скачивать релизы upstream.

### Порядок коммитов

Каждый коммит собирается и проходит `go test`.

```text
0. git remote remove upstream                 (локально, без коммита)
1. module path → github.com/Doremi203/aghelm   (только импорты + go.mod + check-ui-boundaries)
2. имя бинарника: Makefile, .gitignore
3. удалить report_issue (internal/report, cli/report, MCP-инструмент, текст)
4. internal/version: копия IsRelease/VersionWithin, потребители вне update и feed переключены
5. удалить internal/update, notices/release (перенос Version/Dismissed/Seen/Greet), cli update,
   version row action, restart_*.go, internal/feed, feed_pane, docs/messages.json;
   приветствие без IsRelease; go mod tidy
6. переименование идентичности (таблица выше), удаление AGENT_MANAGER_* при старте
7. AGENT_MANAGER_* → AGHELM_*
8. CI → make check, удаление .github/*, install-скриптов, .goreleaser.yaml, tools/badges
9. документация, skills, Python-скрипты; check-identity
```

Шаги 3–5 идут до переименования, чтобы не переименовывать код, который потом
удаляется. `report_issue` ни от чего не зависит и уходит первым. Лента
удаляется вместе с апдейтером, потому что её таймер и разбор живут в пакете
`release`, который удаляется на том же шаге; так `feed.go` не правится ради
промежуточного коммита. Шаг 7 отделён от шага 6: это замена одного префикса
примерно в 120 местах, её проще проверить отдельным диффом. `check-identity`
вводится на шаге 9, когда переименовано всё, и с этого коммита проходит.

## Alternatives Considered

Альтернативы из брифа, предшествовавшего этому документу, и из вопросов,
решённых при подготовке документа.

### A. Сохранить `internal/update` с заглушкой «обновления отключены»

*   **Pros**: код апдейтера, знания о каналах brew/scoop/AUR/Nix и провенанс
    остаются под рукой для собственного конвейера.
*   **Cons**: заглушка — ветка для состояния, которое в этой сборке не
    наступает (AGENTS.md: No speculative branches). Провенанс привязан к
    `.github/workflows/release.yml` (`provenance.go:42`), а workflows
    удаляются, так что проверка не пройдёт никогда. Заглушка должна обойти
    `VersionWithin` и `IsRelease`, которыми пользуются диалог
    `minimum_version` и `main.go`, и при ошибке ломает их молча.
*   **Reason for Rejection**: Rejected by author: апдейтер без рабочего
    провенанса — мёртвый код. Он удаляется, а собственный апдейтер берётся из
    истории git или пишется заново.

### B. Направить апдейтер на релизы Doremi203

*   **Pros**: самообновление работает сразу, как только появится первый релиз.
*   **Cons**: релизов форка нет. Без `release.yml` нет аттестации, и проверка
    провенанса не проходит. Схема тегов форка не решена, и `git describe`
    наследует теги upstream `v0.x`, так что сравнение версий ошибается.
*   **Reason for Rejection**: Rejected by author: без `release.yml`
    провенанс не проходит, а без схемы тегов сравнение версий ошибается. Оба
    вопроса относятся к релизному конвейеру (Non-Goals).

### C. Пакет `internal/identity` как единственный источник имён

*   **Pros**: все значения в одном файле. Следующее переименование меняет одну
    строку. Сокет перестаёт быть объявлен дважды. Тест «каждое значение
    отличается от upstream» проверяет один файл.
*   **Cons**: импорт в каждом пакете с такими литералами (они есть в 54
    нетестовых файлах), в том числе в `internal/remote/value` (литералы в
    `connection.go` и `trouble.go`), который `check-ui-boundaries` держит без
    зависимостей внутри модуля. Строки справки и словаря `vocabulary.go`
    превращаются в конкатенацию. Пакет не мешает написать новый литерал мимо
    него: без проверки он остаётся соглашением.
*   **Reason for Rejection**: Rejected by author: прямая замена плюс
    `check-identity` даёт тот же инвариант без изменения формы кода. Проверка
    падает на новом литерале с запрещённой подстрокой. Пакет такой литерал не
    останавливает.

### D. Гибрид: `internal/identity` только для значений, общих с ОС

*   **Pros**: около 15 значений, от которых зависит изоляция (сокет, каталог,
    MCP-имя, префикс переменных, bundle ID, файлы хуков), лежат в одном
    месте. Текст меняется прямой заменой.
*   **Cons**: два правила вместо одного. Граница «что относится к identity»
    размыта: имя бинарника в подсказке агенту — это текст или изоляция? Ошибка
    в этом вопросе и приводит к тому, что агент вызывает `agent-manager` по
    имени и находит бинарник upstream (The Problem, п. 2).
*   **Reason for Rejection**: Rejected by author: одно правило, которое
    проверяется машиной, проще двух, одно из которых решает человек.

### E. Оставить модуль `agent-manager`, только `Makefile -o aghelm`

*   **Pros**: репозиторий не переименовывается, ни один файл не переезжает.
*   **Cons**: `go install .` и `go install github.com/Doremi203/agent-manager@latest`
    берут имя из последнего элемента пути модуля и ставят `agent-manager`
    поверх upstream в `GOBIN` (The Problem, п. 4). Это закрывается только
    для тех, кто ставит через make.
*   **Reason for Rejection**: Rejected by author: изоляция не должна зависеть
    от того, каким способом установили бинарник.

### F. Оставить имя репозитория и перенести `main` в `cmd/aghelm`

*   **Pros**: адрес репозитория не меняется. `go install …/cmd/aghelm`
    даёт `aghelm` при модуле `github.com/Doremi203/agent-manager`.
*   **Cons**: пять файлов пакета `main` переезжают, git blame у них
    рвётся. Модуль и бинарник называются по-разному, поэтому `check-identity`
    должен разрешать `agent-manager` в составе `Doremi203/agent-manager`.
    Позднее переименование репозитория означало бы вторую
    полную перезапись импортов.
*   **Reason for Rejection**: Rejected by author: путь модуля всё равно
    переписывается на шаге 1, поэтому переименовать репозиторий сейчас
    дешевле всего. Цена — зависимость от редиректа GitHub со старого адреса
    (Risks & Mitigations).

### G. Направить ленту на `Doremi203` с пустым `messages.json`

*   **Pros**: код не меняется. Анонс можно опубликовать коммитом в `main`.
*   **Cons**: фоновый запрос к GitHub раз в 10 минут с каждой установки, а
    ответ пустой. `max_version` придётся сравнивать с версиями форка, схемы
    которых ещё нет.
*   **Reason for Rejection**: Rejected by author: запрос без пользы. Когда
    понадобятся анонсы, лента вернётся из истории git вместе со схемой версий.

### H. Оставить ленту с пустым URL

*   **Pros**: запросов нет, включение — одна строка.
*   **Cons**: та же заглушка, что в альтернативе A: код ленты остаётся
    недостижимым.
*   **Reason for Rejection**: Rejected by author: правило про спекулятивные
    ветки применяется одинаково к апдейтеру и к ленте.

### I. Оставить `install.sh` и `install.ps1` с сообщением «релизов нет»

*   **Pros**: адрес `curl …/install.sh | sh` готов к первому релизу.
*   **Cons**: до первого релиза скрипты ничего не делают. Проверить их на
    настоящем архиве нельзя, потому что архива нет. Ошибку в установщике
    обнаружит первый пользователь первого релиза. Homebrew эти скрипты не
    использует.
*   **Reason for Rejection**: Rejected by author: установка только из
    исходников, пока нет релизов.

### J. Скопировать `state.db` целиком

*   **Pros**: aghelm сразу видит группы, настройки, подключения и пресеты.
*   **Cons**: строки сессий несут `tmux_socket`, а строки с пустым сокетом
    берёт любой поллер (`internal/store/sessions.go:345`). Когда heartbeat
    upstream устареет, форк заберёт поллер и может пометить сессии мёртвыми,
    перезапустить их или возобновить тот же разговор агента во второй копии.
    Простое копирование файла в режиме WAL небезопасно.
*   **Reason for Rejection**: Rejected by author: скопированные строки сессий
    с пустым `tmux_socket` забирает поллер второй копии, а копировать файл в
    режиме WAL небезопасно.

### K. Перенести только настройки (`VACUUM INTO` или `aghelm import`)

*   **Pros**: настройки, группы и подключения переносятся без живых сессий.
*   **Cons**: нужна команда и список таблиц, который придётся сверять с каждой
    новой миграцией. Без `user_version` повторный импорт опасен: upstream
    может переиспользовать имя колонки с другим смыслом.
*   **Reason for Rejection**: Rejected by author: список таблиц нужно
    сверять с каждой миграцией, а без `user_version` повторный импорт
    небезопасен. Перенести настройки руками дешевле.

### L. Оставить префикс `AGENT_MANAGER_*` и задокументировать это

*   **Pros**: около 120 вхождений в 39 файлах не меняются.
*   **Cons**: `AGENT_MANAGER_CONFIG_DIR`, заданный для upstream, направит
    aghelm в чужую `state.db`. Сессия, запущенная в панели одной копии,
    передаёт свой `AGENT_MANAGER_SESSION_ID` процессам другой.
*   **Reason for Rejection**: Rejected by author: общий префикс ломает
    изоляцию, переменные переименовываются в `AGHELM_*`.

### M. Отложить переименование модуля и переписывание документации

*   **Pros**: меньший первый дифф, быстрее работающий бинарник.
*   **Cons**: проверка готовности требует, чтобы в репозитории не было
    `YoanWai`, а `check-identity` падает на документации upstream.
*   **Reason for Rejection**: Rejected by author: без них `check-identity`
    не проходит, а цель «код и документация принадлежат форку» не
    выполнена. Переименование модуля выделено в отдельный механический коммит
    (шаг 1 порядка).

### N. Оставить один workflow, который вызывает `make check`

*   **Pros**: проверка обязательна для каждого push и PR, а не по желанию.
    Workflow запускается на GitHub форка и к upstream не обращается.
*   **Cons**: возвращает зависимость от GitHub Actions и сопровождение
    pin-версий actions, которое AGENTS.md требует для каждого workflow.
    Раннер не покрывает всю матрицу ОС без дополнительных job. Форк личный,
    и проверку запускает тот же человек, который пишет коммит.
*   **Reason for Rejection**: Rejected by author: проверку запускает локально
    тот, кто коммитит; сопровождать CI ради одного разработчика дороже, чем
    запускать `make check`. Остаточный риск описан в Risks & Mitigations.

## Risks & Mitigations

| Риск | Влияние | Смягчение |
| --- | --- | --- |
| Разрешения на `mcp__agent-manager__*` в настройках Claude Code и других CLI не действуют для `mcp__aghelm__*` | Агент в сессии aghelm спрашивает подтверждение на каждый вызов | README говорит, что разрешения выдаются заново под именем `aghelm` |
| Без апдейтера исправления, в том числе исправления безопасности, приходят только при пересборке | Установленный бинарник устаревает незаметно | README: обновление — `git pull && make install` |
| Нет проверки уязвимостей в CI (govulncheck, CodeQL, Dependabot updates удаляются) | Уязвимая зависимость остаётся незамеченной | Dependabot alerts в настройках `Doremi203/aghelm`; `make vulncheck` для ручной проверки |
| upstream, запущенный из панели aghelm, передаёт `AGHELM_*` своим сессиям | Глобальный MCP-сервер `aghelm` в такой сессии действует от имени сессии aghelm | Принятое ограничение: upstream запускать из чистого shell |
| `make check` никто не запускает автоматически | Правило AGENTS.md нарушается в коммите | Принято, см. альтернативу N |
| Механическое переименование около 144 литералов и около 120 переменных | Пропущенное или лишнее вхождение меняет поведение | `check-identity` ловит пропуски в любом тексте; тесты меняются вместе с кодом; ручная проверка «Работа рядом» |
| Старый адрес `github.com/Doremi203/agent-manager` перенаправляет на `aghelm`, пока никто не создаст репозиторий с этим именем | Старые ссылки и `go install` по старому пути начинают вести в чужой репозиторий | В коде и документации нет старого адреса (`check-identity`); remote в локальных клонах обновляется на новый адрес |
| `make check` на Windows без sh не работает | Разработчик на Windows не может запустить проверку | Записано в CONTRIBUTING |

--------------------------------------------------------------------------------

## Non-Functional Requirements

*   **Безопасность.** Удаление апдейтера убирает единственный путь, которым
    бинарник заменял сам себя, вместе с проверкой checksum и провенанса.
    Инвариант AGENTS.md про `internal/update` удаляется вместе с пакетом.
    Новых секретов и сетевых путей нет. Автоматических проверок уязвимостей
    (govulncheck в CI, CodeQL, scorecard, Dependabot updates) больше нет.
    Остаются Dependabot alerts в настройках репозитория и `make vulncheck`,
    запускаемый вручную. Риск принят, см. Risks &
    Mitigations.
*   **Откат.** Каждый шаг — отдельный коммит, который собирается и проходит
    тесты. Откат — `git revert` нужного коммита. Данных для миграции нет:
    `state.db` aghelm новая, а `state.db` upstream не трогается.
*   **Обратная совместимость.** С upstream её нет и не должно быть: другой
    каталог конфигурации, сокет, переменные, имя MCP-сервера. aghelm на
    удалённом хосте по SSH требует aghelm на этом хосте.

--------------------------------------------------------------------------------

## Detailed Implementation Plan

Файлы перечислены по шагам из «Порядка коммитов». Пути от корня репозитория.
Файл, который меняется на нескольких шагах, указан на каждом из них.

### Шаг 0. Remote

*   `git remote remove upstream`. Локальное действие, файлов не меняет.
*   Репозиторий на GitHub уже переименован в `Doremi203/aghelm`, `origin`
    указывает на `https://github.com/Doremi203/aghelm.git`. В других клонах
    remote обновляется так же (`git remote set-url origin …`).

### Шаг 1. Путь модуля

#### `go.mod` (Modify)

*   `module github.com/YoanWai/agent-manager` → `module github.com/Doremi203/aghelm`.

#### Все `.go` файлы с импортом модуля (Modify)

*   Замена префикса `github.com/YoanWai/agent-manager/` →
    `github.com/Doremi203/aghelm/` только в строках импорта. После
    замены выполнить `gofmt -w .`: смена первой буквы префикса может
    переставить строки внутри блока импортов. Дифф содержит только строки
    импорта.

#### `tools/architecture/check-ui-boundaries/main.go` (Modify)

*   Константа `module = "github.com/YoanWai/agent-manager"` заменяется
    значением из `go list -m`, которое программа читает при старте. Так
    следующее переименование её не задевает.

### Шаг 2. Имя бинарника

#### `Makefile` (Modify)

*   `build`: `go build -o aghelm$(shell go env GOEXE) .`.
*   `install`: `go install .` не меняется: имя теперь берётся из пути модуля.
*   Флаг `-X=main.version=$(VERSION)` в `GOFLAGS` не меняется.

#### `.gitignore` (Modify)

*   `/agent-manager`, `/agent-manager.exe` → `/aghelm`, `/aghelm.exe`.

### Шаг 3. Удаление `report_issue`

#### `internal/report/` (Delete)

*   `report.go`, `report_test.go`.

#### `internal/cli/report.go`, `internal/cli/report_test.go` (Delete)

*   Подкоманды `issue` и `feature`.

#### `internal/cli/cli.go`, `internal/cli/cli_test.go` (Modify)

*   Убрать регистрацию `issue`/`feature`, упоминание в справке
    (`Help` в `cli.go`) и случаи в тестах `TestJSONIsPromisedOnlyWhereTheCommandTakesIt` и `TestCommandsAndHelpCoverEverySection`.

#### `internal/mcpserver/mcpserver.go` (Modify)

*   Убрать регистрацию инструмента `report_issue` в `newServerWithMailbox` и
    абзац «Bugs and ideas» в тексте инструкций (`instructionsTail`).

#### `internal/mcpserver/mcpserver_test.go` (Modify)

*   Убрать тесты инструмента (`TestReportIssue*`) и ожидание его в списке
    инструментов.

#### `docs/usage.md` (Modify)

*   Убрать раздел про `report_issue`, `issue` и `feature`.

#### `tools/architecture/check-ui-boundaries/boundaries.go` (Modify)

*   Убрать `/internal/report`, перегенерировать.

### Шаг 4. `internal/version`

#### `internal/version/version.go` (Create)

*   Копируются из `internal/update/update.go`: `IsRelease(v string) bool`,
    `VersionWithin(version, minimum, maximum string) bool` и приватные
    `parseVersion`, `greater`. Поведение то же. Пакет ничего не импортирует
    из модуля. `internal/update` сохраняет свои копии до удаления на шаге 5.

#### `internal/version/version_test.go` (Create)

*   Копируются тесты `IsRelease` и `VersionWithin` из
    `internal/update/update_test.go`.

#### `main.go` (Modify)

*   `update.IsRelease` → `version.IsRelease` в `resolveVersion`. Импорт
    `update` остаётся до шага 5 (`SetBuildSource`).

#### `internal/config/version.go` (Modify)

*   `update.VersionWithin` → `version.VersionWithin`. Импорт `update`
    удаляется.

#### `internal/ui/notices_adapter.go`, `internal/cli/update.go` (Modify)

*   `update.IsRelease` → `version.IsRelease` (`openStartupGreeting` в `notices_adapter.go`,
    `runUpdate` в `cli/update.go`).

#### `tools/architecture/check-ui-boundaries/boundaries.go` (Modify)

*   Перегенерируется через `-write`: `internal/version` появляется в
    графе `config`.

### Шаг 5. Удаление апдейтера, вкладки релизов и ленты

#### `internal/update/` (Delete)

*   Весь каталог, включая `testdata/`: `apply.go`, `apply_unix.go`,
    `apply_windows.go`, `channel.go`, `provenance.go`, `source.go`,
    `update.go` и тесты к ним.

#### `internal/cli/update.go`, `internal/cli/update_test.go` (Delete)

*   Подкоманда `update`.

#### `internal/cli/cli.go` (Modify)

*   Убрать регистрацию группы `update`. Строка справки в `Help` (`cli.go`)
    («update and settings need no caller…») теряет `update`.

#### `internal/cli/cli_test.go` (Modify)

*   Убрать случаи, где ожидается подкоманда `update` или импортируется
    `internal/update`.

#### `internal/ui/notices/release/` (Delete)

*   `release.go`, `release_test.go`.

#### `internal/ui/notices/release_pane.go`, `release_pane_test.go`, `release_tabs.go` (Delete)

*   Вкладка «что нового».

#### `internal/ui/notices/notice.go` (Modify)

*   Получает `Dismissed(st)`, `Seen(st)` и константы `DismissedSetting`,
    `SeenSetting` из `release.go`, без изменений поведения.

#### `internal/ui/notices/panel.go`, `list.go` (Modify)

*   Убрать ветки вкладки релизов: в `list.go` структуру `Update` с полями
    `Latest`, `Refreshing`, `Applying` и строку «N available» (`Panel.Active` в `list.go`),
    ветки ленты и `FeedPending`,
    в `panel.go` поле `Refreshing` и хвост с глифом обновления
    (`frameTail`). Тесты
    `panel_test.go`, `panel_glyphs_test.go`, `list_test.go`,
    `inbox_test.go` (Modify) теряют случаи с релизами.

#### `internal/ui/model.go` (Modify)

*   Поле `release uirelease.State` заменяется на `version string`.
    `uirelease.New` убирается. `NewPanel` получает
    `uinotices.Dismissed(st)`, `uinotices.Seen(st)`. Из `Init` уходят
    `m.release.Check()` и тикер. `m.release.UseCached` и `IndexRanges`
    удаляются.

#### `internal/ui/notices_adapter.go` (Modify)

*   `openStartupGreeting` больше не проверяет `IsRelease`. Решение о
    приветствии переезжает сюда: прочитать `last_seen_version`, если ключа
    нет — записать `m.version` и открыть настройку (`m.openSetup(true)`),
    иначе ничего. Удаляются `startupGreeting` в нынешнем виде,
    обработчики обновления (`Refresh`, `Apply`), маршрутизация
    `m.release.Route`, `releaseRoot` и `RestartPath()`. Константа
    `setupGreeting` становится локальной. Убираются `Sources()` ленты и
    `FeedPending`: таймер ленты живёт в `release.State`, который удаляется
    на этом шаге. В панель идут только встроенные notices и inbox.

#### `internal/ui/notices_adapter_test.go`, `notices_adapter_narrow_test.go`, `notices_test_helpers_test.go` (Modify)

*   Убрать случаи релизов, «что нового» и ленты (импорт `internal/feed`).
    Добавить два случая: пустая
    `state.db` на сборке `dev` открывает настройку и записывает
    `last_seen_version`; второй запуск не открывает ничего.

#### `internal/ui/frame_adapter.go` (Modify)

*   `displayVersion(m.release.Version)` → `displayVersion(m.version)`.
    Удаляется значок новой версии (`m.release.Latest`, `UpdateGlyph`) и
    ветка `m.release.FeedPending`. Тесты `frame_adapter_frame_test.go`,
    `frame_adapter_integration_test.go` (Modify) теряют случаи значка.

#### `internal/ui/settings_adapter.go` (Modify)

*   Убрать обработку `FieldUpdate`, поля `Latest`, `Applying`,
    `UpdateReady` в контексте вида и вызов `m.release.Apply()`.
    `BugReportURL(m.release.Version)` → `BugReportURL(m.version)`.

#### `internal/ui/settings/view.go`, `state.go`, `keys.go`, `click.go` (Modify)

*   `versionRow` выводит только `version <v>`, строка не фокусируется.
    `FieldUpdate` удаляется из списка полей, клавиш и кликов. Тесты
    `keys_test.go`, `click_test.go` (Modify) и
    `internal/ui/settings_adapter_test.go` (Modify) теряют случаи
    обновления.

#### `internal/ui/presentation/glyphs.go` (Modify)

*   `DownloadGlyph` удаляется: его читает только `versionRow` в
    `versionRow` (`settings/view.go`). `UpdateGlyph` остаётся, потому что его читает
    `kindGlyph` в `internal/ui/notices/inbox.go`, который не меняется. Читатели
    `UpdateGlyph` в `updateTag` (`frame_adapter.go`) и `frameTail` (`panel.go`) уходят вместе с
    значком версии.

#### Тесты, которые читают `m.release` (Modify)

*   `internal/ui/rail_adapter_effect_test.go`,
    `internal/ui/workspace_adapter_refresh_test.go`,
    `internal/ui/focus/model_test.go`,
    `internal/ui/pathcomplete/pathcomplete_test.go`: заменить
    `m.release.Version` на `m.version` или убрать установку состояния
    релиза.

#### `internal/ui/helpers_test.go` (Modify)

*   `prepareStoreTemplate` записывает в шаблонную `state.db`
    `last_seen_version`, чтобы `buildModel` не открывал настройку в
    тестах, которые её не проверяют.

#### `main.go` (Modify)

*   Удалить `update.SetBuildSource(buildSource)`, переменную `buildSource`
    с комментарием про Homebrew и ветку `finished.RestartPath()` с вызовом
    `restart`. Импорт `update` уходит.

#### `restart_unix.go`, `restart_windows.go` (Delete)

*   Функция `restart` больше не вызывается.

#### `go.mod`, `go.sum` (Modify)

*   `go mod tidy` убирает `sigstore` и его зависимости.

#### `tools/architecture/check-ui-boundaries/boundaries.go` (Modify)

*   Удалить `releaseBoundary` и строки других пакетов, где упоминаются
    `/internal/update`, `/internal/ui/notices/release` и `/internal/feed`.
    Перегенерировать через `-write`.

#### `internal/feed/` (Delete)

*   `feed.go`, `feed_test.go`.

#### `internal/ui/notices/feed_pane.go`, `feed_pane_test.go` (Delete)

*   Вкладка ленты.

#### `internal/ui/mouse_test.go` (Modify)

*   Убрать импорт `internal/feed` и случай, который его использует.

#### `docs/messages.json` (Delete)

*   Файл ленты.

### Шаг 6. Переименование идентичности

Во всех файлах ниже значения меняются по таблице «Поверхности изоляции» из
Technical Plan: `agent-manager` → `aghelm`, `Agent Manager` → `aghelm`,
`agentmgr` → `aghelm`, `agent_manager` → `aghelm`, `AgentManager` →
`Aghelm`, `dev.agent-manager.notifier` →
`io.github.doremi203.aghelm.notifier`, ссылки `wainmann/agent-manager` и
`YoanWai/agent-manager` → `Doremi203/aghelm`. Тестовые файлы меняются
вместе со своими источниками, так что ожидания в тестах совпадают с кодом.
Комментарии, которые называют программу, правятся тоже: `check-identity` на
шаге 9 проверяет весь текст, включая комментарии.

#### Сокет, каталог, бинарник, MCP (изоляция)

*   `internal/tmux/tmux.go` (`defaultSocket`), `internal/tmux/bindings.go`,
    `internal/tmux/launch.go` (`relaunchHint`), `internal/tmux/platform_windows.go`;
    тесты `bindings_test.go`, `launch_test.go`, `platform_windows_test.go`,
    `tmux_unix_test.go`.
*   `internal/remote/ssh.go` (`-L agentmgr`), `connection.go` (удалённый
    бинарник по умолчанию), `client.go` (префикс `aghelm: ` и тексты
    ошибок), `internal/remote/value/connection.go`,
    `internal/remote/remotetest/remotetest.go`; тесты `client_test.go`,
    `operations_test.go`, `sanitize_test.go`, `ssh_test.go`,
    `ssh_unix_test.go`, `ssh_other_test.go`, `value/connection_test.go`.
*   `internal/config/config.go` (имя каталога, префикс временного
    каталога), `internal/config/builtin.go` (текст), `internal/config/version.go`
    (текст `OutdatedToolError`); тест `config_test.go`.
*   `internal/mcpreg/mcpreg.go` (`serverName`, тексты ошибок); тесты
    `mcpreg_test.go`, `mcpreg_copilot_test.go`, `mcpreg_hermes_test.go`,
    `mcpreg_kiro_test.go`, `mcpreg_timeout_test.go`.
*   `internal/mcpserver/mcpserver.go` (`Implementation.Name`, текст
    инструкций, описания инструментов), `delete.go`, `remote.go`; тесты
    `mcpserver_test.go`, `mcpserver_nocaller_test.go`, `remote_test.go`.
*   `internal/sessionreport/sessionreport.go` (имена файлов хуков,
    манифест, `aghelm.track-conversation`); тест `sessionreport_unix_test.go`.
*   `internal/clipboard/clipboard.go` (`aghelm-pastes`); тест
    `native_live_darwin_test.go`.
*   `internal/notify/mac_darwin.go` (бандл, bundle ID, звуки, иконка,
    `Info.plist`, `AgentManagerNotifierDelegate` →
    `AghelmNotifierDelegate`), `notify.go` (категории Linux, заголовки, OSC 777),
    `command.go`, `wsl.go`; тесты `mac_darwin_test.go`, `notify_test.go`,
    `command_test.go`, `wsl_test.go`.
*   `internal/ui/frame/ime.go`, `internal/ui/focus/render.go` (маркер IME);
    тест `internal/ui/frame/header_test.go`.
*   `internal/catalog/rpc.go` (`clientInfo` `agent_manager` → `aghelm`: имя
    уходит дочерним CLI).
*   `main.go` (префикс ошибок, `--version`, usage); тесты
    `main_test.go`, `main_unix_test.go`.
*   `main.go`: удаление переменных `AGENT_MANAGER_*` из окружения процесса
    до разбора подкоманд, до любого `exec` и запуска сервера tmux. Тест в
    `main_unix_test.go`: процесс, запущенный с `AGENT_MANAGER_SESSION_ID`,
    не передаёт её дочернему процессу. Литерал префикса стоит в списке
    исключений `check-identity`.

#### Текст для агентов и пользователя

*   `internal/launch/launch.go` (заметки о координации, предупреждение);
    тест `launch_test.go`.
*   `internal/sessioncmd/`: `vocabulary.go`, `conversation.go`, `format.go`,
    `group.go`, `lifecycle_placement.go`, `mailbox.go`, `messages.go`,
    `session.go`, `session_lifecycle.go`, `spawn.go`, `telemetry.go`,
    `terminal.go`, `wait.go`; тесты `backend_test.go`, `format_test.go`,
    `session_lifecycle_test.go`, `session_test.go`,
    `terminal_nocaller_test.go`, `terminal_test.go`, `vocabulary_test.go`.
*   `internal/cli/`: `cli.go`, `files.go`, `session.go`, `settings.go`; тесты
    `cli_test.go`, `env_test.go`, `files_test.go`, `parity_test.go`,
    `review_test.go`, `session_test.go`, `settings_test.go`,
    `terminal_test.go`.
*   `internal/app/prompthead.go`, `internal/app/serve.go`.
*   `internal/execution/poller.go`, `polldelivery.go`; тесты
    `poller_test.go`, `polldelivery_test.go`.
*   `internal/store/sessions_creation.go`, `envprofiles.go`; тесты
    `connections_test.go`, `envprofiles_test.go`, `inbox_claim_test.go`,
    `inbox_test.go`, `sessions_test.go`, `transfer_test.go`.
*   `internal/pwsh/pwsh.go`.
*   `internal/hooks/hooks.go` (только комментарии; переменные — шаг 7).
*   `internal/ui/`: `onboarding_adapter_preview.go`; тесты
    `connections_adapter_test.go`, `dialogs_adapter_rename_test.go`,
    `effects_adapter_registry_test.go`, `launchhint_adapter_test.go`,
    `notices_adapter_test.go`, `quick_adapter_test.go`,
    `rail_adapter_view_test.go`, `session_adapter_attach_test.go`,
    `session_adapter_lifecycle_test.go`, `settings_adapter_test.go`,
    `spawn_adapter_form_test.go`, `statusbar_adapter_test.go`,
    `workspace_adapter_refresh_test.go`.
*   `internal/ui/connections/dialog.go`, `rows.go`; тесты `dialog_test.go`,
    `rows_test.go`.
*   `internal/ui/help/catalog.go`; тест `content_test.go`.
*   `internal/ui/launchhint/install/copy.go`, `start.go`.
*   `internal/ui/notices/notice.go` (`RepoURL` →
    `https://github.com/Doremi203/aghelm`), `links.go` (`DocsURL` →
    `RepoURL + "/tree/main/docs"`, у `BugReportURL` и `FeatureRequestURL`
    убирается `template=…`, текст `SupportRequestURL`); тесты
    `links_test.go`, `list_test.go`, `text_test.go`.
*   `internal/ui/onboarding/importer.go`, `view.go`; тест `view_test.go`.
*   `internal/ui/review/comments.go`, `internal/ui/session/editor_resolve.go`,
    `internal/ui/settings/prompts_view.go`.
*   Тесты, где литерал — вход, а не вывод кода:
    `TestTruncateTailRuneSafe` в `internal/ui/presentation/text_test.go` (путь
    `…/agent-manager` → `…/aghelm`, ожидаемая ширина пересчитывается),
    `TestExactExpandsTabs` в `internal/ui/preview/pane_test.go` (строка вывода `go test` →
    `github.com/Doremi203/aghelm/…`),
    `TestLinkPagePrintsTheURLAndCopiesIt` в `internal/ui/terminal/link_page_test.go` (URL issue форка без
    `template`), `TestStartReapedReturnsTheStartError` в `internal/ui/terminal/reap_unix_test.go`
    (`/nonexistent/aghelm-test-binary`).
*   Записанные экраны CLI не меняются, их покрывает список исключений
    `check-identity`: фрагменты в `internal/status/status_copilot_test.go`,
    `status_test.go`, `transcript_test.go` и `internal/status/testdata/`.
*   `internal/deps/deps_test.go` меняется на шаге 8.

### Шаг 7. `AGENT_MANAGER_*` → `AGHELM_*`

#### Go (Modify)

*   `internal/hooks/hooks.go` (`EnvStatusFile`, `EnvSessionID`,
    `EnvLaunch`), `internal/config/config.go` (`CONFIG_DIR`, `EDITOR`,
    `STARTUP_*`), `internal/tmux/launch.go` (`AGENT_PID`, `ENV_KEYS`),
    `internal/store/envprofiles.go`, `internal/mcpserver/mcpserver.go`,
    `internal/ui/session/editor.go`, `editor_resolve.go`.
*   Тесты: `main_test.go`, `main_unix_test.go`
    (`MAIN_TEST`, `HOLD_RUNNING`, `CALLER_OUT`),
    `internal/config/config_test.go`, `internal/cli/env_test.go`,
    `internal/deps/deps_test.go` (`INSTALL_DEPS`),
    `internal/mcpreg/mcpreg_test.go`, `mcpreg_hermes_test.go`,
    `mcpreg_kiro_test.go`, `internal/mcpserver/mcpserver_test.go`,
    `internal/sessioncmd/session_test.go`, `terminal_test.go`,
    `internal/store/envprofiles_test.go`, `internal/tmux/launch_test.go`,
    `platform_windows_test.go`, `tmux_unix_test.go`,
    `internal/ui/session_adapter_attach_test.go`,
    `session_adapter_lifecycle_test.go`.

#### Python (Modify)

*   `tools/compatibility/matrix.py`, `mcp_probe.py`, `tools/e2e/smoke.py`,
    `tools/e2e/test_smoke.py`: имена переменных.

### Шаг 8. CI → `make check`

#### Настройки репозитория (не файл)

*   Включить Dependabot alerts для `Doremi203/aghelm` (Settings → Code
    security). Файл `.github/dependabot.yml` при этом удаляется: он
    управляет PR с обновлениями, а не оповещениями.

#### Delete

*   `.github/workflows/badges.yml`, `ci.yml`, `codeql.yml`, `release.yml`,
    `scheduled-security.yml`, `scorecard.yml`.
*   `.github/dependabot.yml`, `.github/CODEOWNERS`, `.github/FUNDING.yml`,
    `.github/pull_request_template.md`,
    `.github/ISSUE_TEMPLATE/bug_report.yml`, `config.yml`,
    `feature_request.yml`.
*   `.coderabbit.yaml`: конфиг бота ревью upstream.
*   `codecov.yml`: загрузки покрытия больше нет.
*   `install.sh`, `install.ps1`.
*   `.goreleaser.yaml`: конфигурацию никто не запускает, пока нет
    релизного workflow.
*   `tools/badges/` (`main.go`, `main_test.go`, `contributors.go`,
    `contributors_test.go`).

#### `internal/deps/deps_test.go` (Modify)

*   Удалить тесты, которые читают `install.sh` или `install.ps1` из корня
    репозитория, и их хелперы: `installFuncs`, чтение `install.ps1`,
    `repoFile` (все её вызовы читают эти два скрипта). Тесты:
    `TestHaveTTYWithoutATerminalDoesNotAbort`,
    `TestConfirmHonorsTheDepsFlag`, `TestConfirmDeclinesWithoutATerminal`,
    `TestDepInstallCmdOnDarwin`, `TestDepInstallCmdMapsDesktopPackages`,
    `TestDesktopDepsFollowTheSession`, `TestInWSLDoesNotAbort`,
    `TestVerifyAttestationMatchesPowerShellCase`,
    `TestPowerShellAttestationFallsBackToTheOldSigner`,
    `TestDetectOSOnWindowsNamesTheReadableInstaller`. Они проверяют
    функции shell-скриптов, а не пакет `deps`. Остальные тесты файла (подсказки
    по установке CLI, `claude.ai/install.sh` и т. п.) остаются.

#### `Makefile` (Modify)

*   Цели `check` и `vulncheck` по Technical Plan, `check` пока без
    `check-identity` (она появляется на шаге 9). `.PHONY` дополняется
    `check` и `vulncheck`. Версия staticcheck `2026.2.2` берётся из
    `ci.yml:155` до его удаления, govulncheck — `v1.10.0`.

### Шаг 9. Документация

#### Delete

*   `GOVERNANCE.md`, `MAINTAINERS.md`, `README.ja.md`, `README.zh-CN.md`,
    `.github/CODE_OF_CONDUCT.md`.
*   `docs/brand/mark.svg`, `wordmark-dark.svg`, `wordmark-light.svg`:
    знак upstream. Их использует только `README.md`, который
    переписывается без них.

#### Rewrite

*   `README.md`: что такое aghelm, фраза об атрибуции из списка исключений
    `check-identity`, сборка `make install` /
    `go install github.com/Doremi203/aghelm@latest`, обновление
    `git pull && make install`, заметка о том, что разрешения на
    MCP-инструменты в CLI выдаются заново под именем `aghelm`, ссылки на
    `docs/`. Картинки из `docs/` (`demo*.gif`, `screenshot-*.png`) остаются:
    интерфейс не меняется.
*   `.github/SECURITY.md`: сообщать владельцу форка через GitHub Security
    Advisories репозитория `Doremi203/aghelm`.
*   `.github/CONTRIBUTING.md`: убрать раздел про PR и governance. Сохранить
    заголовки «Checks» и «Capturing a TUI frame». `make build` пишет
    `./aghelm`. Описать `make check`, `make vulncheck` и что на Windows
    `check` требует sh. Убрать абзац про
    `buildSource`/обновления.
*   `docs/install.md`: только сборка из исходников. Минимальная версия
    tmux 3.1 остаётся.

#### Modify

*   `AGENTS.md`: Build and test (`go run .`, `make check`,
    `AGHELM_CONFIG_DIR`), удалить раздел Releases, удалить инварианты про
    `internal/update` и pin-версии actions, удалить Pitfalls про
    `docs/messages.json` и длину bullet в релизных заметках, Layout
    (без `update`/`feed`/`report`), ссылки на удалённые
    файлы (`REVIEW.md` остаётся).
*   `docs/architecture.md`: раздел «Rules CI enforces» → «Rules `make check`
    enforces», добавить `check-identity`. Убрать из списка пакетов
    `update`, `feed`, `report`, `notices/release`.
*   `docs/usage.md`, `docs/configuration.md`, `docs/notifications.md`,
    `docs/ssh-connections.md`: имя бинарника, `AGHELM_*`, сокет `aghelm`,
    bundle ID, удалённый бинарник по SSH. Из `usage.md` уходят разделы про
    обновление и вкладку «что нового».
*   `PRODUCT.md`, `DESIGN.md`, `REVIEW.md`: имя продукта и ссылки на
    upstream.
*   `NOTICE`: строка о том, что aghelm — модифицированная версия
    agent-manager, с указанием изменений в общих словах (Apache 2.0 §4b).
*   `skills/agent-manager/SKILL.md` → `skills/aghelm/SKILL.md`,
    `skills/agent-manager-reference/SKILL.md` →
    `skills/aghelm-reference/SKILL.md` (Move + Modify): имя skill, команды.

#### Python и README инструментов (Modify)

*   `tools/e2e/scenarios.py`, `smoke.py`, `ssh_scenarios.py`,
    `test_scenarios.py`, `test_smoke.py`, `tools/e2e/README.md`,
    `tools/compatibility/matrix.py`, `matrix.sh`, `mcp_probe.py`,
    `test_cleanup.py`, `two_manager.py`, `two_manager.sh`,
    `tools/compatibility/README.md`: имя бинарника `aghelm`, сокет, каталог
    `aghelm` в `profile_dir`.
*   `tools/compatibility/matrix.py`: удалить `released_binary` и флаг
    `--release`, который скачивал архивы с релизов upstream;
    `--peer-binary` становится обязательным. В `tools/compatibility/README.md`
    пример `--release v0.38.0` заменяется на `--peer-binary`.
    Экран настройки теперь открывается на чистом `HOME`, а сценарии его не
    ждут. `start_manager` в `scenarios.py` запускает менеджер, только когда
    `state.db` уже содержит `last_seen_version`. Для этого он вызывает
    `aghelm settings list` до старта TUI (`openSettingsStore` в `internal/cli/settings.go`
    открывает `state.db` через `store.Open`, который её создаёт), и пишет
    ключ через `sqlite3`, как это уже делает `seed_store`. Отдельный случай
    в `smoke.py` проверяет, что без ключа появляется экран настройки.

#### `tools/architecture/check-identity/main.go` (Create)

*   Один текстовый проход из Technical Plan: все файлы из `git ls-files`,
    кроме бинарных, включая `.go`. Запрещённые подстроки и исключения —
    списки из Technical Plan. Печатает `file:line: строка` и выходит с кодом
    1, если что-то нашлось. Вводится последним, после переписывания
    документации, поэтому коммит проходит `make check`.

#### `tools/architecture/check-identity/main_test.go` (Create)

*   Случаи: строка с `agent-manager`, `agent_manager` и `AgentManager`
    падает, в том числе в комментарии Go; путь
    `github.com/Doremi203/aghelm/internal/x` проходит; подстрока из списка
    исключений в своём файле проходит, а в чужом падает; прогон по всему
    репозиторию проходит.

#### `Makefile` (Modify)

*   В цель `check` добавляется `go run ./tools/architecture/check-identity`.

## Verification Plan

*   **Сборка:** `go build ./...`, `make build` даёт `./aghelm`,
    `go install .` ставит `aghelm` в `GOBIN`.
*   **Статика:** `gofmt -l .` пуст, `go vet ./...`,
    `go run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./...`,
    три архитектурные проверки и `check-identity`. Всё это вызывает
    `make check`. `check-identity` появляется на шаге 9 и проходит с него.
*   **Тесты:** `mkdir -p /tmp/amtest && env -u TMUX TMUX_TMPDIR=/tmp/amtest SHELL=/bin/sh go test -race ./...`,
    после каждого из шагов 1–9.
*   **Репозиторий:** `check-identity` проходит. Совпадения с запрещёнными
    подстроками остаются только в местах из списка исключений Technical
    Plan.
*   **Бинарник:**
    `strings aghelm | grep -iE 'wainmann|yoanwai|agentmgr|agent-manager\.dev|dev\.agent-manager|AGENT_MANAGER_|api\.github\.com/repos/|sigstore'`
    ничего не находит. `go.mod` не содержит `sigstore`.
*   **Работа рядом (вручную, macOS и Linux):** установить upstream
    `agent-manager` и `aghelm` под одним временным `HOME`
    (`AGHELM_CONFIG_DIR` не задан, чтобы проверить путь по умолчанию). В
    каждом открыть сессию Claude Code и сессию Gemini. Проверить:
    `tmux -L agentmgr ls` и `tmux -L aghelm ls` показывают по одной
    сессии; в глобальном конфиге Gemini две MCP-записи `agent-manager` и
    `aghelm`; файлы хуков `agent-manager-*` и `aghelm-*` лежат рядом;
    `aghelm sessions` из сессии aghelm видит только свою сессию; в
    окружении панели aghelm есть `AGHELM_SESSION_ID` и нет
    `AGENT_MANAGER_SESSION_ID`; на macOS клик по уведомлению aghelm
    открывает aghelm. Затем запустить `aghelm` из панели upstream: в
    окружении сессии aghelm нет ни одной `AGENT_MANAGER_*`, а вызов
    `mcp__agent-manager__list_sessions` из сессии aghelm в Gemini
    отклонён.
*   **Первый запуск:** `aghelm` с `AGHELM_CONFIG_DIR`, указывающим на пустой каталог, открывает
    экран настройки. Второй запуск не открывает. Кадры читаются по
    «Capturing a TUI frame» из CONTRIBUTING.
*   **Сеть:** на свежем запуске наблюдать соединения процесса (`lsof -i -p`
    или `nettop` на macOS, `ss -p` на Linux), пока пользователь открывает
    панель сообщений, Settings и справку и создаёт сессию. Соединений с
    внешними хостами нет; loopback исключается.
*   **SSH:** подключение aghelm к хосту с установленным `aghelm` работает
    и создаёт на нём сокет `aghelm`. Подключение к хосту, где есть только
    `agent-manager`, показывает «aghelm is not on the login shell's PATH».
*   **e2e:** `tools/e2e/smoke.py` и `tools/compatibility/two_manager.py`
    проходят с бинарником `./aghelm`.

--------------------------------------------------------------------------------

*Drafted with EGM*
