import {
  ChevronRight,
  Globe,
  Moon,
  Palette,
  Settings as SettingsIcon,
  Shield,
  Sun,
} from "lucide-react"

import { SectionHeading } from "@/components/sections/section-heading"
import { SegmentedToggle } from "@/components/ui/segmented-toggle"
import { useTheme } from "@/hooks/use-theme"
import { useLocale } from "@/lib/i18n"
import { useMe } from "@/lib/queries"

export function SettingsSection({
  onOpenAdmin,
}: {
  onOpenAdmin?: () => void
}) {
  const { t } = useLocale()
  const { theme, setTheme } = useTheme()
  const { locale, setLocale } = useLocale()
  const meQuery = useMe()
  const isAdmin = !!meQuery.data?.is_admin

  return (
    <section className="px-6 pb-16">
      <SectionHeading icon={SettingsIcon}>{t.settings.title}</SectionHeading>
      <p className="text-muted-foreground mt-3 max-w-md text-base sm:text-[17px]">
        {t.settings.description}
      </p>

      <h3 className="text-foreground mt-10 inline-flex items-center gap-2 text-lg font-semibold tracking-tight">
        <Palette className="text-muted-foreground size-4" strokeWidth={2} />
        {t.settings.appearance}
      </h3>
      <ul className="mt-3 space-y-2">
        <li className="bg-card border-border rounded-2xl border p-4">
          <div className="flex items-center gap-3">
            <span className="bg-foreground/10 text-foreground inline-flex size-9 shrink-0 items-center justify-center rounded-xl">
              {theme === "dark" ? (
                <Moon className="size-4" strokeWidth={2} />
              ) : (
                <Sun className="size-4" strokeWidth={2} />
              )}
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-foreground font-semibold tracking-tight">
                {t.settings.theme}
              </p>
              <p className="text-muted-foreground mt-0.5 text-sm">
                {t.settings.themeSubtitle}
              </p>
            </div>
            <SegmentedToggle
              value={theme}
              onChange={setTheme}
              options={[
                { value: "light", label: t.settings.themeLight, icon: Sun },
                { value: "dark", label: t.settings.themeDark, icon: Moon },
              ]}
            />
          </div>
        </li>

        <li className="bg-card border-border rounded-2xl border p-4">
          <div className="flex items-center gap-3">
            <span className="bg-foreground/10 text-foreground inline-flex size-9 shrink-0 items-center justify-center rounded-xl">
              <Globe className="size-4" strokeWidth={2} />
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-foreground font-semibold tracking-tight">
                {t.settings.language}
              </p>
              <p className="text-muted-foreground mt-0.5 text-sm">
                {t.settings.languageSubtitle}
              </p>
            </div>
            <SegmentedToggle
              value={locale}
              onChange={setLocale}
              options={[
                { value: "en", label: "EN" },
                { value: "ru", label: "RU" },
              ]}
            />
          </div>
        </li>
      </ul>

      {isAdmin && onOpenAdmin && (
        <>
          <h3 className="text-foreground mt-10 inline-flex items-center gap-2 text-lg font-semibold tracking-tight">
            <Shield className="text-muted-foreground size-4" strokeWidth={2} />
            {t.settings.administration}
          </h3>
          <button
            type="button"
            onClick={onOpenAdmin}
            className="bg-card border-border hover:bg-muted/40 mt-3 flex w-full items-center gap-3 rounded-2xl border p-4 text-left transition-colors"
          >
            <span className="inline-flex size-9 shrink-0 items-center justify-center rounded-xl bg-foreground/10 text-foreground">
              <Shield className="size-4" strokeWidth={2} />
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-foreground font-semibold tracking-tight">
                {t.settings.adminEntry}
              </p>
              <p className="text-muted-foreground mt-0.5 text-sm">
                {t.settings.adminEntrySubtitle}
              </p>
            </div>
            <ChevronRight
              className="text-muted-foreground size-4 shrink-0"
              strokeWidth={2}
            />
          </button>
        </>
      )}
    </section>
  )
}
