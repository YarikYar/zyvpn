import { QRCodeSVG } from "qrcode.react"

import { cn } from "@/lib/utils"

export function SubscriptionQR({
  value,
  size = 220,
  className,
}: {
  value: string
  size?: number
  className?: string
}) {
  return (
    <div
      className={cn(
        "flex aspect-square items-center justify-center rounded-2xl bg-white p-3",
        className,
      )}
    >
      {value ? (
        <QRCodeSVG
          value={value}
          size={size}
          level="M"
          bgColor="#ffffff"
          fgColor="#000000"
          className="h-full w-full"
        />
      ) : (
        <div className="text-muted-foreground text-xs">—</div>
      )}
    </div>
  )
}
