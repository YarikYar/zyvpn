import { useState } from "react"
import { Activity, Pencil, Plus, Server, Trash2 } from "lucide-react"
import { AnimatePresence, motion } from "motion/react"

import { flagFor } from "@/lib/flags"
import { useLocale } from "@/lib/i18n"
import {
  useAdminServerCreate,
  useAdminServerDelete,
  useAdminServerTest,
  useAdminServerUpdate,
  useAdminServers,
} from "@/lib/queries"
import type { AdminServerInput, Server as ServerType } from "@/lib/api"
import { cn } from "@/lib/utils"

import {
  AdminEmpty,
  AdminError,
  AdminLoader,
  DestructiveButton,
  FormField,
  GhostButton,
  MutationFeedback,
  PrimaryButton,
  TextInput,
  Toggle,
} from "./shared"

type Mode = { kind: "list" } | { kind: "form"; server: ServerType | null }

export function AdminServersView() {
  const { t } = useLocale()
  const servers = useAdminServers()
  const del = useAdminServerDelete()
  const test = useAdminServerTest()
  const [mode, setMode] = useState<Mode>({ kind: "list" })
  const [testResult, setTestResult] =
    useState<{ id: string; ok: boolean; ping?: number; error?: string } | null>(
      null,
    )

  const handleTest = async (id: string) => {
    setTestResult(null)
    try {
      const result = await test.mutateAsync(id)
      setTestResult({
        id,
        ok: result.ok,
        ping: result.ping_ms,
        error: result.message,
      })
    } catch (err) {
      setTestResult({
        id,
        ok: false,
        error: err instanceof Error ? err.message : t.admin.common.error,
      })
    }
  }

  return (
    <div className="mt-6">
      {mode.kind === "list" && (
        <>
          <PrimaryButton
            onClick={() => setMode({ kind: "form", server: null })}
            className="w-full"
          >
            <Plus className="size-4" strokeWidth={2.25} />
            {t.admin.servers.createCta}
          </PrimaryButton>

          {servers.isPending && <AdminLoader />}
          {servers.error && <AdminError error={servers.error} />}
          {servers.data && servers.data.length === 0 && (
            <AdminEmpty>{t.admin.servers.empty}</AdminEmpty>
          )}

          {servers.data && servers.data.length > 0 && (
            <ul className="mt-4 space-y-2">
              {servers.data.map((s) => {
                const code = (s.country ?? "").toUpperCase()
                const Flag = flagFor(code)
                const online = s.status !== "offline"
                const isTesting = test.isPending && test.variables === s.id
                const lastResult = testResult?.id === s.id ? testResult : null
                return (
                  <li
                    key={s.id}
                    className="bg-card border-border rounded-2xl border p-4"
                  >
                    <div className="flex items-start gap-3">
                      <span className="inline-flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-foreground/5">
                        {Flag ? (
                          <Flag
                            aria-hidden
                            className="h-7 aspect-[3/2] rounded"
                          />
                        ) : (
                          <Server className="text-muted-foreground size-4" strokeWidth={2} />
                        )}
                      </span>
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-1.5">
                          <p className="text-foreground truncate text-base font-semibold tracking-tight">
                            {s.name || s.city || s.country}
                          </p>
                          <span
                            className={cn(
                              "inline-flex shrink-0 items-center gap-1 rounded-full px-1.5 py-0.5 text-[9px] font-semibold tracking-[0.12em] uppercase",
                              online
                                ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-300"
                                : "bg-rose-500/15 text-rose-600 dark:text-rose-300",
                            )}
                          >
                            <span
                              aria-hidden
                              className={cn(
                                "size-1.5 rounded-full",
                                online ? "bg-emerald-500" : "bg-rose-500",
                              )}
                            />
                            {online ? t.admin.servers.online : t.admin.servers.offline}
                          </span>
                        </div>
                        <p className="text-muted-foreground mt-0.5 truncate text-xs tabular-nums">
                          {s.id} · {code}
                          {typeof s.ping_ms === "number" && <> · {s.ping_ms} ms</>}
                        </p>
                        {lastResult && (
                          <p
                            className={cn(
                              "mt-1.5 text-[11px] font-semibold",
                              lastResult.ok
                                ? "text-emerald-600 dark:text-emerald-400"
                                : "text-rose-600 dark:text-rose-400",
                            )}
                          >
                            {lastResult.ok
                              ? t.admin.servers.testOk(lastResult.ping ?? 0)
                              : lastResult.error || t.admin.servers.testFail}
                          </p>
                        )}
                      </div>
                    </div>
                    <div className="mt-3 grid grid-cols-3 gap-2">
                      <GhostButton
                        className="h-9 text-xs"
                        onClick={() => handleTest(s.id)}
                        loading={isTesting}
                      >
                        <Activity className="size-3.5" strokeWidth={2} />
                        {isTesting ? t.admin.servers.testRunning : t.admin.servers.testCta}
                      </GhostButton>
                      <GhostButton
                        className="h-9 text-xs"
                        onClick={() => setMode({ kind: "form", server: s })}
                      >
                        <Pencil className="size-3.5" strokeWidth={2} />
                        {t.admin.servers.editCta}
                      </GhostButton>
                      <DestructiveButton
                        className="h-9 text-xs"
                        onClick={() => {
                          if (window.confirm(t.admin.servers.confirmDelete)) {
                            del.mutate(s.id)
                          }
                        }}
                        loading={del.isPending && del.variables === s.id}
                      >
                        <Trash2 className="size-3.5" strokeWidth={2} />
                        {t.admin.servers.deleteCta}
                      </DestructiveButton>
                    </div>
                  </li>
                )
              })}
            </ul>
          )}
        </>
      )}

      <AnimatePresence>
        {mode.kind === "form" && (
          <motion.div
            key="server-form"
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 8 }}
            transition={{ duration: 0.2 }}
          >
            <ServerForm
              server={mode.server}
              onCancel={() => setMode({ kind: "list" })}
              onSaved={() => setMode({ kind: "list" })}
            />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

function ServerForm({
  server,
  onCancel,
  onSaved,
}: {
  server: ServerType | null
  onCancel: () => void
  onSaved: () => void
}) {
  const { t } = useLocale()
  const create = useAdminServerCreate()
  const update = useAdminServerUpdate()

  const [id, setId] = useState(server?.id ?? "")
  const [country, setCountry] = useState(server?.country ?? "")
  const [city, setCity] = useState(server?.city ?? "")
  const [name, setName] = useState(server?.name ?? "")
  const [host, setHost] = useState("")
  const [port, setPort] = useState("")
  const [isActive, setIsActive] = useState(server?.is_active !== false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const input: AdminServerInput = {
      id: id.trim() || undefined,
      country: country.trim().toUpperCase(),
      city: city.trim() || undefined,
      name: name.trim() || undefined,
      host: host.trim() || undefined,
      port: port ? Number(port) : undefined,
      is_active: isActive,
    }
    try {
      if (server) {
        await update.mutateAsync({ id: server.id, input })
      } else {
        await create.mutateAsync(input)
      }
      onSaved()
    } catch {
      /* MutationFeedback will show */
    }
  }

  const busy = create.isPending || update.isPending
  const mutationState = server ? update : create

  return (
    <form
      onSubmit={handleSubmit}
      className="bg-card border-border space-y-3 rounded-2xl border p-5"
    >
      <h3 className="text-foreground text-base font-semibold tracking-tight">
        {server ? t.admin.servers.updateTitle : t.admin.servers.createTitle}
      </h3>

      {!server && (
        <FormField label={t.admin.servers.fields.id}>
          <TextInput
            value={id}
            onChange={(e) => setId(e.target.value)}
            placeholder="de-fra-01"
          />
        </FormField>
      )}
      <div className="grid grid-cols-2 gap-3">
        <FormField label={t.admin.servers.fields.country}>
          <TextInput
            value={country}
            onChange={(e) => setCountry(e.target.value.toUpperCase())}
            placeholder="DE"
            maxLength={2}
            required
          />
        </FormField>
        <FormField label={t.admin.servers.fields.city}>
          <TextInput
            value={city}
            onChange={(e) => setCity(e.target.value)}
            placeholder="Frankfurt"
          />
        </FormField>
      </div>
      <FormField label={t.admin.servers.fields.name}>
        <TextInput
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </FormField>
      <div className="grid grid-cols-[1fr_120px] gap-3">
        <FormField label={t.admin.servers.fields.host}>
          <TextInput
            value={host}
            onChange={(e) => setHost(e.target.value)}
            placeholder="vpn.example.com"
          />
        </FormField>
        <FormField label={t.admin.servers.fields.port}>
          <TextInput
            inputMode="numeric"
            value={port}
            onChange={(e) => setPort(e.target.value)}
            placeholder="443"
          />
        </FormField>
      </div>

      <div className="border-border/60 bg-background flex items-center justify-between rounded-xl border px-3.5 py-3">
        <span className="text-foreground text-sm font-medium">
          {t.admin.servers.fields.isActive}
        </span>
        <Toggle checked={isActive} onChange={setIsActive} />
      </div>

      <MutationFeedback state={mutationState} />

      <div className="grid grid-cols-2 gap-2 pt-2">
        <GhostButton type="button" onClick={onCancel} disabled={busy}>
          {t.admin.servers.cancelCta}
        </GhostButton>
        <PrimaryButton type="submit" loading={busy}>
          {t.admin.servers.saveCta}
        </PrimaryButton>
      </div>
    </form>
  )
}
