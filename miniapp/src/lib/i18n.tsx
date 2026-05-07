import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"

import { formatTonPurchase } from "@/lib/format"

export type Locale = "en" | "ru"

type CountryCode = "DE" | "FI" | "NL" | "US" | "JP" | "SG"
type PlanId = "friend" | "starter" | "basic" | "pro"
type Load = "low" | "medium" | "high"

type PluralForms = {
  one: string
  few?: string
  many?: string
  other: string
}

type StaticMessages = {
  nav: {
    main: string
    region: string
    subscription: string
    referrals: string
    settings: string
  }
  common: {
    active: string
    popular: string
    back: string
    copy: string
    copied: string
    share: string
    until: string
    today: string
    of: string
    msUnit: string
    gb: string
    unlimited: string
  }
  main: {
    yourSubscription: string
    showKey: string
    activePrefix: string
    daysLeft: string
    term: string
    traffic: string
    used: string
    serverUptime: string
    lastNDays: (n: number) => string
    operational: string
    minorIncident: string
    majorIncident: string
    recentIncidents: string
    resolved: string
    incidentDuration: (seconds: number) => string
    incidentOngoing: string
    incidentUnknownServer: string
    greetings: {
      morning: string
      afternoon: string
      evening: string
      night: string
    }
    daysUsage: (used: number, total: number) => string
  }
  region: {
    title: string
    intro: string
    load: Record<Load, string>
    offline: string
    countries: Record<CountryCode, { country: string; city: string }>
    checkout: {
      title: string
      intro: string
      newRegion: string
      switchPrice: string
      payCta: (ton: number) => string
    }
  }
  subscription: {
    title: string
    intro: string
    plans: Record<PlanId, { name: string; tier: string; description: string }>
    daysLabel: (n: number) => string
    deviceLabel: (n: number) => string
    devicesValue: (n: number) => string
    composeDescription: (days: number, trafficGb: number | null) => string
    fields: { term: string; traffic: string; devices: string; price: string }
    checkout: {
      title: string
      paymentMethod: string
      payWithBalance: (ton: number) => string
      payTon: (ton: number) => string
      payStars: (stars: number) => string
      submitCash: string
      methods: {
        balance: { name: string; enough: string; insufficient: string }
        ton: { name: string }
        stars: { name: string }
        cash: { name: string; subtitle: (rub: number) => string }
      }
    }
    success: {
      title: string
      banner: { title: string; subtitle: string }
      vlessKey: string
      copyKey: string
      switch: string
      switchPrice: string
      howToConnect: string
      stepTitleInstall: string
      stepCopyKey: string
      stepAddFromClipboard: string
      stepConnect: string
      platformIos: string
      platformAndroid: string
      platformDesktop: string
    }
  }
  referrals: {
    title: string
    intro: (percent: number) => ReactNode
    earnedSoFar: string
    invited: string
    pending: string
    howItWorks: string
    yourInviteLink: string
    stepLabel: (i: number) => string
    steps: {
      share: { title: string; description: string }
      friend: { title: string; description: string }
      reward: { title: string; description: (percent: number) => string }
    }
    shareText: string
  }
  settings: {
    title: string
    description: string
    appearance: string
    theme: string
    themeSubtitle: string
    themeLight: string
    themeDark: string
    language: string
    languageSubtitle: string
  }
  maintenance: {
    title: string
    subtitle: string
    retry: string
  }
  wallet: {
    title: string
    heroApprox: (rub: number) => string
    promoTitle: string
    promoPlaceholder: string
    promoSubmit: string
    topUp: string
    topUpVia: { ton: string; stars: string }
    history: string
    tx: {
      subscriptionPayment: string
      regionSwitch: string
      promo: string
      topup: string
      referralReward: string
    }
  }
}

const pluralRules: Record<Locale, Intl.PluralRules> = {
  en: new Intl.PluralRules("en"),
  ru: new Intl.PluralRules("ru"),
}

function plural(locale: Locale, n: number, forms: PluralForms): string {
  const cat = pluralRules[locale].select(n)
  if (cat === "one") return forms.one
  if (cat === "few" && forms.few) return forms.few
  if (cat === "many" && forms.many) return forms.many
  return forms.other
}

