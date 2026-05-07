import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react"

type BackHandler = () => void

type BackButtonContextValue = {
  back: BackHandler | null
  setBack: (handler: BackHandler | null) => void
}

const BackButtonContext = createContext<BackButtonContextValue | null>(null)

export function BackButtonProvider({ children }: { children: ReactNode }) {
  const [back, setBackState] = useState<BackHandler | null>(null)
  const setBack = useCallback((handler: BackHandler | null) => {
    setBackState(() => handler)
  }, [])
  return (
    <BackButtonContext.Provider value={{ back, setBack }}>
      {children}
    </BackButtonContext.Provider>
  )
}

export function useBackButton() {
  const ctx = useContext(BackButtonContext)
  if (!ctx) throw new Error("useBackButton must be used within BackButtonProvider")
  return ctx
}

export function useRegisterBack(handler: BackHandler | null) {
  const { setBack } = useBackButton()
  const handlerRef = useRef<BackHandler | null>(handler)
  handlerRef.current = handler

  const proxy = useRef<BackHandler>(() => {
    handlerRef.current?.()
  }).current

  const enabled = handler !== null
  useEffect(() => {
    if (!enabled) {
      setBack(null)
      return
    }
    setBack(proxy)
    return () => setBack(null)
  }, [enabled, proxy, setBack])
}
