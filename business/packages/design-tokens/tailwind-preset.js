/**
 * The Tailwind half of the visual contract.
 *
 * Every scale here resolves to a `tokens.css` variable, so a utility class and
 * a hand-written `var(--ls-*)` rule can never disagree. Adding a literal colour,
 * radius, shadow or font to this file would recreate the second palette that
 * `tokens.css` exists to prevent: extend the token instead.
 *
 * Colours read the `*-rgb` channel companions so Tailwind can inject its own
 * `<alpha-value>`, which is what makes `bg-primary/10` or `border-line/40` work.
 */

const channel = (token) => `rgb(var(--ls-${token}-rgb) / <alpha-value>)`

/**
 * Semantic names come first; the Stripe-flavoured aliases (`indigo`, `ink`,
 * `surface`, `line`) follow so a console page can be written in either
 * vocabulary without a second set of values behind it.
 */
const colors = {
  transparent: 'transparent',
  current: 'currentColor',

  background: channel('canvas'),
  foreground: channel('ink'),
  card: { DEFAULT: channel('surface'), foreground: channel('ink') },
  popover: { DEFAULT: channel('surface'), foreground: channel('ink') },
  primary: {
    DEFAULT: channel('primary'),
    foreground: channel('on-primary'),
    hover: channel('primary-hover'),
    active: channel('primary-active'),
    soft: channel('primary-soft'),
    line: channel('primary-line'),
  },
  secondary: { DEFAULT: channel('surface-subtle'), foreground: channel('ink') },
  muted: { DEFAULT: channel('surface-subtle'), foreground: channel('muted') },
  accent: { DEFAULT: channel('surface-hover'), foreground: channel('ink') },
  destructive: { DEFAULT: channel('danger'), foreground: channel('on-primary') },
  border: channel('border'),
  input: channel('border'),
  ring: channel('primary'),

  indigo: {
    DEFAULT: channel('primary'),
    50: channel('primary-soft'),
    100: channel('primary-line'),
    600: channel('primary-hover'),
    700: channel('primary-active'),
  },
  ink: {
    DEFAULT: channel('ink'),
    soft: channel('ink-soft'),
    muted: channel('muted'),
    subtle: channel('subtle'),
  },
  canvas: channel('canvas'),
  surface: {
    DEFAULT: channel('surface'),
    2: channel('surface-subtle'),
    3: channel('surface-sunken'),
    hover: channel('surface-hover'),
  },
  line: {
    DEFAULT: channel('border'),
    2: channel('border-soft'),
    strong: channel('border-strong'),
  },

  success: { DEFAULT: channel('success'), bg: channel('success-bg'), line: channel('success-border') },
  warning: { DEFAULT: channel('warning'), bg: channel('warning-bg'), line: channel('warning-border') },
  danger: { DEFAULT: channel('danger'), bg: channel('danger-bg'), line: channel('danger-border') },
  info: { DEFAULT: channel('info'), bg: channel('info-bg'), line: channel('info-border') },
}

/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class'],
  theme: {
    extend: {
      colors,
      borderRadius: {
        sm: 'var(--ls-radius-xs)',
        md: 'var(--ls-radius-sm)',
        lg: 'var(--ls-radius-md)',
        xl: 'var(--ls-radius-lg)',
        '2xl': 'var(--ls-radius-xl)',
        full: 'var(--ls-radius-pill)',
        'r-1': 'var(--ls-radius-xs)',
        'r-2': 'var(--ls-radius-sm)',
        'r-3': 'var(--ls-radius-md)',
        'r-4': 'var(--ls-radius-lg)',
      },
      boxShadow: {
        card: 'var(--ls-shadow-card)',
        card2: 'var(--ls-shadow-raised)',
        raised: 'var(--ls-shadow-raised)',
        pop: 'var(--ls-shadow-pop)',
        focus: 'var(--ls-focus-ring)',
      },
      fontFamily: {
        sans: 'var(--ls-font-sans)',
        mono: 'var(--ls-font-mono)',
      },
      fontSize: {
        xs2: ['11px', { lineHeight: '1.4' }],
        sm2: ['12.5px', { lineHeight: '1.4' }],
      },
      minHeight: { touch: 'var(--ls-touch)' },
      keyframes: {
        'ls-fade-in': { from: { opacity: '0' }, to: { opacity: '1' } },
        'ls-zoom-in': {
          from: { opacity: '0', transform: 'scale(.98) translateY(8px)' },
          to: { opacity: '1', transform: 'scale(1) translateY(0)' },
        },
      },
      animation: {
        'ls-fade-in': 'ls-fade-in 200ms ease-out',
        'ls-zoom-in': 'ls-zoom-in 200ms cubic-bezier(.2,.8,.2,1)',
      },
    },
  },
  plugins: [],
}
