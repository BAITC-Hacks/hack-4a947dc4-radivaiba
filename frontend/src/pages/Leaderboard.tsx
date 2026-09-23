import { useEffect, useState } from 'react';
import { ArrowRight, CalendarDays, Crown, Trophy } from 'lucide-react';
import { api } from '../api/client';
import type { EmployeeSummary, Leaderboard as Board } from '../api/types';
import { useI18n } from '../i18n';
import { Empty, ErrorMessage, Loading } from '../components/Shared';
export default function Leaderboard({ onEmployee }: { onEmployee?: (id: string) => void }) {
  const { tr, locale } = useI18n();
  const [month, setMonth] = useState('');
  const [board, setBoard] = useState<Board | null>(null);
  const [error, setError] = useState('');
  const [retry, setRetry] = useState(0);
  const [employees, setEmployees] = useState<EmployeeSummary[]>([]);
  const [department, setDepartment] = useState('');
  const [role, setRole] = useState('');
  const isHR = !!onEmployee;
  useEffect(() => {
    let active = true;
    if (isHR)
      api
        .employees()
        .then((result) => {
          if (active) setEmployees(result.items || []);
        })
        .catch((err) => {
          if (active) setError(err.message);
        });
    return () => {
      active = false;
    };
  }, [isHR]);
  useEffect(() => {
    const abort = new AbortController();
    setError('');
    api
      .leaderboard(month, abort.signal, isHR ? { department, role } : {})
      .then(setBoard)
      .catch((err) => {
        if (!abort.signal.aborted) setError(err.message);
      });
    return () => abort.abort();
  }, [month, locale, retry, department, role, isHR]);
  return (
    <>
      <header className="page-heading">
        <div>
          <span className="eyebrow">
            <Trophy size={13} />
            {tr('РОСТ, КОТОРЫЙ ВИДНО', 'КӨРІНЕТІН ӨСУ', 'PROGRESS WORTH CELEBRATING')}
          </span>
          <h1>{tr('Лидеры месяца по EXP', 'Айдың EXP көшбасшылары', 'Monthly EXP leaders')}</h1>
          <p>
            {tr(
              'Подтверждённые результаты — общие поводы для радости.',
              'Расталған нәтижелер — бәрімізге ортақ қуаныш.',
              'Approved results give us something to celebrate together.',
            )}
          </p>
        </div>
        {board && (
          <label className="row">
            <CalendarDays size={18} />
            <select
              className="month-select"
              aria-label={tr('Месяц рейтинга', 'Рейтинг айы', 'Leaderboard month')}
              value={month || board.month}
              onChange={(e) => setMonth(e.target.value)}
            >
              {[...new Set([board.month, ...(board.months || [])])]
                .sort()
                .reverse()
                .map((value) => (
                  <option value={value} key={value}>
                    {value}
                  </option>
                ))}
            </select>
          </label>
        )}
      </header>
      <div className="leaderboard-hero">
        <div>
          <span className="eyebrow">{board?.month || 'MONTHLY EXP'}</span>
          <h2>
            {tr(
              'Растём каждый в своём темпе',
              'Әрқайсымыз өз қарқынымызбен өсеміз',
              'Every person grows at their own pace',
            )}
          </h2>
          <p>
            {tr(
              'EXP отражает подтверждённые активности за месяц. Он не заменяет оценку качества работы и не обещает повышение.',
              'EXP айдағы расталған белсенділіктерді көрсетеді. Ол жұмыс сапасын бағалауды алмастырмайды және жоғарылауды уәде етпейді.',
              'EXP reflects approved activities this month. It does not replace a performance review or promise a promotion.',
            )}
          </p>
        </div>
        <span className="trophy">
          <Trophy size={34} />
        </span>
      </div>
      {isHR && (
        <div className="filters">
          <label>
            {tr('Подразделение', 'Бөлім', 'Department')}
            <select value={department} onChange={(event) => setDepartment(event.target.value)}>
              <option value="">
                {tr('Все подразделения', 'Барлық бөлімдер', 'All departments')}
              </option>
              {[...new Set(employees.map((employee) => employee.department))]
                .sort()
                .map((value) => (
                  <option key={value}>{value}</option>
                ))}
            </select>
          </label>
          <label>
            {tr('Роль', 'Рөл', 'Role')}
            <select value={role} onChange={(event) => setRole(event.target.value)}>
              <option value="">{tr('Все роли', 'Барлық рөлдер', 'All roles')}</option>
              {[...new Set(employees.map((employee) => employee.role))].sort().map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
          </label>
          {(department || role) && (
            <button
              className="text-button"
              onClick={() => {
                setDepartment('');
                setRole('');
              }}
            >
              {tr('Сбросить фильтры', 'Сүзгілерді тазалау', 'Clear filters')}
            </button>
          )}
        </div>
      )}
      {error && <ErrorMessage message={error} retry={() => setRetry((v) => v + 1)} />}{' '}
      {!board && !error && <Loading />}
      {board && (
        <section className="panel">
          {board.items?.length ? (
            board.items.map((entry, index) => (
              <div
                className={`leader-row ${entry.rank === 1 ? 'winner' : ''}`}
                key={`${entry.full_name}-${index}`}
              >
                <span className="leader-rank">
                  {entry.rank === 1 ? <Crown size={23} /> : entry.rank}
                </span>
                <span className="mini-avatar">
                  {entry.full_name
                    .split(' ')
                    .slice(0, 2)
                    .map((part) => part[0])
                    .join('')}
                </span>
                <div className="leader-name">
                  <strong>{entry.full_name}</strong>
                  {(entry.department || entry.role) && (
                    <p>{[entry.department, entry.role].filter(Boolean).join(' · ')}</p>
                  )}
                </div>
                <span className="leader-exp">
                  {entry.exp}
                  <small>EXP</small>
                </span>
                {onEmployee && entry.employee_id && (
                  <button
                    className="icon-button"
                    aria-label={`${tr('Открыть профиль', 'Профильді ашу', 'Open profile')} ${entry.full_name}`}
                    onClick={() => onEmployee(entry.employee_id!)}
                  >
                    <ArrowRight size={17} />
                  </button>
                )}
              </div>
            ))
          ) : (
            <Empty
              title={tr(
                'Новый месяц — новые возможности',
                'Жаңа ай — жаңа мүмкіндіктер',
                'A new month, a fresh start',
              )}
              text={tr(
                'Первые подтверждённые результаты появятся здесь. При одинаковом EXP участники разделяют место.',
                'Алғашқы расталған нәтижелер осында пайда болады. EXP тең болса, қатысушылар бір орынды бөліседі.',
                'The first approved results will appear here. People with equal EXP share a rank.',
              )}
            />
          )}
        </section>
      )}
      <p className="privacy-note">
        {tr(
          'Рейтинг виден вошедшим участникам. Материалы заданий и чужие профили остаются закрытыми. При равном EXP — одинаковое место.',
          'Рейтинг кірген қатысушыларға көрінеді. Тапсырма материалдары мен басқа адамдардың профильдері жабық қалады. EXP тең болса, орын да бірдей.',
          'Signed-in participants can see this leaderboard. Evidence and other employees’ private profiles remain protected. Equal EXP means a shared rank.',
        )}
      </p>
    </>
  );
}