function formatIncidentDurationEn(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "0s"
  const total = Math.round(seconds)
  if (total < 60) return `${total}s`
  if (total < 3600) {
    const minutes = Math.floor(total / 60)
    return `${minutes} min`
  }
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  return `${hours}h ${String(minutes).padStart(2, "0")}m`
}

function formatIncidentDurationRu(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "0 с"
  const total = Math.round(seconds)
  if (total < 60) return `${total} с`
  if (total < 3600) {
    const minutes = Math.floor(total / 60)
    return `${minutes} ${plural("ru", minutes, { one: "мин", few: "мин", many: "мин", other: "мин" })}`
  }
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  return `${hours} ч ${String(minutes).padStart(2, "0")} мин`
}

const en: StaticMessages = {
  nav: {
    main: "Main",
    region: "Region",
    subscription: "Subscription",
    referrals: "Referrals",
    settings: "Settings",
  },
  common: {
    active: "Active",
    popular: "Popular",
    back: "Back",
    copy: "Copy",
    copied: "Copied",
    share: "Share",
    until: "Until",
    today: "Today",
    of: "of",
    msUnit: "ms",
    gb: "GB",
    unlimited: "Unlimited",
  },
  main: {
    yourSubscription: "Your subscription",
    showKey: "Show key",
    activePrefix: "Active",
    daysLeft: "days left",
    term: "Term",
    traffic: "Traffic",
    used: "used",
    serverUptime: "Server uptime",
    lastNDays: (n) => `Last ${n} days`,
    operational: "Operational",
    minorIncident: "Minor incident",
    majorIncident: "Major incident",
    recentIncidents: "Recent incidents",
    resolved: "Resolved",
    incidentDuration: (seconds) => formatIncidentDurationEn(seconds),
    incidentOngoing: "Ongoing",
    incidentUnknownServer: "Unknown server",
    greetings: {
      morning: "Good morning",
      afternoon: "Good afternoon",
      evening: "Good evening",
      night: "Good night",
    },
    daysUsage: (used, total) =>
      `${used} of ${total} ${plural("en", total, { one: "day", other: "days" })}`,
  },
  region: {
    title: "Regions",
    intro: "Pick the closest server for the best speed.",
    load: { low: "Low load", medium: "Moderate load", high: "High load" },
    offline: "Offline",
    countries: {
      DE: { country: "Germany", city: "Frankfurt" },
      FI: { country: "Finland", city: "Helsinki" },
      NL: { country: "Netherlands", city: "Amsterdam" },
      US: { country: "USA", city: "New York" },
      JP: { country: "Japan", city: "Tokyo" },
      SG: { country: "Singapore", city: "Singapore" },
    },
    checkout: {
      title: "Change region",
      intro:
        "Switch to a different server. Active sessions will reroute through the new region.",
      newRegion: "New region",
      switchPrice: "0.10 TON",
      payCta: (ton) => `Pay ${formatTonPurchase(ton)} TON`,
    },
  },
  subscription: {
    title: "Subscription",
    intro:
      "Pick a plan and the payment method that suits you. Activation is instant.",
    plans: {
      friend: {
        name: "Friend pass",
        tier: "Referral",
        description: "Unlimited month, unlocked by your referrer",
      },
      starter: {
        name: "Starter",
        tier: "Entry",
        description: "7 days of VPN access, 50 GB traffic",
      },
      basic: {
        name: "Basic",
        tier: "Standard",
        description: "30 days of VPN access, 100 GB traffic",
      },
      pro: {
        name: "Pro",
        tier: "Premium",
        description: "30 days of VPN access, unlimited traffic",
      },
    },
    daysLabel: (n) =>
      `${n} ${plural("en", n, { one: "day", other: "days" })}`,
    deviceLabel: (n) =>
      `${n} ${plural("en", n, { one: "device", other: "devices" })}`,
    devicesValue: (n) =>
      `${n} ${plural("en", n, { one: "device", other: "devices" })}`,
    composeDescription: (days, trafficGb) => {
      const dayWord = plural("en", days, { one: "day", other: "days" })
      if (trafficGb === null) {
        return `${days} ${dayWord} of VPN access, unlimited traffic`
      }
      return `${days} ${dayWord} of VPN access, ${trafficGb} GB traffic`
    },
    fields: {
      term: "Term",
      traffic: "Traffic",
      devices: "Devices",
      price: "Price",
    },
    checkout: {
      title: "Checkout",
      paymentMethod: "Payment method",
      payWithBalance: (ton) => `Pay with balance (${formatTonPurchase(ton)} TON)`,
      payTon: (ton) => `Pay ${formatTonPurchase(ton)} TON`,
      payStars: (stars) => `Pay ${stars} Stars`,
      submitCash: "Submit cash request",
      methods: {
        balance: {
          name: "Balance",
          enough: "(enough)",
          insufficient: "(insufficient)",
        },
        ton: { name: "TON" },
        stars: { name: "Telegram Stars" },
        cash: {
          name: "Cash to agent",
          subtitle: (rub) => `≈${rub} ₽ – activation after confirmation`,
        },
      },
    },
    success: {
      title: "Your key",
      banner: {
        title: "Payment successful",
        subtitle:
          "Your subscription is now active. Connect using the key below.",
      },
      vlessKey: "VLESS key",
      copyKey: "Copy key",
      switch: "Switch",
      switchPrice: "(0.10 TON)",
      howToConnect: "How to connect",
      stepTitleInstall: "Install a VPN client",
      stepCopyKey: "Copy the key above",
      stepAddFromClipboard: "In the app, choose “Add from clipboard”",
      stepConnect: "Connect – and you're online.",
      platformIos: "iOS:",
      platformAndroid: "Android:",
      platformDesktop: "Windows / Mac:",
    },
  },
  referrals: {
    title: "Referrals",
    intro: (percent) => (
      <>
        Invite friends and earn{" "}
        <span className="text-foreground font-semibold">{percent}%</span> of
        every payment they make – straight to your balance.
      </>
    ),
    earnedSoFar: "Earned so far",
    invited: "Invited",
    pending: "Pending",
    howItWorks: "How it works",
    yourInviteLink: "Your invite link",
    stepLabel: (i) => `Step ${i}`,
    steps: {
      share: {
        title: "Share your link",
        description: "Send your personal invite link to a friend.",
      },
      friend: {
        title: "Friend signs up",
        description: "They join via your link and activate any plan.",
      },
      reward: {
        title: "You get rewarded",
        description: (percent) =>
          `${percent}% of every payment your friend makes lands on your balance.`,
      },
    },
    shareText: "Join me on ZYVPN – fast, private and a free bonus on me.",
  },
  settings: {
    title: "Settings",
    description: "Tweak the app to your liking. Changes save instantly.",
    appearance: "Appearance",
    theme: "Theme",
    themeSubtitle: "Light or dark interface",
    themeLight: "Light",
    themeDark: "Dark",
    language: "Language",
    languageSubtitle: "Interface language",
  },
  maintenance: {
    title: "Service is temporarily unavailable",
    subtitle: "We're working on it. Try again in a few minutes.",
    retry: "Try again",
  },
  wallet: {
    title: "Balance",
    heroApprox: (rub) => `TON ≈ ${rub} ₽`,
    promoTitle: "Promo code",
    promoPlaceholder: "Enter promo code",
    promoSubmit: "OK",
    topUp: "Top up balance",
    topUpVia: { ton: "TON", stars: "Stars" },
    history: "Transaction history",
    tx: {
      subscriptionPayment: "Subscription payment",
      regionSwitch: "Region switch",
      promo: "Promo code",
      topup: "Top-up",
      referralReward: "Referral reward",
    },
  },
}

