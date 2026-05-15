# Памятка фронту: переход на подписочную модель с мульти-регионом

Бэк-ветка: `corevpn-dev` (объединяет два больших изменения — мультисерверная подписка + удаление налички). После мержа бэка фронт должен обновиться синхронно — старая модель `subscription.server_id` / paid region switch / cash-оплата отвалятся.

## Что вообще удаляется (TL;DR)

1. **Платная смена региона** — концепции больше нет, юзер получает все серверы тарифа сразу.
2. **Оплата наличкой** — провайдер `cash` удалён вместе со всеми связанными ручками.
3. **Промокоды типа `cash_plan` и `region_switch`** — удалены вместе с соответствующей логикой.

Подробности по каждому пункту ниже.

## Что меняется в модели

**Было:** 1 подписка = 1 сервер; смена региона — платная транзакция (`POST /api/subscription/switch-region`), счётчик бесплатных смен в `users.free_region_switches`.

**Стало:** 1 подписка = N серверов из тарифа; юзер получает **subscription URL** (`/sub/<token>`) — её добавляют в любой VPN-клиент (v2rayNG/Hiddify/Streisand/Shadowrocket), клиент тянет список всех доступных серверов автоматически. Никаких смен.

## Изменения в API

### `GET /api/user/me`

- ❌ **уходит:** `free_region_switches`
- ✅ всё остальное без изменений (включая `is_admin`)

### `GET /api/subscription/status`

Старый shape:
```json
{
  "active": true,
  "subscription": {
    "server_id": "...",
    "connection_key": "vless://...",
    "xui_email": "..."
  },
  "days_remaining": 25,
  "traffic_gb": { "used": 12, "limit": 100, "remaining": 88 }
}
```

Новый shape (реальный, проверено по коду):
```json
{
  "active": true,
  "subscription_url": "https://api.zaruchevskiy.ru/sub/abc123def",
  "days_remaining": 25,
  "traffic_gb": { "used": 12, "limit": 100, "remaining": 88 },
  "subscription": {
    "id": "uuid",
    "user_id": 12345,
    "plan_id": "uuid",
    "status": "active",
    "sub_token": "abc123def",
    "started_at": "2026-05-01T10:00:00Z",
    "expires_at": "2026-05-31T10:00:00Z",
    "traffic_limit": 107374182400,
    "traffic_used": 12884901888,
    "max_devices": 3,
    "created_at": "2026-05-01T10:00:00Z",
    "servers": [
      {
        "id": "uuid (subscription_client id, НЕ сервер)",
        "subscription_id": "uuid",
        "server_id": "uuid (вот это id сервера)",
        "xui_client_id": "uuid",
        "xui_email": "u12345_s1a2b3c4_1700000000",
        "connection_key": "vless://...",
        "traffic_used": 5000000000,
        "enabled": true,
        "created_at": "2026-05-01T10:00:00Z",
        "server": {
          "id": "uuid",
          "name": "Germany 1",
          "country": "DE",
          "city": "Frankfurt",
          "flag_emoji": "🇩🇪",
          "server_address": "62.60.x.x",
          "server_port": 14549,
          "is_active": true,
          "sort_order": 0,
          "capacity": 100,
          "current_load": 23,
          "ping_ms": 45,
          "status": "online",
          "last_check_at": "2026-05-15T18:55:00Z",
          "created_at": "...",
          "updated_at": "..."
        }
      }
    ]
  }
}
```

Важно:
- `subscription_url` — **топ-уровень**, не внутри `subscription`.
- `subscription.servers[]` — массив `SubscriptionClient`, у каждого `id` это id клиента (не сервера). Реальный id сервера — в `server_id` и внутри `server.id`.
- `connection_key` живёт **на уровне SubscriptionClient** (`servers[i].connection_key`), а не внутри `server` — это VLESS URI для конкретного юзера на этом сервере.
- Поля `subscription.traffic_limit/traffic_used` — в **байтах**, не GB. Отдельно есть `traffic_gb` объект на топ-уровне в GB.
- `sub_token` присутствует в `subscription` — это секрет, не показывать в UI открыто.
- Поля верхнего уровня `subscription.server_id` / `connection_key` / `xui_email` / `xui_client_id` — **удалены**, всё переехало в `servers[]`.

