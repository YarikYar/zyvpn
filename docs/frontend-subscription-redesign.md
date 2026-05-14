# Памятка фронту: переход на подписочную модель с мульти-регионом

Бэк-ветка: `corevpn-dev`. После мержа бэка фронт должен обновиться синхронно — старая модель `subscription.server_id` / paid region switch отвалится.

## Что меняется в модели

**Было:** 1 подписка = 1 сервер; смена региона — платная транзакция (`POST /api/subscription/switch-region`), счётчик бесплатных смен в `users.free_region_switches`.

**Стало:** 1 подписка = N серверов из тарифа; юзер получает **subscription URL** (`/sub/<token>`) — её добавляют в любой VPN-клиент (v2rayNG/Hiddify/Streisand/Shadowrocket), клиент тянет список всех доступных серверов автоматически. Никаких смен.

## Изменения в API

### `GET /api/user/me`

- ❌ **уходит:** `free_region_switches`
- ✅ всё остальное без изменений (включая `is_admin`)

### `GET /api/subscription`

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

Новый shape:
```json
{
  "active": true,
  "subscription": {
    "id": "...",
    "plan_id": "...",
    "status": "active",
    "expires_at": "...",
    "subscription_url": "https://api.zaruchevskiy.ru/sub/abc123def",
    "servers": [
      {
        "id": "...",
        "name": "Germany 1",
        "country": "DE",
        "city": "Frankfurt",
        "flag": "🇩🇪",
        "ping_ms": 45,
        "status": "online",
        "connection_key": "vless://..."
      },
      { "id": "...", "name": "Netherlands 1", "country": "NL", ... }
    ]
  },
  "days_remaining": 25,
  "traffic_gb": { "used": 12, "limit": 100, "remaining": 88 }
}
```

Поля `subscription.server_id` / `subscription.connection_key` / `subscription.xui_email` на верхнем уровне — **нет**. Это всё теперь внутри `subscription.servers[]`.

### `GET /api/plans`

К каждому плану добавляется `servers[]`:
```json
{
  "id": "...",
  "name": "Standard",
  "price_ton": 5,
  "duration_days": 30,
  "traffic_limit_gb": 100,
  "servers": [
    { "id": "...", "name": "Germany 1", "country": "DE", "city": "Frankfurt", "flag": "🇩🇪" },
    { "id": "...", "name": "Netherlands 1", "country": "NL", "city": "Amsterdam", "flag": "🇳🇱" }
  ]
}
```

Юзеру в карточке плана показываем список флажков-регионов с тултипами «Germany — Frankfurt» etc.

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
  "price_ton": 5,
  "duration_days": 30,
  "traffic_limit_gb": 100,
  "server_ids": ["uuid1", "uuid2", "uuid3"]
}
```

`PATCH /api/admin/plans/:id` — то же, `server_ids` можно менять.

`GET /api/admin/users/:id/subscription` — возвращает subscription URL и список серверов (для саппорта).

`POST /api/admin/users/:id/subscription/rotate-token` — ротировать token (если URL утёк/скомпрометировали).

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

### `RegionSection` (она же `admin/servers` это не тот случай!)

Юзерская секция «Регионы» — **радикальный пересмотр**:
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
  - ✅ добавить `subscription_url: string`
  - ✅ добавить `servers: ServerEntry[]`
- Тип `Plan`:
  - ✅ добавить `servers: ServerEntry[]`
- Тип `User`:
  - ❌ удалить `free_region_switches`
- Новый тип `ServerEntry { id, name, country, city, flag, ping_ms?, status?, connection_key? }`

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
