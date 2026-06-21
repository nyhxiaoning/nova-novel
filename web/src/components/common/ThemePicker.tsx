import { useState, useRef, useEffect, useCallback } from 'react'
import { useTheme } from 'next-themes'
import { useTranslation } from 'react-i18next'
import { Check } from 'lucide-react'
import { updateUserSettings } from '@/features/settings/api'

type ThemeValue = 'dark' | 'light' | 'book-yellow' | 'beige' | 'gray-white'

interface ThemeOption {
  value: ThemeValue
  labelKey: string
  color: string
  ring: string
}

const THEME_OPTIONS: ThemeOption[] = [
  { value: 'dark', labelKey: 'theme.dark', color: '#1a1a1a', ring: '#525252' },
  { value: 'light', labelKey: 'theme.light', color: '#edf1f5', ring: '#cbd4df' },
  { value: 'book-yellow', labelKey: 'theme.bookYellow', color: '#f5edd6', ring: '#d4c9a8' },
  { value: 'beige', labelKey: 'theme.beige', color: '#f2ece3', ring: '#cfc5b5' },
  { value: 'gray-white', labelKey: 'theme.grayWhite', color: '#f0f0f2', ring: '#d0d0d5' },
]

export function ThemePicker() {
  const { t } = useTranslation()
  const { theme, setTheme } = useTheme()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open])

  const current = (theme as ThemeValue) || 'dark'
  const currentOption = THEME_OPTIONS.find((o) => o.value === current) || THEME_OPTIONS[0]

  const handleSelect = useCallback(async (value: ThemeValue) => {
    setTheme(value)
    setOpen(false)
    try {
      await updateUserSettings({ theme: value })
      window.dispatchEvent(new CustomEvent('nova:settings-updated'))
    } catch {
      // ignore persist failure; localStorage already applied
    }
  }, [setTheme])

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="nova-nav-item flex items-center gap-1.5 rounded-[var(--nova-radius)] px-2 py-1 text-[var(--nova-text-faint)] hover:text-[var(--nova-text-muted)]"
        title={t('theme.label')}
      >
        <span
          className="h-2.5 w-2.5 rounded-full shadow-[inset_0_0_0_1px_rgba(0,0,0,0.1)]"
          style={{ background: currentOption.color }}
        />
      </button>

      {open && (
        <div className="absolute right-0 top-full z-50 mt-1 w-40 rounded-[var(--nova-radius)] border border-[var(--nova-border)] bg-[var(--nova-menu-bg)] p-1 shadow-lg backdrop-blur-sm">
          {THEME_OPTIONS.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => handleSelect(option.value)}
              className="flex w-full items-center gap-2 rounded-[6px] px-2 py-1.5 text-left text-xs transition-colors hover:bg-[var(--nova-menu-item-hover-bg)]"
            >
              <span
                className="h-3 w-3 shrink-0 rounded-full shadow-[inset_0_0_0_1px_rgba(0,0,0,0.1)]"
                style={{ background: option.color }}
              />
              <span className="flex-1 text-[var(--nova-text)]">{t(option.labelKey)}</span>
              {current === option.value && <Check className="h-3 w-3 shrink-0 text-[var(--nova-accent-blue)]" />}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
