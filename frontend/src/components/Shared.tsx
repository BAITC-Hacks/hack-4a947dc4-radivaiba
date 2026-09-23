import { AlertCircle, ArrowUpRight, Compass, LoaderCircle } from 'lucide-react';
export function Brand() {
  return (
    <div className="brand">
      <span className="brand-icon">
        <Compass size={23} strokeWidth={1.8} />
      </span>
      <span>
        career<span className="brand-light">quest</span>
        <small>РАСТИ В СВОЁМ НАПРАВЛЕНИИ</small>
      </span>
    </div>
  );
}
export function Loading({ text = 'Загружаем данные…' }: { text?: string }) {
  return (
    <div className="loading" role="status">
      <LoaderCircle className="spin" size={22} />
      {text}
    </div>
  );
}
export function ErrorMessage({ message, retry }: { message: string; retry?: () => void }) {
  return (
    <div className="error" role="alert">
      <AlertCircle size={18} />
      <span>{message}</span>
      {retry && (
        <button className="text-button" onClick={retry}>
          Повторить
        </button>
      )}
    </div>
  );
}
export function Empty({ title, text }: { title: string; text: string }) {
  return (
    <div className="empty">
      <Compass size={32} />
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
      <strong>
        {value}
        <ArrowUpRight size={20} />
      </strong>
      <small>{note}</small>
    </div>
  );
}
export const dateLabel = (date: string) =>
  new Date(`${date}T12:00:00`).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
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