const ru: StaticMessages = {
  nav: {
    main: "Главная",
    region: "Регион",
    subscription: "Подписка",
    referrals: "Рефералы",
    settings: "Настройки",
  },
  common: {
    active: "Активен",
    popular: "Популярный",
    back: "Назад",
    copy: "Копировать",
    copied: "Скопировано",
    share: "Поделиться",
    until: "До",
    today: "Сегодня",
    of: "из",
    msUnit: "мс",
    gb: "ГБ",
    unlimited: "Безлимит",
  },
  main: {
    yourSubscription: "Ваша подписка",
    showKey: "Показать ключ",
    activePrefix: "Активна",
    daysLeft: "дней",
    term: "Срок",
    traffic: "Трафик",
    used: "использовано",
    serverUptime: "Аптайм сервера",
    lastNDays: (n) =>
      `За ${n} ${plural("ru", n, { one: "день", few: "дня", many: "дней", other: "дней" })}`,
    operational: "Работает",
    minorIncident: "Малый сбой",
    majorIncident: "Серьёзный сбой",
    recentIncidents: "Последние инциденты",
    resolved: "Решено",
    incidentDuration: (seconds) => formatIncidentDurationRu(seconds),
    incidentOngoing: "В процессе",
    incidentUnknownServer: "Неизвестный сервер",
    greetings: {
      morning: "Доброе утро",
      afternoon: "Добрый день",
      evening: "Добрый вечер",
      night: "Доброй ночи",
    },
    daysUsage: (used, total) =>
      `${used} из ${total} ${plural("ru", total, { one: "дня", few: "дней", many: "дней", other: "дней" })}`,
  },
  region: {
    title: "Регионы",
    intro: "Выбери ближайший сервер – будет быстрее.",
    load: { low: "Низкая нагрузка", medium: "Умеренная нагрузка", high: "Высокая нагрузка" },
    offline: "Не в сети",
    countries: {
      DE: { country: "Германия", city: "Франкфурт" },
      FI: { country: "Финляндия", city: "Хельсинки" },
      NL: { country: "Нидерланды", city: "Амстердам" },
      US: { country: "США", city: "Нью-Йорк" },
      JP: { country: "Япония", city: "Токио" },
      SG: { country: "Сингапур", city: "Сингапур" },
    },
    checkout: {
      title: "Смена региона",
      intro:
        "Сменить сервер. Активные сессии переключатся через новый регион.",
      newRegion: "Новый регион",
      switchPrice: "0.10 TON",
      payCta: (ton) => `Оплатить ${formatTonPurchase(ton)} TON`,
    },
  },
  subscription: {
    title: "Подписка",
    intro:
      "Выбери тариф и удобный способ оплаты. Активация мгновенная.",
    plans: {
      friend: {
        name: "Свой кент",
        tier: "Реферал",
        description: "Месяц безлимита, открывает твой реферер",
      },
      starter: {
        name: "Старт",
        tier: "Начальный",
        description: "7 дней доступа, 50 ГБ трафика",
      },
      basic: {
        name: "Базовый",
        tier: "Стандарт",
        description: "30 дней доступа, 100 ГБ трафика",
      },
      pro: {
        name: "Pro",
        tier: "Премиум",
        description: "30 дней доступа, безлимитный трафик",
      },
    },
    daysLabel: (n) =>
      `${n} ${plural("ru", n, { one: "день", few: "дня", many: "дней", other: "дней" })}`,
    deviceLabel: (n) =>
      plural("ru", n, { one: "устройство", few: "устройства", many: "устройств", other: "устройств" }),
    devicesValue: (n) =>
      `${n} ${plural("ru", n, { one: "устройство", few: "устройства", many: "устройств", other: "устройств" })}`,
    composeDescription: (days, trafficGb) => {
      const dayWord = plural("ru", days, { one: "день", few: "дня", many: "дней", other: "дней" })
      if (trafficGb === null) {
        return `${days} ${dayWord} доступа, безлимитный трафик`
      }
      return `${days} ${dayWord} доступа, ${trafficGb} ГБ трафика`
    },
    fields: {
      term: "Срок",
      traffic: "Трафик",
      devices: "Устройства",
      price: "Цена",
    },
    checkout: {
      title: "Оплата",
      paymentMethod: "Способ оплаты",
      payWithBalance: (ton) => `Оплатить с баланса (${formatTonPurchase(ton)} TON)`,
      payTon: (ton) => `Оплатить ${formatTonPurchase(ton)} TON`,
      payStars: (stars) => `Оплатить ${stars} Stars`,
      submitCash: "Запросить оплату наличными",
      methods: {
        balance: {
          name: "Баланс",
          enough: "(хватает)",
          insufficient: "(недостаточно)",
        },
        ton: { name: "TON" },
        stars: { name: "Telegram Stars" },
        cash: {
          name: "Наличными представителю",
          subtitle: (rub) => `≈${rub} ₽ – активация после подтверждения`,
        },
      },
    },
    success: {
      title: "Ваш ключ",
      banner: {
        title: "Оплата прошла",
        subtitle: "Подписка активна. Подключайся по ключу ниже.",
      },
      vlessKey: "VLESS-ключ",
      copyKey: "Скопировать ключ",
      switch: "Сменить",
      switchPrice: "(0.10 TON)",
      howToConnect: "Как подключиться",
      stepTitleInstall: "Установи VPN-клиент",
      stepCopyKey: "Скопируй ключ выше",
      stepAddFromClipboard: "В приложении выбери «Добавить из буфера»",
      stepConnect: "Подключайся – и ты в сети.",
      platformIos: "iOS:",
      platformAndroid: "Android:",
      platformDesktop: "Windows / Mac:",
    },
  },
  referrals: {
    title: "Рефералы",
    intro: (percent) => (
      <>
        Приглашай друзей и получай{" "}
        <span className="text-foreground font-semibold">{percent}%</span> с
        каждого их платежа – прямо на баланс.
      </>
    ),
    earnedSoFar: "Заработано всего",
    invited: "Приглашено",
    pending: "Ожидают",
    howItWorks: "Как это работает",
    yourInviteLink: "Твоя ссылка",
    stepLabel: (i) => `Шаг ${i}`,
    steps: {
      share: {
        title: "Поделись ссылкой",
        description: "Отправь персональную пригласительную ссылку другу.",
      },
      friend: {
        title: "Друг регистрируется",
        description: "Он переходит по твоей ссылке и активирует любой тариф.",
      },
      reward: {
        title: "Ты получаешь награду",
        description: (percent) =>
          `${percent}% от каждого платежа друга падает тебе на баланс.`,
      },
    },
    shareText: "Залетай в ZYVPN – быстро, приватно, и бонус от меня.",
  },
  settings: {
    title: "Настройки",
    description: "Подстрой приложение под себя. Изменения применяются сразу.",
    appearance: "Оформление",
    theme: "Тема",
    themeSubtitle: "Светлый или тёмный интерфейс",
    themeLight: "Светлая",
    themeDark: "Тёмная",
    language: "Язык",
    languageSubtitle: "Язык интерфейса",
  },
  maintenance: {
    title: "Сервис временно недоступен",
    subtitle: "Уже чиним. Попробуй через пару минут.",
    retry: "Повторить",
  },
  wallet: {
    title: "Баланс",
    heroApprox: (rub) => `TON ≈ ${rub} ₽`,
    promoTitle: "Промокод",
    promoPlaceholder: "Введите промокод",
    promoSubmit: "ОК",
    topUp: "Пополнить баланс",
    topUpVia: { ton: "TON", stars: "Stars" },
    history: "История операций",
    tx: {
      subscriptionPayment: "Оплата подписки",
      regionSwitch: "Смена региона",
      promo: "Промокод",
      topup: "Пополнение",
      referralReward: "Реферальная награда",
    },
  },
}

