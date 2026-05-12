import { useState } from "react"
import { Ban, ShieldOff, User, Globe } from "lucide-react"

import { SegmentedToggle } from "@/components/ui/segmented-toggle"
import { useLocale } from "@/lib/i18n"
import {
  useAdminBans,
  useAdminBansIp,
  useAdminUnbanIp,
  useAdminUserMutations,
} from "@/lib/queries"

import {
  AdminCard,
  AdminEmpty,
  AdminError,
  AdminLoader,
  FormField,
  GhostButton,
  PrimaryButton,
  TextInput,
  formatAdminDate,
} from "./shared"

type Tab = "users" | "ip"

export function AdminBansView() {
  const { t } = useLocale()
  const [tab, setTab] = useState<Tab>("users")

  return (
    <div className="mt-6">
      <SegmentedToggle<Tab>
        value={tab}
        onChange={setTab}
        options={[
          { value: "users", label: t.admin.bans.tabUsers, icon: User },
          { value: "ip", label: t.admin.bans.tabIp, icon: Globe },
        ]}
      />

      {tab === "users" && <UserBansList />}
      {tab === "ip" && <IpBansList />}
    </div>
  )
}

function UserBansList() {
  const { t, locale } = useLocale()
  const query = useAdminBans()

  if (query.isPending) return <AdminLoader />
  if (query.error) return <AdminError error={query.error} />
  if (!query.data || query.data.length === 0)
    return <AdminEmpty>{t.admin.bans.emptyUsers}</AdminEmpty>

  return (
    <ul className="mt-4 space-y-2">
      {query.data.map((b) => (
        <UserBanRow
          key={b.user_id}
          userId={b.user_id}
          name={
            b.username
              ? `@${b.username}`
              : b.first_name || `ID ${b.user_id}`
          }
          reason={b.reason}
          date={formatAdminDate(b.banned_at, locale)}
        />
      ))}
    </ul>
  )
}

function UserBanRow({
  userId,
  name,
  reason,
  date,
}: {
  userId: number
  name: string
  reason?: string
  date?: string
}) {
  const { t } = useLocale()
  const m = useAdminUserMutations(userId)
  return (
    <li className="bg-card border-border flex items-start gap-3 rounded-2xl border p-4">
      <span className="inline-flex size-9 shrink-0 items-center justify-center rounded-xl bg-rose-500/15 text-rose-600 dark:text-rose-300">
        <Ban className="size-4" strokeWidth={2.25} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-foreground truncate text-sm font-semibold tracking-tight">
          {name}
        </p>
        <p className="text-muted-foreground mt-0.5 truncate text-xs tabular-nums">
          ID {userId}
          {date && <> · {date}</>}
        </p>
        {reason && (
          <p className="text-muted-foreground mt-1.5 line-clamp-2 text-xs">
            {reason}
          </p>
        )}
      </div>
      <GhostButton
        onClick={() => {
          if (window.confirm(t.admin.users.confirmUnban)) {
            m.unban.mutate()
          }
        }}
        loading={m.unban.isPending}
        className="h-9 shrink-0 px-3 text-xs"
      >
        <ShieldOff className="size-3.5" strokeWidth={2} />
        {t.admin.bans.unban}
      </GhostButton>
    </li>
  )
}

function IpBansList() {
  const { t, locale } = useLocale()
  const query = useAdminBansIp()
  const unbanIp = useAdminUnbanIp()

  return (
    <div className="mt-5">
      <AddIpForm />

      {query.isPending && <AdminLoader />}
      {query.error && <AdminError error={query.error} />}
      {query.data && query.data.length === 0 && (
        <AdminEmpty>{t.admin.bans.emptyIps}</AdminEmpty>
      )}

      {query.data && query.data.length > 0 && (
        <ul className="mt-5 space-y-2">
          {query.data.map((b) => (
            <li
              key={b.ip}
              className="bg-card border-border flex items-start gap-3 rounded-2xl border p-4"
            >
              <span className="inline-flex size-9 shrink-0 items-center justify-center rounded-xl bg-rose-500/15 text-rose-600 dark:text-rose-300">
                <Globe className="size-4" strokeWidth={2.25} />
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-foreground truncate font-mono text-sm font-semibold">
                  {b.ip}
                </p>
                {b.banned_at && (
                  <p className="text-muted-foreground mt-0.5 truncate text-xs tabular-nums">
                    {formatAdminDate(b.banned_at, locale)}
                  </p>
                )}
                {b.reason && (
                  <p className="text-muted-foreground mt-1.5 line-clamp-2 text-xs">
                    {b.reason}
                  </p>
                )}
              </div>
              <GhostButton
                onClick={() => unbanIp.mutate({ ip: b.ip })}
                loading={unbanIp.isPending && unbanIp.variables?.ip === b.ip}
                className="h-9 shrink-0 px-3 text-xs"
              >
                <ShieldOff className="size-3.5" strokeWidth={2} />
                {t.admin.bans.unban}
              </GhostButton>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function AddIpForm() {
  const { t } = useLocale()
  const unbanIp = useAdminUnbanIp()
  const [ip, setIp] = useState("")
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    if (!ip.trim()) return
    try {
      await unbanIp.mutateAsync({ ip: ip.trim() })
      setIp("")
    } catch (err) {
      setError(err instanceof Error ? err.message : t.admin.common.error)
    }
  }

  return (
    <AdminCard>
      <p className="text-foreground inline-flex items-center gap-2 text-sm font-semibold tracking-tight">
        <Globe className="text-muted-foreground size-4" strokeWidth={2} />
        {t.admin.bans.addIp}
      </p>
      <form onSubmit={handleSubmit} className="mt-3 space-y-2.5">
        <FormField label={t.admin.bans.addIp}>
          <TextInput
            value={ip}
            onChange={(e) => setIp(e.target.value)}
            placeholder={t.admin.bans.ipPlaceholder}
          />
        </FormField>
        <PrimaryButton type="submit" loading={unbanIp.isPending} className="w-full">
          {t.admin.bans.addIpCta}
        </PrimaryButton>
        {error && (
          <p className="text-xs text-rose-500 dark:text-rose-400">{error}</p>
        )}
      </form>
    </AdminCard>
  )
}
