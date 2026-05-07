import type { ComponentType, SVGProps } from "react"

export function SectionHeading({
  icon: Icon,
  children,
}: {
  icon: ComponentType<SVGProps<SVGSVGElement>>
  children: React.ReactNode
}) {
  return (
    <h2 className="text-foreground text-3xl leading-[1.1] font-medium tracking-tight sm:text-4xl">
      <span className="flex items-center gap-3">
        <Icon
          className="text-muted-foreground size-7 shrink-0 sm:size-8"
          strokeWidth={1.75}
        />
        <span>{children}</span>
      </span>
    </h2>
  )
}