const tables: Record<Locale, StaticMessages> = { en, ru }

type LocaleContextValue = {
  locale: Locale
  setLocale: (l: Locale) => void
  t: StaticMessages
}

const LocaleContext = createContext<LocaleContextValue | null>(null)

const STORAGE_KEY = "locale"

function detectInitialLocale(): Locale {
  if (typeof window === "undefined") return "en"
  try {
    const saved = window.localStorage.getItem(STORAGE_KEY)
    if (saved === "en" || saved === "ru") return saved
  } catch {
    /* noop */
  }
  type TelegramUser = { language_code?: string }
  type TelegramWebApp = { initDataUnsafe?: { user?: TelegramUser } }
  const tgWindow = window as { Telegram?: { WebApp?: TelegramWebApp } }
  const tgLang = tgWindow.Telegram?.WebApp?.initDataUnsafe?.user?.language_code
  if (tgLang?.toLowerCase().startsWith("ru")) return "ru"
  if (
    typeof navigator !== "undefined" &&
    navigator.language?.toLowerCase().startsWith("ru")
  ) {
    return "ru"
  }
  return "en"
}

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => detectInitialLocale())

  useEffect(() => {
    document.documentElement.setAttribute("lang", locale)
  }, [locale])

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l)
    try {
      window.localStorage.setItem(STORAGE_KEY, l)
    } catch {
      /* noop */
    }
  }, [])

  const value = useMemo<LocaleContextValue>(
    () => ({ locale, setLocale, t: tables[locale] }),
    [locale, setLocale],
  )

  return (
    <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
  )
}

export function useLocale() {
  const ctx = useContext(LocaleContext)
  if (!ctx) throw new Error("useLocale must be used within LocaleProvider")
  return ctx
}