### `GET /api/plans`

К каждому плану добавляется `servers[]` (массив **полных** `Server` объектов — те же поля что и в `subscription.servers[].server`):
```json
{
  "id": "uuid",
  "name": "Standard",
  "description": "...",
  "duration_days": 30,
  "traffic_gb": 100,
  "max_devices": 3,
  "price_ton": 5,
  "price_stars": 500,
  "price_usd": 4.99,
  "is_active": true,
  "sort_order": 0,
  "created_at": "...",
  "servers": [
    {
      "id": "uuid",
      "name": "Germany 1",
      "country": "DE",
      "city": "Frankfurt",
      "flag_emoji": "🇩🇪",
      "server_address": "...",
      "server_port": 443,
      "is_active": true,
      "ping_ms": 45,
      "status": "online",
      "capacity": 100,
      "current_load": 23,
      "...": "и т.д."
    }
  ]
}
```

Важно:
- Поле `traffic_gb`, не `traffic_limit_gb` (моя ошибка ранее).
- Поле флага сервера — **`flag_emoji`**, не `flag`.
- Юзеру в карточке плана показываем список флажков-регионов с тултипами «Germany — Frankfurt».

### `POST /api/subscription/switch-region`

❌ **Удалён.** Запросы возвращают `410 Gone` или `404`. Нужно убрать хук `useSwitchRegion` (или как он называется), карточку «сменить регион» в `RegionSection`, расчёт стоимости.

### Новый endpoint: `GET /sub/<token>`

**Фронт сам его НЕ дёргает.** Используется только для отображения в UI:
- QR-код, в котором лежит этот URL
- Кнопка «скопировать»
- Кнопка «поделиться» (Telegram WebApp share)

VPN-клиент (v2rayNG/Hiddify) сам этот URL ходит и тянет конфиги.

### Admin endpoints

`POST /api/admin/plans` body:
```json
{
  "name": "...",
  "description": "...",
  "duration_days": 30,
  "traffic_gb": 100,
  "max_devices": 3,
  "price_ton": 5,
  "price_stars": 500,
  "price_usd": 4.99,
  "sort_order": 0,
  "server_ids": ["uuid1", "uuid2", "uuid3"]
}
```

`PUT /api/admin/plans/:plan_id` (метод PUT, параметр `:plan_id`) — те же поля все опциональны через указатели, `server_ids: []` можно менять (nil = не трогать, `[]` = очистить).

`GET /api/admin/users/:user_id` — возвращает `UserWithSubscription`. **Важно:** subscription там сейчас отдаётся «голой», без `servers[]` и без `subscription_url`. Если для саппорта нужен URL — используй ротацию (она его возвращает) или мы добавим отдельный endpoint позже.

`POST /api/admin/users/:user_id/subscription/rotate-token` — выпустить новый sub_token (старый URL сразу умирает). Ответ: `{"success": true, "subscription_url": "https://.../sub/<new_token>"}`. Подойдёт для саппорта если URL утёк или нужно отозвать доступ.

## Изменения в UI

### `MainSection` (главная)

- ❌ убрать «Подключиться к серверу X» / «Сменить регион»
- ✅ добавить карточку **Subscription URL** как основной CTA:
  - QR-код (большой, по центру)
  - URL текстом с кнопкой «Скопировать»
  - Кнопка «Поделиться» (`Telegram.WebApp.openTelegramLink` или native share)
  - Под URL — компактная строка флажков `🇩🇪 🇳🇱 🇺🇸` доступных регионов
- ✅ блок «Как подключиться» с deeplink'ами на инструкции (v2rayNG / Hiddify / Streisand / Shadowrocket)

