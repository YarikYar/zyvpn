export type TgUser = {
  firstName?: string
  lastName?: string
  username?: string
  photoUrl?: string
}

export const MOCK_USER: TgUser = {
  firstName: "Alex",
  lastName: "Polanski",
  username: "darkf",
  photoUrl: "",
}

export function getInitials(user: TgUser): string {
  const first = user.firstName?.trim()?.[0] ?? ""
  const last = user.lastName?.trim()?.[0] ?? ""
  const fallback = user.username?.trim()?.[0] ?? ""
  const value = first + last || fallback || "?"
  return value.toUpperCase()
}

export function getDisplayName(user: TgUser): string {
  if (user.username) return `@${user.username}`
  const full = `${user.firstName ?? ""} ${user.lastName ?? ""}`.trim()
  return full || "Guest"
}
