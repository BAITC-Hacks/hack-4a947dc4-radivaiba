import { useEffect, useRef, type ReactNode } from 'react';
import { AlertCircle, Leaf, LoaderCircle, RotateCcw, X } from 'lucide-react';
import { formatDate, useI18n, useLabels } from '../i18n';
export function Brand() {
  const { tr } = useI18n();
  return (
    <a href="#/" className="brand" aria-label="Career Quest">
      <span className="brand-icon">
        <Leaf size={24} />
      </span>
      <span>
        career<span className="brand-light">quest</span>
        <small>
          {tr('Расти в своём направлении', 'Өз бағытыңызда өсіңіз', 'Grow in your own direction')}
        </small>
      </span>
    </a>
  );
}
export function Loading({ text }: { text?: string }) {
  const { tr } = useI18n();
  return (
    <div className="loading" role="status">
      <LoaderCircle className="spin" size={22} />
      {text || tr('Загружаем…', 'Жүктелуде…', 'Loading…')}
    </div>
  );
}
export function ErrorMessage({ message, retry }: { message: string; retry?: () => void }) {
  const { tr } = useI18n();
  return (
    <div className="error" role="alert">
      <AlertCircle size={20} />
      <span>{message}</span>
      {retry && (
        <button className="text-button" onClick={retry}>
          <RotateCcw size={15} />
          {tr('Повторить', 'Қайталау', 'Retry')}
        </button>
      )}
    </div>
  );
}
export function Empty({ title, text }: { title: string; text: string }) {
  return (
    <div className="empty">
      <Leaf size={30} />
      <h3>{title}</h3>
      <p>{text}</p>
    </div>
  );
}
export function Stat({
  label,
  value,
  note,
}: {
  label: string;
  value: string | number;
  note: string;
}) {
  return (
    <div className="stat">
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{note}</small>
    </div>
  );
}
export function Status({ value }: { value: string }) {
  const { status } = useLabels();
  return (
    <span className={`status status-${value}`}>
      <i />
      {status(value)}
    </span>
  );
}
export function Modal({
  title,
  children,
  onClose,
  wide = false,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  wide?: boolean;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const { tr } = useI18n();
  useEffect(() => {
    const dialog = ref.current;
    dialog?.showModal();
    const previous = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      dialog?.close();
      document.body.style.overflow = previous;
    };
  }, []);
  return (
    <dialog
      ref={ref}
      className={`dialog ${wide ? 'dialog-wide' : ''}`}
      aria-labelledby="dialog-title"
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div className="dialog-content">
        <header className="dialog-header">
          <h2 id="dialog-title">{title}</h2>
          <button
            className="icon-button"
            autoFocus
            onClick={onClose}
            aria-label={tr('Закрыть', 'Жабу', 'Close')}
          >
            <X size={21} />
          </button>
        </header>
        {children}
      </div>
    </dialog>
  );
}
export const dateLabel = formatDate;
export const statusLabels: Record<string, string> = {
  completed: 'Завершено',
  in_progress: 'В процессе',
  dropped: 'Прервано',
  no_show: 'Пропуск',
  declined: 'Отказ',
  overdue: 'Просрочено',
};
export const typeLabels: Record<string, string> = {
  course: 'Курс',
  workshop: 'Воркшоп',
  mentoring: 'Менторство',
  certification: 'Сертификация',
  meetup: 'Встреча',
  compliance: 'Обязательное',
  onboarding: 'Онбординг',
};
export const formatLabels: Record<string, string> = {
  online: 'Онлайн',
  offline: 'Очно',
  self_paced: 'В своём темпе',
};