`handleShowKey` ранее открывал `success` view с одним connection_key — теперь open subscription card с URL.

### `RegionSection` (юзерская «Регионы», не путать с `admin/servers`)

**Радикальный пересмотр:**
- ❌ убрать «выбрать регион для смены»
- ✅ показать список всех серверов из подписки:
  - Флажок, имя, город, ping, status indicator (online/offline)
  - На каждом сервере — opt-in кнопка «Скопировать ключ для этого сервера» (на случай если юзер хочет отдельный VLESS URI, а не subscription URL)
- Опционально переименовать «Регионы» → «Серверы» / «Доступ»
- Никаких платных действий и счётчиков

### `SubscriptionSection`

- ✅ **Subscription URL — главный элемент:** большой QR, URL текстом + копи-кнопка
- ✅ Expandable панель «Инструкции по подключению» с шагами для популярных клиентов
- ✅ График использования трафика остаётся
- ✅ Список доступных регионов — chip'ы
- ✅ Кнопка «Продлить» — покупает тот же план

### `WalletSection`

- ❌ убрать упоминания «смены региона за TON»
- ❌ убрать «бесплатные смены остались: N»
- ✅ всё остальное (topup, история) без изменений

### `SettingsSection`

Никаких изменений (админ-кнопка уже работает).

### `PlansSection` (юзерская)

- ✅ к каждому плану показать `servers[]` в виде флажков с тултипами
- Опционально фильтр «нужен регион X» — показывать только подходящие планы

### `admin/plans.tsx`

- ✅ форма создания/редактирования плана — добавить **multi-select списка серверов**
- Группировать по регионам в UI (header «DE / NL / US» с серверами под ним)
- Сохранять `server_ids[]`
- Показывать счётчик «3 сервера в 2 регионах» в карточке плана

### `admin/users.tsx`

- ✅ карточка юзера — показать subscription URL (с возможностью «Reissue token»)
- Удобно: ссылка-копирование для саппорта

### `admin/servers.tsx`

- При создании нового сервера хорошо бы предложить «добавить в планы X, Y, Z?» (опционально, не блокирует MVP)

### `admin/logs / cash / bans / promo / stats`

Без изменений в этой итерации.

## Изменения в `lib/`

### `lib/api.ts`

- Тип `Subscription`:
  - ❌ удалить `server_id`, `connection_key`, `xui_email`, `xui_client_id` с верхнего уровня
  - ✅ добавить `sub_token: string` (но НЕ светить в UI открыто)
  - ✅ добавить `servers: SubscriptionClient[]`
- Тип `SubscriptionStatusResponse` (то что отдаёт `/api/subscription/status`):
  - `subscription_url: string` — **топ-уровень**, не внутри `subscription`
  - `subscription: Subscription`
  - `traffic_gb: { used, limit, remaining }`
  - `days_remaining: number`
  - `active: boolean`
- Новый тип `SubscriptionClient`:
  - `id: string` (id клиента, не сервера)
  - `server_id: string`
  - `xui_client_id: string`
  - `xui_email: string`
  - `connection_key: string` — VLESS URI для этого юзера на этом сервере
  - `traffic_used: number` (bytes)
  - `enabled: boolean`
  - `server: Server` — вложенный полный объект сервера
- Тип `Server` (то что в `subscription.servers[].server` и в `plan.servers[]`):
  - `id, name, country, city, flag_emoji` (имя поля `flag_emoji`!)
  - `server_address, server_port, is_active, sort_order`
  - `capacity, current_load, ping_ms?, status, last_check_at?`
- Тип `Plan`:
  - ✅ добавить `servers: Server[]`
  - Поле трафика называется `traffic_gb` (не `traffic_limit_gb`)
- Тип `User`:
  - ❌ удалить `free_region_switches`

### `lib/queries.ts`

