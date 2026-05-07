import { ArrowUpRight, Shield } from "lucide-react"

const BOT_LINK = "https://t.me/zyvpn_bot"

export function NoInitDataScreen() {
  return (
    <div
      className="flex flex-col items-center justify-center px-6 text-center"
      style={{ minHeight: "100dvh" }}
    >
      <div className="bg-foreground text-background flex size-16 items-center justify-center rounded-2xl shadow-lg">
        <Shield className="size-8" strokeWidth={2.25} />
      </div>
      <h1 className="text-foreground mt-6 text-2xl font-semibold tracking-tight sm:text-3xl">
        Open inside Telegram
      </h1>
      <p className="text-muted-foreground mt-3 max-w-md text-base sm:text-[17px]">
        ZyVPN is a Telegram Mini App and needs to be launched from{" "}
        <a
          href={BOT_LINK}
          target="_blank"
          rel="noreferrer"
          className="text-foreground underline-offset-4 hover:underline"
        >
          @zyvpn_bot
        </a>
        . Open the bot in Telegram and tap the menu button.
      </p>
      <a
        href={BOT_LINK}
        target="_blank"
        rel="noreferrer"
        className="bg-foreground text-background hover:bg-foreground/90 mt-8 inline-flex h-12 items-center justify-center gap-2 rounded-full px-6 text-base font-semibold tracking-tight transition-colors"
      >
        Open @zyvpn_bot
        <ArrowUpRight className="size-4" strokeWidth={2.25} />
      </a>
    </div>
  )
}
