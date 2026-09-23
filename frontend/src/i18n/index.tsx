import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';

export type Locale = 'ru' | 'kk' | 'en';
const storageKey = 'careerquest.locale';
let currentLocale: Locale = 'ru';
export function getLocale(): Locale {
  try {
    const saved = localStorage.getItem(storageKey);
    if (saved === 'ru' || saved === 'kk' || saved === 'en') currentLocale = saved;
  } catch {
    /* Private browsing may disable storage. */
  }
  return currentLocale;
}
export const localeTag: Record<Locale, string> = { ru: 'ru-RU', kk: 'kk-KZ', en: 'en-GB' };
type Translate = (ru: string, kk: string, en: string) => string;
const Context = createContext<{
  locale: Locale;
  setLocale: (locale: Locale) => void;
  tr: Translate;
}>({ locale: 'ru', setLocale: () => {}, tr: (ru) => ru });
export function LanguageProvider({ children }: { children: ReactNode }) {
  const [locale, update] = useState<Locale>(getLocale);
  const setLocale = (next: Locale) => {
    currentLocale = next;
    update(next);
    try {
      localStorage.setItem(storageKey, next);
    } catch {
      /* Keep the preference for this session. */
    }
  };
  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);
  return (
    <Context.Provider value={{ locale, setLocale, tr: (ru, kk, en) => ({ ru, kk, en })[locale] }}>
      {children}
    </Context.Provider>
  );
}
export const useI18n = () => useContext(Context);
export function LanguageSwitch() {
  const { locale, setLocale, tr } = useI18n();
  return (
    <div
      className="language-switch"
      role="group"
      aria-label={tr('Язык интерфейса', 'Интерфейс тілі', 'Interface language')}
    >
      {(
        [
          ['ru', 'RU'],
          ['kk', 'KZ'],
          ['en', 'ENG'],
        ] as const
      ).map(([id, label]) => (
        <button
          key={id}
          type="button"
          lang={id}
          aria-pressed={locale === id}
          onClick={() => setLocale(id)}
        >
          {label}
        </button>
      ))}
    </div>
  );
}
export function formatDate(value: string, locale: Locale = 'ru', time = false) {
  if (!value) return '—';
  const date = new Date(value.length === 10 ? `${value}T12:00:00Z` : value);
  if (Number.isNaN(date.getTime())) return value;
  if (locale === 'kk') {
    // Some embedded browsers have incomplete Kazakh CLDR month data ("M10").
    const months = [
      'қаңтар',
      'ақпан',
      'наурыз',
      'сәуір',
      'мамыр',
      'маусым',
      'шілде',
      'тамыз',
      'қыркүйек',
      'қазан',
      'қараша',
      'желтоқсан',
    ];
    const parts = Object.fromEntries(
      new Intl.DateTimeFormat('en-GB', {
        day: 'numeric',
        month: 'numeric',
        year: 'numeric',
        timeZone: 'Asia/Qyzylorda',
        hourCycle: 'h23',
        ...(time ? ({ hour: '2-digit', minute: '2-digit' } as const) : {}),
      })
        .formatToParts(date)
        .map((part) => [part.type, part.value]),
    );
    return `${parts.day} ${months[Number(parts.month) - 1]} ${parts.year}${time ? `, ${parts.hour}:${parts.minute}` : ''}`;
  }
  return new Intl.DateTimeFormat(localeTag[locale], {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    ...(time ? ({ hour: '2-digit', minute: '2-digit' } as const) : {}),
    timeZone: 'Asia/Qyzylorda',
  }).format(date);
}
export function useLabels() {
  const { tr } = useI18n();
  const statuses: Record<string, string> = {
    available: tr('Можно начать', 'Бастауға болады', 'Ready to start'),
    recommended: tr('Ваш следующий шаг', 'Келесі қадамыңыз', 'Your next step'),
    in_progress: tr('В работе', 'Орындалуда', 'In progress'),
    pending: tr('На проверке', 'Тексерілуде', 'Awaiting review'),
    pending_review: tr('На проверке', 'Тексерілуде', 'Awaiting review'),
    changes_requested: tr('Нужна доработка', 'Толықтыру қажет', 'Changes requested'),
    completed: tr('Выполнено', 'Орындалды', 'Completed'),
    approved: tr('Подтверждено', 'Расталды', 'Approved'),
    locked: tr('Пока недоступно', 'Әзірге қолжетімсіз', 'Not available yet'),
    revoked: tr('Подтверждение отменено', 'Растау жойылды', 'Approval revoked'),
    dropped: tr('Прервано', 'Тоқтатылды', 'Dropped'),
    no_show: tr('Пропуск', 'Қатыспады', 'Missed'),
    declined: tr('Отказ', 'Бас тартты', 'Declined'),
    overdue: tr('Просрочено', 'Мерзімі өтті', 'Overdue'),
  };
  const categories: Record<string, string> = {
    engineering: tr('Инженерия', 'Инженерия', 'Engineering'),
    frontend: tr('Фронтенд', 'Фронтенд', 'Frontend'),
    data: tr('Данные', 'Деректер', 'Data'),
    quality: tr('Качество', 'Сапа', 'Quality'),
    product: tr('Продукт', 'Өнім', 'Product'),
    sales: tr('Продажи', 'Сату', 'Sales'),
    hr: tr('Работа с людьми', 'Адамдармен жұмыс', 'People'),
    support: tr('Поддержка', 'Қолдау', 'Support'),
    communication: tr('Общение', 'Қарым-қатынас', 'Communication'),
    collaboration: tr('Сотрудничество', 'Ынтымақтастық', 'Collaboration'),
    leadership: tr('Лидерство', 'Көшбасшылық', 'Leadership'),
    thinking: tr('Мышление', 'Ойлау', 'Thinking'),
    personal_effectiveness: tr('Личная эффективность', 'Жеке тиімділік', 'Personal effectiveness'),
  };
  const formats: Record<string, string> = {
    online: tr('Онлайн', 'Онлайн', 'Online'),
    offline: tr('Очно', 'Офлайн', 'In person'),
    self_paced: tr('В своём темпе', 'Өз қарқыныңызбен', 'Self-paced'),
  };
  return {
    status: (value: string) => statuses[value] || value,
    category: (value: string) => categories[value] || value,
    format: (value: string) => formats[value] || value,
  };
}