- `useMe` — убрать использование `free_region_switches`
- `useSubscription` — обновить тип ответа
- ❌ удалить `useSwitchRegion`, `useRegionSwitchCost` (или как они называются)
- `usePlans` — каждый план теперь имеет `servers[]`

### `lib/i18n.tsx`

Удалить ключи (примерно):
- `region.switch.*`
- `region.switchCost`
- `wallet.regionSwitches`
- `region.confirmSwitch`
- ...всё что про платную смену

Добавить (примерные):
- `subscription.url.title` — «Ваша подписка»
- `subscription.url.copy` — «Скопировать ссылку»
- `subscription.url.copied` — «Скопировано»
- `subscription.url.qr` — «QR для подключения»
- `subscription.url.share` — «Поделиться»
- `subscription.howToConnect.title` — «Как подключиться»
- `subscription.howToConnect.step1/2/3` — шаги
- `subscription.servers.title` — «Доступные серверы»
- `subscription.servers.ping` — «Пинг»
- `subscription.servers.status.online/offline`
- `plans.included.servers` — «Серверы в тарифе»
- `admin.plans.serverPicker.title` — «Выберите серверы»
- `admin.plans.serverCount` — «{n} серверов в {m} регионах»
- `admin.users.subscriptionUrl` — «URL подписки»
- `admin.users.reissueToken` — «Перевыпустить токен»

## Новые компоненты которых ещё нет

1. **QR-код** — `qrcode.react` или `react-qr-code` (легче, ~10KB). Один компонент `<SubscriptionQR url={...} />`
2. **Server chip** — `<ServerChip server={...} />` с флажком, городом, пингом, статус-индикатором
3. **Server multi-select** — `<ServerPicker selectedIds={...} onChange={...} />` для админки, с группировкой по регионам и галочкой «выбрать всё в регионе»
4. **Subscription URL card** — основной элемент `MainSection` и `SubscriptionSection`. QR + URL + copy/share + кнопка «Как подключиться»
5. **Connect instructions modal** — drawer/modal с инструкциями для popular VPN-клиентов с скриншотами/иконками

## Удаление оплаты наличкой

### API изменения

#### `POST /api/subscription/buy`
- ❌ Провайдер `cash` больше не принимается. Запрос `{"provider": "cash"}` → 400.
- ✅ Допустимы только `ton` и `stars`.

#### Удалённые админ-эндпоинты
- ❌ `POST /api/admin/users/:user_id/cash-payment` — больше не существует
- ❌ `GET /api/admin/payments/cash/pending`
- ❌ `POST /api/admin/payments/cash/:payment_id/approve`
- ❌ `POST /api/admin/payments/cash/:payment_id/reject`

#### `POST /api/admin/promo` и `/api/admin/promo/bulk`
- ❌ Тип `cash_plan` больше не валиден
- ❌ Поля `plan_id` и `cash_amount_rub` в request body удалены
- ✅ Допустимые типы: только `balance` и `days`

### UI изменения

#### Юзерская часть (`PlansSection`, экран выбора оплаты)
- ❌ Убрать кнопку «Оплатить наличными» / RUB-варианта на экране тарифа
- ❌ Убрать любые upsell-сообщения «свяжитесь с представителем»
- ✅ Оставить только TON + Stars кнопки

#### Admin → `admin/cash.tsx`
- ❌ **Этот экран целиком теряет смысл** — список pending cash payments больше не существует.
- ✅ Удалить файл из навигации админки или скрыть пункт «Cash payments».

#### Admin → `admin/promo.tsx`
- ❌ Из формы создания/bulk-создания промокода убрать опцию `cash_plan`
- ❌ Убрать поля «План (для cash_plan)» и «Сумма в рублях»
- ✅ Оставить только `balance` и `days` в селекторе типа

#### Admin → `admin/users.tsx`
- ❌ Если есть кнопка «Создать cash payment» на карточке юзера — убрать
- ✅ Кнопка «Перевыпустить subscription URL» остаётся (см. раздел subscription)

