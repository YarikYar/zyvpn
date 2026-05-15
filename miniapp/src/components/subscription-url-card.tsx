import { useState } from "react"
import { Check, Copy, Link2, Share2 } from "lucide-react"
import { AnimatePresence, motion } from "motion/react"

import { SubscriptionQR } from "@/components/subscription-qr"
import { ConnectInstructions } from "@/components/connect-instructions-modal"
import { useLocale } from "@/lib/i18n"
import { copyToClipboard, shareLink, triggerHaptic } from "@/lib/telegram"
import { cn } from "@/lib/utils"

const FEEDBACK_SPRING = {
  type: "spring",
  stiffness: 420,
  damping: 28,
  mass: 0.8,
} as const

export function SubscriptionUrlCard({
  url,
  className,
}: {
  url: string
  className?: string
}) {
  const { t } = useLocale()
  const [copied, setCopied] = useState(false)
  const [instructionsOpen, setInstructionsOpen] = useState(false)

  const handleCopy = async () => {
    if (!url) return
    const ok = await copyToClipboard(url)
    if (!ok) return
    triggerHaptic({ type: "notification", status: "success" })
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1600)
  }

  const handleShare = () => {
    if (!url) return
    triggerHaptic({ type: "selection" })
    shareLink(url, t.subscription.url.shareText)
  }

  return (
    <div className={cn("space-y-3", className)}>
      <div className="bg-card border-border relative overflow-hidden rounded-3xl border bg-gradient-to-br from-foreground/5 via-foreground/2 to-transparent p-5 shadow-lg">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <p className="text-muted-foreground inline-flex items-center gap-1.5 text-xs font-semibold tracking-[0.18em] uppercase">
              <Link2 className="size-3" strokeWidth={2.5} />
              {t.subscription.url.title}
            </p>
            <p className="text-muted-foreground mt-1.5 max-w-md text-xs leading-snug sm:text-sm">
              {t.subscription.url.helper}
            </p>
          </div>
        </div>

        <div className="mt-5 flex justify-center">
          <SubscriptionQR
            value={url}
            size={240}
            className="size-56 sm:size-64"
          />
        </div>
        <p className="text-muted-foreground mt-3 text-center text-xs">
          {t.subscription.url.qrCaption}
        </p>

        <div className="bg-background border-border/70 mt-5 rounded-2xl border p-3.5">
          <p className="text-muted-foreground text-[10px] font-semibold tracking-[0.18em] uppercase">
            URL
          </p>
          <p
            className="text-foreground/90 mt-1.5 font-mono text-[12px] leading-relaxed break-all sm:text-[13px]"
            style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace" }}
          >
            {url || "—"}
          </p>
        </div>

        <div className="mt-3 grid grid-cols-2 gap-2">
          <button
            type="button"
            onClick={handleCopy}
            disabled={!url}
            className="bg-foreground text-background hover:bg-foreground/90 disabled:bg-muted disabled:text-muted-foreground relative inline-flex h-12 items-center justify-center gap-2 rounded-2xl text-sm font-semibold tracking-tight transition-colors disabled:cursor-not-allowed"
          >
            <AnimatePresence mode="wait" initial={false}>
              {copied ? (
                <motion.span
                  key="copied"
                  initial={{ opacity: 0, scale: 0.7 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.7 }}
                  transition={FEEDBACK_SPRING}
                  className="inline-flex items-center gap-2"
                >
                  <Check className="size-4" strokeWidth={2.5} />
                  {t.subscription.url.copied}
                </motion.span>
              ) : (
                <motion.span
                  key="copy"
                  initial={{ opacity: 0, scale: 0.7 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.7 }}
                  transition={FEEDBACK_SPRING}
                  className="inline-flex items-center gap-2"
                >
                  <Copy className="size-4" strokeWidth={2} />
                  {t.subscription.url.copy}
                </motion.span>
              )}
            </AnimatePresence>
          </button>
          <button
            type="button"
            onClick={handleShare}
            disabled={!url}
            className="bg-background border-border text-foreground hover:bg-muted/60 inline-flex h-12 items-center justify-center gap-2 rounded-2xl border text-sm font-semibold tracking-tight transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Share2 className="size-4" strokeWidth={2.25} />
            {t.subscription.url.share}
          </button>
        </div>
      </div>

      <ConnectInstructions
        open={instructionsOpen}
        onToggle={() => setInstructionsOpen((v) => !v)}
      />
    </div>
  )
}