#### Telegram-бот
- ❌ В админский чат больше не приходят уведомления «Новый платёж наличными» с кнопками Подтвердить/Отклонить — это бэк-only изменение, фронт не трогает.

### `lib/api.ts`
- В типе `Payment` поле `provider` теперь union `"ton" | "stars" | "balance"` (без `"cash"`)
- В типе `PromoCode` убрать поля `plan_id` и `cash_amount_rub`
- Убрать тип `PromoCodeType = "cash_plan"` если он в union

### `lib/i18n.tsx`
Удалить ключи (примерные):
- `payment.method.cash`, `payment.method.cashDesc`
- `payment.cash.contact`, `payment.cash.amount`
- `admin.cash.*` (целая ветка)
- `admin.promo.types.cashPlan`
- `promo.types.regionSwitch` (если оставались) и `promo.types.cashPlan`

### Чек-лист (cash removal)
- [ ] `src/lib/api.ts` — типы `Payment.provider`, `PromoCode`
- [ ] `src/lib/queries.ts` — убрать мутации `useCashPayment*`, `useApproveCash`, `useRejectCash`
- [ ] `src/lib/i18n.tsx` — удалить cash-ключи
- [ ] `src/components/sections/admin/cash.tsx` — удалить файл, убрать из навигации
- [ ] `src/components/sections/admin/promo.tsx` — убрать тип `cash_plan` из селектора
- [ ] `src/components/sections/admin/users.tsx` — убрать «Создать cash payment»
- [ ] Экран оплаты тарифа — оставить только TON + Stars
- [ ] Поиск по коду на `cash`/`Cash` — удалить все остатки

---

## Чек-лист по файлам

- [ ] `src/lib/api.ts` — обновить типы `Subscription`, `Plan`, `User`, добавить `ServerEntry`
- [ ] `src/lib/queries.ts` — обновить хуки, убрать switch-region
- [ ] `src/lib/i18n.tsx` — старые/новые ключи
- [ ] `src/App.tsx` — `handleShowKey` (теперь open subscription URL card)
- [ ] `src/components/sections/main.tsx` — карточка subscription URL вместо connection_key
- [ ] `src/components/sections/region.tsx` — переделать в список «Серверы», убрать платные смены
- [ ] `src/components/sections/subscription.tsx` — QR + URL как главный элемент
- [ ] `src/components/sections/wallet.tsx` — убрать упоминания смены региона
- [ ] `src/components/sections/admin/plans.tsx` — multi-select серверов в форме
- [ ] `src/components/sections/admin/users.tsx` — show + reissue subscription URL
- [ ] `src/components/subscription-qr.tsx` (новый)
- [ ] `src/components/server-chip.tsx` (новый)
- [ ] `src/components/server-picker.tsx` (новый, для админки)
- [ ] `src/components/subscription-url-card.tsx` (новый)
- [ ] `src/components/connect-instructions-modal.tsx` (новый)

## Порядок раскатки

1. Бэк мержится в main → деплой → старые endpoint'ы (switch-region) умирают, новые работают
2. Фронт обновляет типы и API-клиент → коммитит → CI собирает с новыми типами
3. Фронт переписывает UI секции → серии PR'ов

В промежутке между шагами 1 и 2 фронт **сломается** на тех экранах что юзают `subscription.server_id` / `subscription.connection_key` — нужно либо координировать релиз, либо на бэке временно отдавать deprecated-поля наряду с новыми (передавать `connection_key` из первого `servers[0]`, `server_id` оттуда же). По вкусу. Чище — одновременный релиз с feature-флагом.

## Чего НЕ делаем в этой итерации

- Свой агент / отказ от 3x-ui (отложено, нет времени)
- JSON-формат subscription URL (`/sub/<token>.json`) для sing-box — v2
- Encryption subscription URL contents — v2
- Per-tier speed limits — отложено
- Авто-провижининг существующих подписок на новые серверы плана — админ руками расширяет
